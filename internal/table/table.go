// Package table renders a slice of structs as a paginated ASCII table.
//
// Columns are derived by reflection from the element struct's exported
// fields. The `table_field` struct tag controls per-column behavior:
//
//   - table_field:"-"        skip this field
//   - (tag absent or empty)  column header is the field name
//   - table_field:"Custom"   column header is "Custom"
//
// The optional `table_format` tag picks a non-default formatter:
//
//   - table_format:"bytes"   render an integer field via humanize.IBytes
//
// time.Time fields are rendered as relative time (e.g. "13s ago").
package table

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"golang.org/x/term"
)

type column struct {
	header     string
	fieldIndex int
	format     formatter
}

// Render renders items as a paginated ASCII table.
//
// items must be a slice; the element type's struct fields determine the
// columns. page is 1-based. total is the total row count across all pages
// (returned by the backend / service), used to compute the total page count
// in the header. The trailing newline is included.
func Render(items any, page, pageSize int, total int64) (string, error) {
	rv := reflect.ValueOf(items)
	if !rv.IsValid() {
		return "", fmt.Errorf("table: items is nil")
	}
	if rv.Kind() != reflect.Slice {
		return "", fmt.Errorf("table: items must be a slice, got %s", rv.Kind())
	}

	elemType := rv.Type().Elem()
	for elemType.Kind() == reflect.Pointer {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return "", fmt.Errorf("table: element type must be a struct, got %s", elemType.Kind())
	}

	cols := buildColumns(elemType)

	// Build the body row by row.
	body := make([][]string, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		row := rv.Index(i)
		for row.Kind() == reflect.Pointer {
			row = row.Elem()
		}
		cells := make([]string, len(cols))
		for j, c := range cols {
			cells[j] = c.format(row.Field(c.fieldIndex))
		}
		body = append(body, cells)
	}

	widths := computeWidths(cols, body)

	var b strings.Builder
	b.WriteString(renderHeader(page, pageSize, total))
	b.WriteByte('\n')
	if len(cols) == 0 {
		return b.String(), nil
	}

	// Fit columns to terminal width by trimming the last column that still
	// fits and dropping the rest. Columns that survive truncation get their
	// content suffixed with "..." so the user can tell.
	hdrs := headers(cols)
	if w := terminalWidth(); w > 0 {
		widths, hdrs, body = fitToWidth(widths, hdrs, body, w)
	}

	writeBorder(&b, widths)
	writeRow(&b, hdrs, widths)
	writeBorder(&b, widths)
	for _, row := range body {
		writeRow(&b, row, widths)
	}
	writeBorder(&b, widths)
	return b.String(), nil
}

// terminalWidth returns the column count of stdout, or 0 if stdout is not a
// terminal (e.g. piped to a file) so the renderer leaves the table untrimmed.
func terminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 0
	}
	return w
}

// tableWidth returns the rendered width of a table with the given column
// widths: "| col1 | col2 | ... |" — each column contributes width+3 (the
// "| " prefix plus a trailing space), plus 1 for the final "|".
func tableWidth(widths []int) int {
	total := 1
	for _, w := range widths {
		total += w + 3
	}
	return total
}

// minTruncatedColWidth is the smallest width we'll shrink a column to before
// dropping it entirely. 3 leaves just enough room for "..." with no content.
const minTruncatedColWidth = 3

// fitToWidth shrinks the table to fit termWidth. It keeps the leading
// columns intact, picks one "boundary" column to shrink and tag with "...",
// and drops all columns after it. If the boundary column would have to
// shrink below minTruncatedColWidth, it is dropped too and the previous
// column becomes the boundary.
func fitToWidth(widths []int, hdrs []string, body [][]string, termWidth int) ([]int, []string, [][]string) {
	if tableWidth(widths) <= termWidth {
		return widths, hdrs, body
	}

	// Walk columns until adding the next one would exceed termWidth; that's
	// our boundary column. Anything past it is dropped.
	keep := 0
	used := 1 // leading "|"
	for i, w := range widths {
		next := used + w + 3
		if next > termWidth {
			keep = i
			break
		}
		used = next
		keep = i + 1
	}

	// "keep" is the count of fully-kept columns. The boundary column is at
	// index "keep" (if any). Try to shrink the boundary so the table is
	// exactly termWidth wide; drop it if even the minimum doesn't fit.
	for keep > 0 || (keep == 0 && len(widths) > 0) {
		// Try fitting a boundary column of width = termWidth - used - 3.
		if keep < len(widths) {
			avail := termWidth - used - 3
			if avail >= minTruncatedColWidth {
				newWidths := append([]int(nil), widths[:keep]...)
				newWidths = append(newWidths, avail)
				newHdrs := append([]string(nil), hdrs[:keep]...)
				newHdrs = append(newHdrs, truncateCell(hdrs[keep], avail))
				newBody := make([][]string, len(body))
				for i, row := range body {
					nr := append([]string(nil), row[:keep]...)
					nr = append(nr, truncateCell(row[keep], avail))
					newBody[i] = nr
				}
				return newWidths, newHdrs, newBody
			}
		}
		// Boundary column can't be shrunk far enough; drop it and try the
		// previous column as the new boundary.
		if keep == 0 {
			break
		}
		keep--
		used -= widths[keep] + 3
	}

	// Even the first column doesn't fit termWidth; emit it anyway so the
	// output remains structurally valid (the terminal will hard-wrap).
	return widths[:1], hdrs[:1], projectColumns(body, 1)
}

// projectColumns returns body with each row trimmed to the first n columns.
func projectColumns(body [][]string, n int) [][]string {
	out := make([][]string, len(body))
	for i, row := range body {
		out[i] = row[:n]
	}
	return out
}

// truncateCell trims s to width characters, appending "..." when truncation
// happens. Width must be >= minTruncatedColWidth.
func truncateCell(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	return s[:width-3] + "..."
}

func buildColumns(t reflect.Type) []column {
	cols := make([]column, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		nameTag := f.Tag.Get("table_field")
		if nameTag == "-" {
			continue
		}
		header := nameTag
		if header == "" {
			header = f.Name
		}
		cols = append(cols, column{
			header:     header,
			fieldIndex: i,
			format:     pickFormatter(f.Type, f.Tag.Get("table_format")),
		})
	}
	return cols
}

func computeWidths(cols []column, body [][]string) []int {
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c.header)
	}
	for _, row := range body {
		for i, cell := range row {
			if n := len(cell); n > widths[i] {
				widths[i] = n
			}
		}
	}
	return widths
}

func headers(cols []column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.header
	}
	return out
}

// renderHeader returns the "Page: X/Y, Total: N" line (no trailing newline).
// When pageSize <= 0, totalPages falls back to 1 unless total is 0.
func renderHeader(page, pageSize int, total int64) string {
	var totalPages int64
	switch {
	case total <= 0:
		totalPages = 0
	case pageSize <= 0:
		totalPages = 1
	default:
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	return fmt.Sprintf("Page: %d/%d, Total: %d", page, totalPages, total)
}

func writeBorder(b *strings.Builder, widths []int) {
	b.WriteByte('+')
	for _, w := range widths {
		b.WriteString(strings.Repeat("-", w+2))
		b.WriteByte('+')
	}
	b.WriteByte('\n')
}

func writeRow(b *strings.Builder, cells []string, widths []int) {
	b.WriteByte('|')
	for i, cell := range cells {
		fmt.Fprintf(b, " %-*s |", widths[i], cell)
	}
	b.WriteByte('\n')
}
