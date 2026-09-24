package template

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/template"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/template/build"
)

type initOperation struct {
	path string
	from string

	cpuCount *int32
	memoryMB *int32
}

func (o *initOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "init [name]",
		Aliases: []string{"it"},
		Short:   "Scaffold a new template directory",
		Args:    cobra.MaximumNArgs(1),
	}

	c.Flags().StringVarP(&o.path, "path", "p", ".", "Project root the template directory is created in")
	c.Flags().StringVarP(&o.from, "from", "", "",
		"Base image the Dockerfile starts from (defaults to the platform's base image for your region)")

	flags.NullableInt32VarP(c.Flags(), &o.cpuCount, "cpu-count", "", "CPU cores to record in the template config")
	flags.NullableInt32VarP(c.Flags(), &o.memoryMB, "memory-mb", "",
		"Memory to record in the template config, in MiB (must be even)")

	return c
}

func (o *initOperation) Run(ctx cmd.OperationContext) error {
	name, err := o.name(ctx)
	if err != nil {
		return err
	}

	if err := template.ValidateName(name); err != nil {
		return err
	}

	if o.memoryMB != nil && *o.memoryMB%2 != 0 {
		return fmt.Errorf("memory must be an even number, got %d", *o.memoryMB)
	}

	// The base image depends on the region: the one served inside mainland
	// China is slow to pull from anywhere else, and the other way round.
	from := o.from
	if from == "" {
		from = build.GetBaseImage(ctx.Client.Region())
	}

	dir := filepath.Join(o.path, name)

	// Refusing rather than merging: a directory that is already there may hold
	// a template whose Dockerfile this would overwrite.
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("directory %q already exists", dir)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	cfg := &LocalConfig{TemplateName: name, Dockerfile: defaultDockerfileName}
	if o.cpuCount != nil {
		cfg.CPUCount = *o.cpuCount
	}
	if o.memoryMB != nil {
		cfg.MemoryMB = *o.memoryMB
	}

	if err := saveConfig(dir, cfg); err != nil {
		return err
	}

	dockerfile := fmt.Sprintf("FROM %s\n\nRUN echo \"Hello from %s!\"\n\n# Add your customizations here\n", from, name)
	if err := os.WriteFile(filepath.Join(dir, defaultDockerfileName), []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("write Dockerfile: %w", err)
	}

	fmt.Printf("\nTemplate initialized in %s\n", dir)
	fmt.Printf("\nTo build it:\n")
	fmt.Printf("   ucloud-sandbox-cli template build %s -p %s\n\n", name, o.path)

	return nil
}

// name reads the template's name off the command line, asking for one when it
// is not there.
func (o *initOperation) name(ctx cmd.OperationContext) (string, error) {
	if len(ctx.Args) > 0 {
		return ctx.Args[0], nil
	}

	return prompt.AskTemplateName("my-template")
}
