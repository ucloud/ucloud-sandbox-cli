package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/table"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

// listedEntry is a display-friendly view of EntryInfo for table rendering.
type listedEntry struct {
	Name         string    `table_field:"Name"`
	Type         string    `table_field:"Type"`
	Size         int64     `table_field:"Size" table_format:"bytes"`
	Permissions  string    `table_field:"Permissions"`
	Owner        string    `table_field:"Owner"`
	Group        string    `table_field:"Group"`
	ModifiedTime time.Time `table_field:"Modified"`
}

func toListedEntry(e sdk.EntryInfo) listedEntry {
	return listedEntry{
		Name:         e.Name,
		Type:         string(e.Type),
		Size:         e.Size,
		Permissions:  e.Permissions,
		Owner:        e.Owner,
		Group:        e.Group,
		ModifiedTime: e.ModifiedTime,
	}
}

func newLsCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:     "ls <sandbox-id> [path]",
		Aliases: []string{"list"},
		Short:   "List a directory or file",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			path := "."
			if len(args) == 2 {
				path = args[1]
			}

			ctx := context.Background()
			sbx, err := client.ConnectSandbox(ctx, args[0])
			if err != nil {
				return err
			}

			// Resolve the path so a file argument lists just that file while a
			// directory argument lists its entries.
			info, err := sbx.Files.GetInfo(ctx, path)
			if err != nil {
				return err
			}

			var entries []sdk.EntryInfo
			if info.Type == sdk.EntryTypeDir {
				entries, err = sbx.Files.List(ctx, path)
				if err != nil {
					return err
				}
			} else {
				entries = []sdk.EntryInfo{*info}
			}

			if format == "json" {
				return json.NewEncoder(os.Stdout).Encode(entries)
			}

			if len(entries) == 0 {
				fmt.Println("No entries found.")
				return nil
			}

			rows := make([]listedEntry, len(entries))
			for i, e := range entries {
				rows[i] = toListedEntry(e)
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
