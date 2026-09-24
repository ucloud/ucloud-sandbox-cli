package fs

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/filesystem"
)

type lsOperation struct {
	session

	depth uint32

	list list.Options
}

func (o *lsOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "ls <sandbox-id> [path]",
		Aliases: []string{"list"},
		Short:   "List a directory or file",
		Args:    cobra.RangeArgs(1, 2),
	}

	o.AddFlags(c.Flags())

	c.Flags().Uint32VarP(&o.depth, "depth", "d", 1, "How far below the directory to descend")

	o.list.AddFlags(c)

	return c
}

// listedEntry is a display-friendly view of filesystem.EntryInfo. The path and
// the symlink target are carried in JSON only, so the table keeps the columns
// that fit a terminal.
type listedEntry struct {
	Name          string    `table_field:"Name" json:"name"`
	Path          string    `table_field:"-" json:"path"`
	Type          string    `table_field:"Type" json:"type"`
	Size          int64     `table_field:"Size" table_format:"bytes" json:"size"`
	Permissions   string    `table_field:"Permissions" json:"permissions"`
	Owner         string    `table_field:"Owner" json:"owner"`
	Group         string    `table_field:"Group" json:"group"`
	ModifiedTime  time.Time `table_field:"Modified" json:"modified_time"`
	SymlinkTarget string    `table_field:"-" json:"symlink_target,omitempty"`
}

func toListedEntry(e *filesystem.EntryInfo) listedEntry {
	row := listedEntry{
		Name:          e.GetName(),
		Path:          e.GetPath(),
		Type:          entryType(e.GetType()),
		Size:          e.GetSize(),
		Permissions:   e.GetPermissions(),
		Owner:         e.GetOwner(),
		Group:         e.GetGroup(),
		SymlinkTarget: e.GetSymlinkTarget(),
	}

	if modified := e.GetModifiedTime(); modified != nil {
		row.ModifiedTime = modified.AsTime()
	}

	return row
}

// entryType names the kind of entry the way a filesystem would, rather than by
// the protobuf enum's own spelling.
func entryType(t filesystem.FileType) string {
	switch t {
	case filesystem.FileType_FILE_TYPE_FILE:
		return "file"
	case filesystem.FileType_FILE_TYPE_DIRECTORY:
		return "dir"
	case filesystem.FileType_FILE_TYPE_SYMLINK:
		return "symlink"
	default:
		return "unknown"
	}
}

func (o *lsOperation) Run(ctx cmd.OperationContext) error {
	if err := o.list.Validate(); err != nil {
		return err
	}

	fs, err := o.open(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	path := "."
	if len(ctx.Args) == 2 {
		path = ctx.Args[1]
	}

	// The path is resolved first, so naming a file lists that file while
	// naming a directory lists what is in it.
	info, err := fs.GetInfo(ctx, path)
	if err != nil {
		return err
	}

	entries := []*filesystem.EntryInfo{info}
	if info.GetType() == filesystem.FileType_FILE_TYPE_DIRECTORY {
		entries, err = fs.List(ctx, path, o.depth)
		if err != nil {
			return err
		}
	}

	// The rows are built before paging so that --format json prints them
	// rather than the protobuf entries behind them.
	rows := make([]listedEntry, len(entries))
	for i, entry := range entries {
		rows[i] = toListedEntry(entry)
	}

	page := list.FromSlice(rows, o.list.Page, o.list.Limit)

	return list.Render(os.Stdout, page, o.list, identity, "No entries found.")
}

// identity is the row conversion for a page whose items are already rows.
func identity(e listedEntry) listedEntry { return e }
