package template

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
	"github.com/ucloud/ucloud-sandbox-cli/internal/template"
)

func newInitCmd() *cobra.Command {
	var path string
	var name string
	var cpuCount int
	var memoryMB int
	var from string

	cmd := &cobra.Command{
		Use:     "init [name]",
		Aliases: []string{"it"},
		Short:   "Initialize a new sandbox template",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			// Get name
			if len(args) > 0 {
				name = args[0]
			} else if name == "" {
				name, err = prompt.AskTemplateName("my-template")
				if err != nil {
					return err
				}
			}

			if err := template.ValidateName(name); err != nil {
				return err
			}

			// Validate memory
			if memoryMB%2 != 0 {
				return fmt.Errorf("memory must be an even number")
			}

			// Create directory
			templateDir := filepath.Join(path, name)
			if _, err := os.Stat(templateDir); err == nil {
				return fmt.Errorf("directory '%s' already exists", name)
			}

			if err := os.MkdirAll(templateDir, 0755); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}

			// Write config
			cfg := &LocalConfig{
				TemplateName: name,
				CPUCount:     cpuCount,
				MemoryMB:     memoryMB,
				Dockerfile:   "template.dockerfile",
			}
			if err := saveConfig(templateDir, cfg); err != nil {
				return err
			}

			// Write Dockerfile
			dockerfileContent := fmt.Sprintf(`FROM %s

RUN echo "Hello from %s!"

# Add your customizations here
`, from, name)

			dockerfilePath := filepath.Join(templateDir, "template.dockerfile")
			if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
				return fmt.Errorf("write Dockerfile: %w", err)
			}

			// Print success
			fmt.Printf("\n🎉 Template initialized successfully!\n")
			fmt.Printf("\nTemplate created in: ./%s/\n", name)
			fmt.Printf("\n🔨 To build your template:\n")
			fmt.Printf("   ucloud-sandbox-cli template build %s\n\n", name)

			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", ".", "Project root path")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Template name")
	cmd.Flags().IntVar(&cpuCount, "cpu", 2, "CPU count")
	cmd.Flags().IntVar(&memoryMB, "memory", 1024, "Memory in MB (must be even)")
	cmd.Flags().StringVar(&from, "from", "base", "Base image or template")
	return cmd
}
