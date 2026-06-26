package table

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sample struct {
	Name      string `table_field:"-"` // skipped
	Title     string // header = "Title"
	Renamed   string `table_field:"Alias"` // header = "Alias"
	Size      int64  `table_format:"bytes"`
	Active    bool
	Tags      []string
	CreatedAt time.Time `table_field:"Created"`

	unexported string //nolint:unused // verify reflection skips it
}

func TestRender_Basic(t *testing.T) {
	now := time.Now()
	items := []sample{
		{
			Name:      "ignored",
			Title:     "first",
			Renamed:   "alpha",
			Size:      2048,
			Active:    true,
			Tags:      []string{"a", "b"},
			CreatedAt: now.Add(-2 * time.Hour),
		},
		{
			Title:     "second",
			Renamed:   "",
			Size:      0,
			Active:    false,
			Tags:      nil,
			CreatedAt: time.Time{},
		},
	}

	out, err := Render(items, 1, 20, 42)
	require.NoError(t, err)

	// Skipped field must not appear.
	assert.NotContains(t, out, "Name")

	// Renamed column header.
	assert.Contains(t, out, "Alias")
	assert.Contains(t, out, "Created")

	// Pagination header. ceil(42/20) = 3.
	assert.True(t, strings.HasPrefix(out, "Page: 1/3, Total: 42\n"), "header line, got: %q", strings.SplitN(out, "\n", 2)[0])

	// Byte humanization on Size.
	assert.Contains(t, out, "2.0 KiB")

	// Zero string becomes "-".
	assert.Contains(t, out, "| -")

	// Zero time becomes "-".
	assert.Contains(t, strings.Split(out, "\n")[5], " - ")

	// Slice rendered as "N items".
	assert.Contains(t, out, "2 items")

	// Bool stringification.
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "false")
}

func TestRender_EmptySlice(t *testing.T) {
	out, err := Render([]sample{}, 1, 20, 0)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, "Page: 1/0, Total: 0\n"))
	// Should still render the header row and borders.
	assert.Contains(t, out, "Title")
}

func TestRender_PointerElems(t *testing.T) {
	items := []*sample{{Title: "ptr"}}
	out, err := Render(items, 1, 20, 1)
	require.NoError(t, err)
	assert.Contains(t, out, "ptr")
}

func TestRender_Errors(t *testing.T) {
	_, err := Render("not a slice", 1, 20, 0)
	assert.Error(t, err)

	_, err = Render([]int{1, 2}, 1, 20, 2)
	assert.Error(t, err) // element must be a struct
}

func TestRender_TotalPages(t *testing.T) {
	// page-size 0 with non-zero total -> 1 page.
	out, err := Render([]sample{{Title: "x"}}, 1, 0, 5)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, "Page: 1/1, Total: 5\n"))
}

func TestFitToWidth_NoTruncation(t *testing.T) {
	widths := []int{6, 13, 5}
	hdrs := []string{"Remote", "Owner", "Name"}
	body := [][]string{{"github", "fioncat", "peregrine"}}
	gotW, gotH, gotB := fitToWidth(widths, hdrs, body, 200)
	assert.Equal(t, widths, gotW)
	assert.Equal(t, hdrs, gotH)
	assert.Equal(t, body, gotB)
}

func TestFitToWidth_TruncatesBoundaryColumn(t *testing.T) {
	// Full table width = 1 + (6+3) + (13+3) + (9+3) = 38. Force a tighter
	// terminal so the third column has to shrink.
	widths := []int{6, 13, 9}
	hdrs := []string{"Remote", "Owner", "Name"}
	body := [][]string{
		{"github", "fioncat", "peregrine"},
		{"ucloud", "agent-sandbox", "deploy"},
	}
	const termWidth = 32
	gotW, gotH, gotB := fitToWidth(widths, hdrs, body, termWidth)

	assert.Equal(t, 3, len(gotW), "third column should still be present, just shrunk")
	assert.Equal(t, termWidth, tableWidth(gotW), "fitted table must equal terminal width exactly")
	assert.Contains(t, gotH[2], "...", "shrunk header gets an ellipsis")
	for _, row := range gotB {
		assert.LessOrEqual(t, len(row[2]), gotW[2])
	}
}

func TestFitToWidth_DropsColumnsThatCannotShrinkEnough(t *testing.T) {
	// Only the first column fits; the second's minimum (3) plus overhead
	// already exceeds termWidth, so it must be dropped.
	widths := []int{6, 13}
	hdrs := []string{"Remote", "Owner"}
	body := [][]string{{"github", "fioncat"}}
	const termWidth = 11 // boundary col would need width >= 3, total = 10 + 3 + 3 = 16 > 11
	gotW, gotH, gotB := fitToWidth(widths, hdrs, body, termWidth)
	assert.Equal(t, 1, len(gotW))
	assert.Equal(t, []string{"Remote"}, gotH)
	assert.Equal(t, [][]string{{"github"}}, gotB)
}

func TestTruncateCell(t *testing.T) {
	assert.Equal(t, "hello", truncateCell("hello", 10))
	assert.Equal(t, "hello", truncateCell("hello", 5))
	assert.Equal(t, "he...", truncateCell("hello world", 5))
	assert.Equal(t, "...", truncateCell("hello", 3))
}
