package fs

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/v2/contentapi"
)

type listOperation struct{}

func (o *listOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "list <volume-id> <path>",
		Aliases: []string{"ls"},
		Short:   "List a directory in ls -l format",
		Args:    cobra.ExactArgs(2),
	}
}

func (o *listOperation) Run(ctx cmd.OperationContext) error {
	content, err := open(ctx)
	if err != nil {
		return err
	}
	entries, err := content.ListDir(ctx, ctx.Args[1], 1)
	if err != nil {
		return err
	}
	slices.SortFunc(entries, func(a, b contentapi.Entry) int {
		return strings.Compare(a.Name, b.Name)
	})
	w := tabwriter.NewWriter(ctx.Cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	for _, entry := range entries {
		mode := fs.FileMode(entry.Mode).Perm()
		kind := "-"
		switch entry.Type {
		case contentapi.Directory:
			kind = "d"
		case contentapi.Symlink:
			kind = "l"
		}
		name := entry.Name
		if entry.Type == contentapi.Symlink && entry.Target != nil {
			name += " -> " + *entry.Target
		}
		if _, err := fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\t%s\n",
			kind+mode.String()[1:], entry.Uid, entry.Gid, entry.Size, entry.Mtime.Format("Jan _2 15:04"), name); err != nil {
			return err
		}
	}
	return w.Flush()
}
