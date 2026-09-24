package list

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

type item struct {
	Name string `json:"name" table_field:"Name"`
}

// paginatorOf returns a paginator that hands back pages in order, using each
// page's index as its cursor.
func paginatorOf(pages [][]item) *transport.Paginator[item] {
	next := 0
	return transport.NewPaginator(func(ctx context.Context, token string) ([]item, string, error) {
		page := pages[next]
		next++
		if next < len(pages) {
			return page, "cursor", nil
		}
		return page, "", nil
	})
}

func items(names ...string) []item {
	out := make([]item, len(names))
	for i, name := range names {
		out[i] = item{Name: name}
	}
	return out
}

func names(in []item) []string {
	out := make([]string, len(in))
	for i, it := range in {
		out[i] = it.Name
	}
	return out
}

func TestFromSlice(t *testing.T) {
	all := items("a", "b", "c", "d", "e")

	cases := []struct {
		name    string
		page    int
		limit   int
		want    []string
		hasMore bool
	}{
		{name: "no limit lists everything", page: 1, limit: 0, want: []string{"a", "b", "c", "d", "e"}},
		{name: "first page", page: 1, limit: 2, want: []string{"a", "b"}, hasMore: true},
		{name: "middle page", page: 2, limit: 2, want: []string{"c", "d"}, hasMore: true},
		{name: "partial last page", page: 3, limit: 2, want: []string{"e"}},
		{name: "page past the end", page: 4, limit: 2, want: []string{}},
		{name: "limit covers everything", page: 1, limit: 10, want: []string{"a", "b", "c", "d", "e"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page := FromSlice(all, tc.page, tc.limit)
			assert.Equal(t, tc.want, names(page.Items))
			assert.Equal(t, int64(len(all)), page.Total, "slice listings always know their total")
			assert.Equal(t, tc.hasMore, page.HasMore)
		})
	}
}

func TestFromSlice_Empty(t *testing.T) {
	page := FromSlice([]item{}, 1, 10)
	assert.Empty(t, page.Items)
	assert.Zero(t, page.Total)
	assert.False(t, page.HasMore)
}

func TestFromPaginator_NoLimitCollectsEverything(t *testing.T) {
	p := paginatorOf([][]item{items("a", "b"), items("c")})

	page, err := FromPaginator(context.Background(), p, 1, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, names(page.Items))
	assert.Equal(t, int64(3), page.Total)
	assert.False(t, page.HasMore)
}

func TestFromPaginator_SkipsToRequestedPage(t *testing.T) {
	p := paginatorOf([][]item{items("a", "b"), items("c", "d"), items("e", "f")})

	page, err := FromPaginator(context.Background(), p, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "d"}, names(page.Items))
	// The cursor has not been exhausted, so neither count is known yet.
	assert.Equal(t, TotalUnknown, page.Total)
	assert.True(t, page.HasMore)
}

func TestFromPaginator_ExhaustedListingKnowsItsTotal(t *testing.T) {
	p := paginatorOf([][]item{items("a", "b"), items("c")})

	page, err := FromPaginator(context.Background(), p, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"c"}, names(page.Items))
	assert.Equal(t, int64(3), page.Total)
	assert.False(t, page.HasMore)
}

func TestFromPaginator_ShortPagesStillFillTheRequestedPage(t *testing.T) {
	// The endpoint may return fewer items per page than the limit asked for,
	// so a page must be assembled from however many fetches it takes.
	p := paginatorOf([][]item{items("a"), items("b"), items("c"), items("d")})

	page, err := FromPaginator(context.Background(), p, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "d"}, names(page.Items))
	assert.Equal(t, int64(4), page.Total)
	assert.False(t, page.HasMore)
}

func TestFromPaginator_PagePastTheEnd(t *testing.T) {
	p := paginatorOf([][]item{items("a", "b")})

	page, err := FromPaginator(context.Background(), p, 5, 2)
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	assert.Equal(t, int64(2), page.Total)
	assert.False(t, page.HasMore)
}

func TestFromPaginator_FetchError(t *testing.T) {
	wantErr := errors.New("boom")
	p := transport.NewPaginator(func(ctx context.Context, token string) ([]item, string, error) {
		return nil, "", wantErr
	})

	_, err := FromPaginator(context.Background(), p, 1, 2)
	assert.ErrorIs(t, err, wantErr)
}

func TestOptionsValidate(t *testing.T) {
	cases := []struct {
		name    string
		opts    Options
		wantErr string
	}{
		{name: "defaults", opts: Options{Page: 1, Limit: 0, Format: FormatPretty}},
		{name: "json", opts: Options{Page: 2, Limit: 10, Format: FormatJSON}},
		{name: "page below one", opts: Options{Page: 0, Format: FormatPretty}, wantErr: "--page"},
		{name: "negative limit", opts: Options{Page: 1, Limit: -1, Format: FormatPretty}, wantErr: "--limit"},
		{name: "unknown format", opts: Options{Page: 1, Format: "yaml"}, wantErr: "unsupported output format"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.Validate()
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestAddFlagsDefaults(t *testing.T) {
	var opts Options
	cmd := &cobra.Command{Use: "list"}
	opts.AddFlags(cmd)

	require.NoError(t, cmd.ParseFlags(nil))
	assert.Equal(t, Options{Page: 1, Limit: 0, Format: FormatPretty}, opts)
	require.NoError(t, opts.Validate())

	require.NoError(t, cmd.ParseFlags([]string{"-p", "3", "-l", "20", "-f", "json"}))
	assert.Equal(t, Options{Page: 3, Limit: 20, Format: FormatJSON}, opts)
}

func TestRender_JSONIsIndented(t *testing.T) {
	var buf bytes.Buffer
	page := Page[item]{Items: items("a"), Total: 1}

	require.NoError(t, Render(&buf, page, Options{Page: 1, Format: FormatJSON}, toRow, "none"))
	assert.Equal(t, "[\n  {\n    \"name\": \"a\"\n  }\n]\n", buf.String())
}

func TestRender_JSONEmptyIsArrayNotNull(t *testing.T) {
	var buf bytes.Buffer

	require.NoError(t, Render(&buf, Page[item]{}, Options{Page: 1, Format: FormatJSON}, toRow, "none"))
	assert.Equal(t, "[]\n", buf.String())
}

func TestRender_TableEmptyMessage(t *testing.T) {
	var buf bytes.Buffer

	require.NoError(t, Render(&buf, Page[item]{}, Options{Page: 1, Format: FormatPretty}, toRow, "No items found."))
	assert.Equal(t, "No items found.\n", buf.String())
}

func TestRender_Table(t *testing.T) {
	var buf bytes.Buffer
	page := Page[item]{Items: items("a", "b"), Total: 5, HasMore: true}

	require.NoError(t, Render(&buf, page, Options{Page: 1, Limit: 2, Format: FormatPretty}, toRow, "none"))
	out := buf.String()
	assert.True(t, strings.HasPrefix(out, "Page: 1/3, Total: 5\n"), "header line, got: %q", strings.SplitN(out, "\n", 2)[0])
	assert.Contains(t, out, "Name")
	assert.Contains(t, out, "| a")
	assert.True(t, strings.HasSuffix(out, "Use --page/--limit to see more.\n"), "paging hint, got: %q", out)
}

func TestRender_TableUnknownTotal(t *testing.T) {
	var buf bytes.Buffer
	page := Page[item]{Items: items("a"), Total: TotalUnknown, HasMore: true}

	require.NoError(t, Render(&buf, page, Options{Page: 2, Limit: 1, Format: FormatPretty}, toRow, "none"))
	assert.True(t, strings.HasPrefix(buf.String(), "Page: 2/?, Total: ?\n"))
}

func toRow(i item) item { return i }
