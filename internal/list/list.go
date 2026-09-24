// Package list renders the resource listings behind the CLI's `ls` commands.
//
// It unifies the two shapes the SDK exposes:
//
//   - Cursor-paginated endpoints (sandboxes, snapshots) hand back a
//     transport.Paginator. The cursor only moves forward and no total is
//     reported, so reaching page N means fetching and discarding the N-1
//     pages before it, and the total stays unknown until the listing is
//     exhausted.
//   - Unpaginated endpoints (templates, volumes) hand back a slice, which is
//     paginated client-side with an always-accurate total.
//
// Both produce a Page, which Render writes as an ASCII table or as indented
// JSON.
package list

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/table"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Output formats accepted by the --format flag.
const (
	FormatPretty = "pretty"
	FormatJSON   = "json"
)

// TotalUnknown marks a page whose total item count the endpoint did not
// report.
const TotalUnknown int64 = -1

// Page is one page of a listing.
type Page[T any] struct {
	Items []T

	// Total is the item count across all pages, or TotalUnknown when a cursor
	// listing has not been walked to its end.
	Total int64

	// HasMore reports whether items exist beyond this page.
	HasMore bool
}

// Options are the paging and output flags shared by every `ls` command.
type Options struct {
	Page   int
	Limit  int
	Format string
}

// AddFlags registers --page, --limit and --format on cmd.
func (o *Options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().IntVarP(&o.Page, "page", "p", 1, "Page number")
	cmd.Flags().IntVarP(&o.Limit, "limit", "l", 0, "Items per page (0 lists everything)")
	cmd.Flags().StringVarP(&o.Format, "format", "f", FormatPretty, "Output format (pretty, json)")
}

// Validate checks the flag values.
func (o *Options) Validate() error {
	if o.Page < 1 {
		return fmt.Errorf("invalid --page %d: must be 1 or greater", o.Page)
	}
	if o.Limit < 0 {
		return fmt.Errorf("invalid --limit %d: must be 0 or greater", o.Limit)
	}
	switch o.Format {
	case FormatPretty, FormatJSON:
		return nil
	default:
		return fmt.Errorf("unsupported output format %q: expected %s or %s", o.Format, FormatPretty, FormatJSON)
	}
}

// FromPaginator walks p far enough to return the requested page. page is
// 1-based; a limit of zero or less collects the whole listing into one page.
//
// Pages are not assumed to hold exactly limit items — the endpoint may return
// fewer — so items are accumulated until enough have arrived or the paginator
// is exhausted.
func FromPaginator[T any](ctx context.Context, p *transport.Paginator[T], page, limit int) (Page[T], error) {
	if limit <= 0 {
		items, err := p.All(ctx)
		if err != nil {
			return Page[T]{}, err
		}
		return Page[T]{Items: items, Total: int64(len(items))}, nil
	}

	want := page * limit
	var all []T
	for p.HasNext() && len(all) < want {
		items, err := p.NextItems(ctx)
		if err != nil {
			return Page[T]{}, err
		}
		all = append(all, items...)
	}

	// The listing is only fully known once the cursor runs out; until then the
	// total is whatever the backend still holds.
	total := TotalUnknown
	hasMore := true
	if !p.HasNext() {
		total = int64(len(all))
		hasMore = len(all) > want
	}

	return Page[T]{Items: slicePage(all, page, limit), Total: total, HasMore: hasMore}, nil
}

// FromSlice paginates an already-fetched slice client-side. page is 1-based; a
// limit of zero or less returns everything as a single page.
func FromSlice[T any](items []T, page, limit int) Page[T] {
	total := int64(len(items))
	if limit <= 0 {
		return Page[T]{Items: items, Total: total}
	}
	return Page[T]{
		Items:   slicePage(items, page, limit),
		Total:   total,
		HasMore: int64(page*limit) < total,
	}
}

// slicePage returns the page-th window of limit items, or nil when the page
// lies past the end.
func slicePage[T any](items []T, page, limit int) []T {
	start := (page - 1) * limit
	if start >= len(items) {
		return nil
	}
	return items[start:min(start+limit, len(items))]
}

// Render writes p to w: indented JSON of the raw items, or an ASCII table of
// the rows toRow builds from them. emptyMessage is printed instead of an empty
// table.
func Render[T, R any](w io.Writer, p Page[T], o Options, toRow func(T) R, emptyMessage string) error {
	if o.Format == FormatJSON {
		return writeJSON(w, p.Items)
	}

	if len(p.Items) == 0 {
		_, err := fmt.Fprintln(w, emptyMessage)
		return err
	}

	rows := make([]R, len(p.Items))
	for i, item := range p.Items {
		rows[i] = toRow(item)
	}
	out, err := table.Render(rows, o.Page, o.Limit, p.Total)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, out); err != nil {
		return err
	}
	if p.HasMore {
		_, err = fmt.Fprintln(w, "Use --page/--limit to see more.")
	}
	return err
}

// writeJSON writes items as indented JSON, rendering an absent slice as "[]"
// rather than "null".
func writeJSON[T any](w io.Writer, items []T) error {
	if items == nil {
		items = []T{}
	}
	encoded, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", encoded)
	return err
}
