package sdtab

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

func taggedStructColumns[S any]() Columns[S] {
	return taggedStruct[S]{}
}

// wrapper around tagged structs making them
// implement Columns
type taggedStruct[S any] struct{}

func (t taggedStruct[S]) Columns() []Column[S] {
	var row S
	return structColumns[S](reflect.TypeOf(row))
}

func structColumns[R any](rowType reflect.Type) []Column[R] {
	var res []Column[R]
	for _, field := range reflect.VisibleFields(rowType) {
		c := structFieldColumn[R](field)
		if c != nil {
			res = append(res, *c)
		}
	}
	return res
}

func structFieldColumn[R any](f reflect.StructField) *Column[R] {
	tag := f.Tag.Get("sdtab")
	if tag == "" {
		return nil
	}

	// Split by commas. The first part is the column title.
	parts := strings.Split(tag, ",")
	res := &Column[R]{
		Title: parts[0],
		Get: func(r R) string {
			val := reflect.ValueOf(r).FieldByName(f.Name).Interface()
			return fmt.Sprint(val)
		},
	}

	// The rest of the comma-separated parts are "key" or "key=value" attributes.
	for _, attr := range parts[1:] {
		switch attr {
		case "trunc":
			res.Truncate = true
		}
	}

	return res
}

func FromStruct[S any](w io.Writer) *T[S] {
	return New[S](w, taggedStructColumns[S]())
}
