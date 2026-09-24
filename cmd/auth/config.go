package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

// fallbackEditor opens the config file when $EDITOR says nothing.
const fallbackEditor = "vim"

// maskedValue stands in for a secret, so the config can be shown without
// putting the secret itself on the screen.
const maskedValue = "****"

type configOperation struct {
	edit bool
}

func (o *configOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "config [editor]",
		Short: "Show or edit the configuration",
		// The editor only makes sense together with --edit.
		Args: cobra.MaximumNArgs(1),
	}

	c.Flags().BoolVarP(&o.edit, "edit", "e", false,
		"Open the config file in an editor ($EDITOR, or "+fallbackEditor+")")

	return c
}

func (o *configOperation) Run(ctx cmd.OperationContext) error {
	if len(ctx.Args) > 0 && !o.edit {
		return fmt.Errorf("an editor can only be given together with --edit")
	}

	if o.edit {
		return editConfig(resolveEditor(ctx.Args))
	}

	// The resolved config is shown, environment overrides included, because
	// that is what the other commands actually run with.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	return writeMaskedConfig(ctx.Cmd.OutOrStdout(), cfg)
}

// resolveEditor picks the editor to open the config file with: the one named
// on the command line, then $EDITOR, then a fallback.
func resolveEditor(args []string) string {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return strings.TrimSpace(args[0])
	}

	if editor := strings.TrimSpace(os.Getenv("EDITOR")); editor != "" {
		return editor
	}

	return fallbackEditor
}

// editConfig opens the config file in editor and reports whether what came
// back still parses.
func editConfig(editor string) error {
	path, err := config.Path()
	if err != nil {
		return err
	}

	// An editor started on a path whose directory does not exist cannot save,
	// so the file is created first. It also gives the user valid JSON to edit
	// rather than an empty buffer.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := config.Save(&config.Config{}); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat config: %w", err)
	}

	// $EDITOR may carry arguments of its own, for example "code --wait".
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("no editor to open %s with", path)
	}

	command := exec.Command(parts[0], append(parts[1:], path)...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("run editor %q: %w", editor, err)
	}

	// Saying so now beats the next command failing on a file the user no
	// longer has in mind.
	if _, err := config.LoadFile(); err != nil {
		return fmt.Errorf("%s was saved but no longer parses: %w", path, err)
	}

	return nil
}

// writeMaskedConfig writes cfg as indented JSON with its secrets masked.
func writeMaskedConfig(w io.Writer, cfg *config.Config) error {
	masked := *cfg
	masked.APIKey = maskSecret(masked.APIKey)

	// The copy above is shallow, so the registry map is still the caller's:
	// masking in place would blank the passwords they are holding.
	if cfg.Registries != nil {
		registries := make(map[string]config.RegistryAuth, len(cfg.Registries))
		for domain, auth := range cfg.Registries {
			auth.Password = maskSecret(auth.Password)
			registries[domain] = auth
		}
		masked.Registries = registries
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(&masked); err != nil {
		return fmt.Errorf("print config: %w", err)
	}

	return nil
}

// maskSecret masks a sensitive config value, leaving an unset one unset so the
// output still shows what is missing.
func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}

	return maskedValue
}
