package template

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/table"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

// listedTemplate is a display-friendly view of TemplateInfo for table rendering.
type listedTemplate struct {
	TemplateID string    `table_field:"Template ID"`
	Names      string    `table_field:"Name"`
	Visibility string    `table_field:"Access"`
	CPUCount   int       `table_field:"vCPU"`
	MemoryMB   int       `table_field:"RAM (MB)"`
	CreatedAt  time.Time `table_field:"Created"`
}

func toListedTemplate(t sdk.TemplateInfo) listedTemplate {
	return listedTemplate{
		TemplateID: t.TemplateID,
		Names:      strings.Join(t.Names, ", "),
		Visibility: visibility(t.Public),
		CPUCount:   t.CPUCount,
		MemoryMB:   t.MemoryMB,
		CreatedAt:  t.CreatedAt,
	}
}

func visibility(public bool) string {
	if public {
		return "Public"
	}
	return "Private"
}

func newListCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List sandbox templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()
			paginator := client.ListTemplates(ctx)

			var templates []sdk.TemplateInfo
			for paginator.HasNext() {
				items, err := paginator.NextItems(ctx)
				if err != nil {
					return err
				}
				templates = append(templates, items...)
			}

			if format == "json" {
				return json.NewEncoder(os.Stdout).Encode(templates)
			}

			if len(templates) == 0 {
				fmt.Println("No templates found.")
				return nil
			}

			rows := make([]listedTemplate, len(templates))
			for i, t := range templates {
				rows[i] = toListedTemplate(t)
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
