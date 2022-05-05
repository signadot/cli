package sdtab

import (
	"os"
	"testing"
)

type Thing struct {
	FieldA string `sdtab:"a,1"`
	FieldB string `sdtab:"b,2"`
}

var d = []Thing{
	{FieldA: "a1", FieldB: "b1"},
	{
		FieldA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		FieldB: "b",
	},
	{
		FieldA: "a",
		FieldB: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	},
	{
		FieldA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		FieldB: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	},
}

func TestTable(t *testing.T) {
	cols := Columns[Thing]()
	tab := FromColumns(os.Stdout, cols)
	if err := tab.WriteHeader(); err != nil {
		t.Error(err)
		return
	}
	for i := range d {
		thing := &d[i]
		tab.WriteRow(*thing)
	}
	tab.Flush()
}
