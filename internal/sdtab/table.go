package sdtab

import (
	"fmt"
	"golang.org/x/term"
	"io"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"
)

type Column[T any] interface {
	Grab(T) string
	Name() string
	Weight() int
	Format(s string, t int) string
}

type column[T any] struct {
	fname   string
	name    string
	weight  int
	notrunc bool
}

func (c *column[T]) Grab(v T) string {
	val := reflect.ValueOf(v)
	val = val.FieldByName(c.fname)
	return fmt.Sprintf("%s", val)
}

func (c *column[T]) Name() string {
	return c.name
}

func (c *column[T]) Weight() int {
	return c.weight
}

func (c *column[T]) Format(v string, w int) string {
	if w < len(v) && !c.notrunc {
		if w > 5 {
			return v[:w-3] + "..."
		}
		return v[:w]
	}
	return v
}

func StructFieldColumn[T any](f reflect.StructField) Column[T] {
	parts := strings.Split(f.Tag.Get("sdtab"), ",")
	if len(parts) == 0 {
		return nil
	}
	res := &column[T]{name: parts[0], fname: f.Name}
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
	columns       []Column[R]
	width, height int
	rows          int
	ttlWeight     int
	buf           []string
}

func Columns[T any]() []Column[T] {
	var v T
	rowType := reflect.TypeOf(v)
	var res []Column[T]
	for _, field := range reflect.VisibleFields(rowType) {
		c := StructFieldColumn[T](field)
		if c != nil {
			res = append(res, c)
		}
	}
	return res
}

func New[R any](w io.Writer) *T[R] {
	return FromColumns[R](w, Columns[R]())
}

func FromColumns[R any](w io.Writer, cols []Column[R]) *T[R] {
	width, height, e := term.GetSize(0)
	if e != nil {
		width = 100
		height = 50
	}
	ttlWeight := 0
	for _, c := range cols {
		ttlWeight += c.Weight()
	}

	return &T[R]{
		tw:        tabwriter.NewWriter(w, 0, 0, 3, ' ', 0),
		columns:   cols,
		width:     width,
		height:    height,
		ttlWeight: ttlWeight,
		buf:       make([]string, len(cols))}
}

func (t *T[R]) Flush() error {
	return t.tw.Flush()
}

func (t *T[R]) WriteHeader() error {
	for i, col := range t.columns {
		t.buf[i] = col.Name()
	}
	return t.write()
}

func (t *T[R]) write() error {
	for i, col := range t.columns {
		trunc := (col.Weight() * t.width) / t.ttlWeight
		if i != 0 {
			_, err := t.tw.Write([]byte("\t"))
			if err != nil {
				return err
			}
		}
		elt := col.Format(t.buf[i], trunc)
		_, err := t.tw.Write([]byte(elt))
		if err != nil {
			return err
		}
	}
	_, err := t.tw.Write([]byte("\n"))

	return err
}

func (t *T[R]) WriteRow(r R) error {
	for i, col := range t.columns {
		t.buf[i] = col.Grab(r)
	}
	return t.write()
}
