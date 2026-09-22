package template

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
	"github.com/ucloud/ucloud-sandbox-cli/internal/table"
)

type listedTemplateTag struct {
	Tag       string    `table_field:"Tag"`
	BuildID   string    `table_field:"Build ID"`
	CreatedAt time.Time `table_field:"Created"`
}

func newTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tag",
		Aliases: []string{"tags"},
		Short:   "Manage template tags",
	}
	cmd.AddCommand(newTagAssignCmd())
	cmd.AddCommand(newTagRemoveCmd())
	cmd.AddCommand(newTagListCmd())
	return cmd
}

func newTagAssignCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "assign <template:source-tag> <tag> [tag ...]",
		Aliases: []string{"add"},
		Short:   "Assign tags to the referenced template build",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("template source is required (for example: my-template:default)")
			}
			if len(args) == 1 {
				return fmt.Errorf("at least one tag is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			assigned, err := client.Templates().AssignTags(cmd.Context(), args[0], args[1:])
			if err != nil {
				return err
			}
			fmt.Printf("Assigned tags %s to build %s.\n", strings.Join(assigned.Tags, ", "), assigned.BuildID)
			return nil
		},
	}
}

func newTagRemoveCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:     "remove <template> <tag> [tag ...]",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove tags from a template",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("template is required")
			}
			if len(args) == 1 {
				return fmt.Errorf("at least one tag is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				confirmed, err := prompt.Confirm(fmt.Sprintf("Remove tags %s from template %s?", strings.Join(args[1:], ", "), args[0]))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Println("Canceled.")
					return nil
				}
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}
			if err := client.Templates().DeleteTags(cmd.Context(), args[0], args[1:]); err != nil {
				return err
			}
			fmt.Printf("Removed tags %s from template %s.\n", strings.Join(args[1:], ", "), args[0])
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newTagListCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:     "list <template>",
		Aliases: []string{"ls"},
		Short:   "List template tags",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("template is required")
			}
			if len(args) > 1 {
				return fmt.Errorf("only one template can be specified")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			tags, err := client.Templates().ListTags(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if format == "json" {
				return json.NewEncoder(os.Stdout).Encode(tags)
			}
			if len(tags) == 0 {
				fmt.Println("No tags found.")
				return nil
			}

			rows := make([]listedTemplateTag, len(tags))
			for i, tag := range tags {
				rows[i] = listedTemplateTag(tag)
			}
			out, err := table.Render(rows, 1, 0, int64(len(rows)))
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "pretty", "Output format (pretty, json)")
	return cmd
}
