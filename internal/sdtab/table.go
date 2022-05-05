package sdtab

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"

	"golang.org/x/term"
)

type column struct {
	fieldName string
	title     string
	weight    int
	notrunc   bool
}

func (c *column) grab(v any) string {
	val := reflect.ValueOf(v)
	val = val.FieldByName(c.fieldName)
	return fmt.Sprintf("%s", val)
}

func (c *column) format(v string, w int) string {
	if w < len(v) && !c.notrunc {
		if w > 5 {
			return v[:w-3] + "..."
		}
		return v[:w]
	}
	return v
}

func structFieldColumn(f reflect.StructField) *column {
	parts := strings.Split(f.Tag.Get("sdtab"), ",")
	if len(parts) == 0 {
		return nil
	}
	res := &column{title: parts[0], fieldName: f.Name}
	if len(parts) == 1 {
		return res
	}
	v, e := strconv.Atoi(parts[1])
	if e != nil {
		res.weight = 1
	} else {
		res.weight = v
	}
	if len(parts) == 2 {
		return res
	}
	switch parts[2] {
	case "-":
		res.notrunc = true
	default:

	}
	return res
}

type T[R any] struct {
	tw            *tabwriter.Writer
	columns       []*column
	width, height int
	rows          int
	totalWeight   int
}

func structColumns(rowType reflect.Type) []*column {
	var res []*column
	for _, field := range reflect.VisibleFields(rowType) {
		c := structFieldColumn(field)
		if c != nil {
			res = append(res, c)
		}
	}
	return res
}

func New[R any](w io.Writer) *T[R] {
	var row R
	cols := structColumns(reflect.TypeOf(row))

	width, height, e := term.GetSize(0)
	if e != nil {
		width = 100
		height = 50
	}
	totalWeight := 0
	for _, c := range cols {
		totalWeight += c.weight
	}

	return &T[R]{
		tw:          tabwriter.NewWriter(w, 0, 0, 3, ' ', 0),
		columns:     cols,
		width:       width,
		height:      height,
		totalWeight: totalWeight,
	}
}

func (t *T[R]) Flush() error {
	return t.tw.Flush()
}

func (t *T[R]) WriteHeader() error {
	cells := make([]string, 0, len(t.columns))
	for _, col := range t.columns {
		cells = append(cells, col.title)
	}
	return t.writeRow(cells)
}

func (t *T[R]) writeRow(cells []string) error {
	for i, col := range t.columns {
		trunc := (col.weight * t.width) / t.totalWeight
		if i != 0 {
			_, err := t.tw.Write([]byte("\t"))
			if err != nil {
				return err
			}
		}
		elt := col.format(cells[i], trunc)
		_, err := t.tw.Write([]byte(elt))
		if err != nil {
			return err
		}
	}
	_, err := t.tw.Write([]byte("\n"))

	return err
}

func (t *T[R]) WriteRow(r R) error {
	cells := make([]string, 0, len(t.columns))
	for _, col := range t.columns {
		cells = append(cells, col.grab(r))
	}
	return t.writeRow(cells)
}
