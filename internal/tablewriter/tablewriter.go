// Package tablewriter is an opinionated wrapper around text/tabwriter for our CLI.
package tablewriter

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
)

type T[R any] struct {
	tw   *tabwriter.Writer
	cols []column
}

func New[R any](w io.Writer) (*T[R], error) {
	t := &T[R]{
		tw: tabwriter.NewWriter(w, 0, 0, 3, ' ', 0),
	}

	// Extract column metadata from each field of the row struct.
	var row R
	rowType := reflect.TypeOf(row)
	for _, field := range reflect.VisibleFields(rowType) {
		t.cols = append(t.cols, newColumn(field))
	}

	if err := t.WriteHeader(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *T[R]) WriteHeader() error {
	var values []string
	for _, col := range t.cols {
		values = append(values, col.Header)
	}
	return t.writeRow(values)
}

func (t *T[R]) writeRow(values []string) error {
	_, err := fmt.Fprintln(t.tw, strings.Join(values, "\t"))
	return err
}

func (t *T[R]) WriteRow(row R) error {
	return t.writeRow(formatRow(reflect.ValueOf(row), t.cols))
}

func (t *T[R]) Flush() error {
	return t.tw.Flush()
}
