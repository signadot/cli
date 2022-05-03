package tablewriter

import (
	"fmt"
	"reflect"
	"time"
)

func formatRow(row reflect.Value, cols []column) []string {
	var vals []string
	for _, col := range cols {
		cell := row.FieldByIndex(col.Index).Interface()
		vals = append(vals, formatCell(cell, &col))
	}
	return vals
}

func formatCell(cell any, col *column) string {
	var s string

	switch c := cell.(type) {
	case time.Time:
		s = c.Format("2006-01-02 15:04:05 MST")
	default:
		s = fmt.Sprintf("%v", cell)
	}

	if col.MaxLen > 0 && len(s) > col.MaxLen {
		s = s[:col.MaxLen] + "..."
	}
	return s
}
