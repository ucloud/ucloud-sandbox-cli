package template

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/pflag"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

// targets is the flag set shared by the commands that act on templates named
// three different ways.
type targets struct {
	path   string
	pick   bool
	yes    bool
	prompt bool
}

// AddFlags registers the flags picking which templates to act on.
func (t *targets) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&t.path, "path", "p", ".", "Project root the local template config is read from")
	fs.BoolVarP(&t.pick, "select", "s", false, "Pick a template from a list instead of naming one")
	fs.BoolVarP(&t.yes, "yes", "y", false, "Don't ask for confirmation")
}

// resolve decides which templates to act on: the ones named on the command
// line, then the one picked from a list, then the one the local config
// records. The local config is returned only when it is what was used, so the
// caller can clean it up after a delete.
func (t *targets) resolve(ctx cmd.OperationContext) ([]string, *LocalConfig, error) {
	if len(ctx.Args) > 0 {
		return ctx.Args, nil, nil
	}

	if t.pick {
		id, err := t.choose(ctx)
		if err != nil {
			return nil, nil, err
		}
		return []string{id}, nil, nil
	}

	cfg, err := loadConfig(t.path)
	if err != nil || cfg.TemplateID == "" {
		return nil, nil, fmt.Errorf(
			"no template given: name one, use --select, or run this where %s records a template_id",
			configFileName)
	}

	return []string{cfg.TemplateID}, cfg, nil
}

// choose lists the team's templates and asks which one to act on.
func (t *targets) choose(ctx cmd.OperationContext) (string, error) {
	templates, err := ctx.Client.Templates().ListV2(ctx)
	if err != nil {
		return "", err
	}

	if len(templates) == 0 {
		return "", fmt.Errorf("no templates available")
	}

	items := make([]string, 0, len(templates))
	for _, tpl := range templates {
		label := tpl.TemplateID
		if len(tpl.Names) > 0 {
			label = fmt.Sprintf("%s (%s)", strings.Join(tpl.Names, ", "), tpl.TemplateID)
		}
		items = append(items, label)
	}

	index, _, err := (&promptui.Select{Label: "Select template", Items: items}).Run()
	if err != nil {
		return "", err
	}

	return templates[index].TemplateID, nil
}

// report prints what is about to happen to the chosen templates.
func report(action string, ids []string) {
	fmt.Printf("\nTemplates to %s:\n", action)
	for _, id := range ids {
		fmt.Printf("  - %s\n", id)
	}
	fmt.Println()
}
