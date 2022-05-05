package sdtab

import (
	"os"
)

type Data struct {
	FieldA string `sdtab:"a,1"`
	FieldB string `sdtab:"b,2"`
}

var testData = []Data{
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

func ExampleT() {
	tab := New[Data](os.Stdout)
	if err := tab.WriteHeader(); err != nil {
		panic(err)
	}
	for _, d := range testData {
		tab.WriteRow(d)
	}
	tab.Flush()

	// Output:
	// a                                   b
	// a1                                  b1
	// aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...   b
	// a                                   bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb...
	// aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...   bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb...
}
