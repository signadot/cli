package sdtab

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	margin      = "   "
	truncSuffix = ".."
	truncMinLen = 6
)

type T[R any] struct {
	out io.Writer

	columns    []Column[R]
	termWidth  int
	termHeight int

	rowBuf [][]string
}

type Column[Row any] struct {
	Title    string
	Truncate bool
	Get      func(r Row) string
	MinWidth int
}

type Columns[R any] interface {
	Columns() []Column[R]
}

func New[R any](w io.Writer, cs Columns[R]) *T[R] {
	t := &T[R]{
		out:     w,
		columns: cs.Columns(),
	}

	// Try to auto-detect the terminal size.
	t.detectTermSize()

	return t
}

func (t *T[R]) SetTermSize(width, height int) {
	t.termWidth = width
	t.termHeight = height
}

func (t *T[R]) AddHeader() {
	header := make([]string, len(t.columns))
	for i, col := range t.columns {
		header[i] = col.Title
	}
	t.rowBuf = append(t.rowBuf, header)
}

func (t *T[R]) AddRow(r R) {
	row := make([]string, len(t.columns))
	for i, col := range t.columns {
		row[i] = col.Get(r)
	}
	t.rowBuf = append(t.rowBuf, row)
}

func (t *T[R]) Flush() error {
	// Compute the desired width of each column.
	colWidth := t.columnWidths()

	// Write out the rows with the computed column widths.
	for _, row := range t.rowBuf {
		if err := t.writeRow(row, colWidth); err != nil {
			return err
		}
	}

	// Clear out the row buffer.
	t.rowBuf = t.rowBuf[:0]
	return nil
}

func (t *T[R]) detectTermSize() {
	file, ok := t.out.(*os.File)
	if !ok {
		return
	}
	width, height, err := term.GetSize(int(file.Fd()))
	if err != nil {
		return
	}
	t.SetTermSize(width, height)
}

func (t *T[R]) writeRow(row []string, colWidth []int) error {
	for i, v := range row {
		// Add some margin between columns.
		if i > 0 {
			if _, err := fmt.Fprint(t.out, margin); err != nil {
				return err
			}
		}

		// Truncate the value, if necessary, and then print it.
		cw := colWidth[i]
		v = truncate(v, cw)
		if _, err := fmt.Fprint(t.out, v); err != nil {
			return err
		}

		// Add trailing spaces to fill up the column, unless it's the last one.
		if i == len(row)-1 {
			continue
		}
		pad := cw - utf8.RuneCountInString(v)
		if pad > 0 {
			if _, err := fmt.Fprint(t.out, strings.Repeat(" ", pad)); err != nil {
				return err
			}
		}
	}
	// Write newline.
	if _, err := fmt.Fprintln(t.out); err != nil {
		return err
	}
	return nil
}

func (t *T[R]) columnWidths() []int {
	// Find the max width of the data in each column.
	colWidth := make([]int, len(t.columns))
	for _, row := range t.rowBuf {
		for i, v := range row {
			// Use the printed width rather than the number of bytes.
			w := utf8.RuneCountInString(v)
			if w > colWidth[i] {
				colWidth[i] = w
			}
		}
	}

	// Find the total width of the table, including margin.
	tableWidth := len(margin) * (len(t.columns) - 1)
	for _, v := range colWidth {
		tableWidth += v
	}

	// Use the full widths if the terminal is unlimited, or the table fits.
	if t.termWidth == 0 {
		return colWidth
	}

	// Find the total width of all truncatable columns.
	truncLen := 0
	for i, col := range t.columns {
		if col.Truncate {
			truncLen += colWidth[i]
		}
	}

	// The rest of the table width can't budge.
	fixedLen := tableWidth - truncLen
	if tableWidth == t.termWidth {
		return colWidth
	}
	if tableWidth > t.termWidth {
		// How small would we have to shrink the truncatable columns to fit?
		goal := t.termWidth - fixedLen
		// Try to shrink all truncatable columns by the same factor.
		factor := float64(goal) / float64(truncLen)
		for i, col := range t.columns {
			w := int(float64(colWidth[i]) * factor)
			if col.Truncate {
				// Don't shrink any given column too far.
				if w < truncMinLen {
					w = truncMinLen
				}
			}
			colWidth[i] = w
		}
		return colWidth
	}
	// fill out to the width of the terminal
	factor := float64(t.termWidth-tableWidth) / float64(len(t.columns))
	pad := int(math.Round(factor))
	for i := range t.columns {
		if tableWidth+pad > t.termWidth {
			// in case rounding went past the termWidth
			pad = t.termWidth - tableWidth
		}
		colWidth[i] += pad
	}
	return colWidth
}

func truncate(v string, maxWidth int) string {
	if utf8.RuneCountInString(v) <= maxWidth {
		return v
	}

	// If the maxWidth is long enough, save some space for the suffix.
	suffix := ""
	if maxWidth > 5 {
		maxWidth -= len(truncSuffix)
		suffix = truncSuffix
	}

	// Truncate by printed width (rune count) rather than byte length.
	runes := 0
	for offset := range v {
		if runes >= maxWidth {
			return v[:offset] + suffix
		}
		runes++
	}
	// We didn't need to truncate.
	return v
}
