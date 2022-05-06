package sdtab

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

func ExampleTruncate() {
	type Data struct {
		FieldA string `sdtab:"A,trunc"`
		FieldB string `sdtab:"B,trunc"`
	}

	var testData = []Data{
		{FieldA: "a1", FieldB: "b1"},
		{
			FieldA: strings.Repeat("a", 70),
			FieldB: "b",
		},
		{
			FieldA: "a",
			FieldB: strings.Repeat("b", 180),
		},
		{
			FieldA: strings.Repeat("a", 70),
			FieldB: strings.Repeat("b", 180),
		},
	}

	tab := FromStruct[Data](os.Stdout)
	tab.SetTermSize(100, 50)
	tab.AddHeader()
	for _, d := range testData {
		tab.AddRow(d)
	}
	tab.Flush()

	// Output:
	// A                             B
	// a1                            b1
	// aaaaaaaaaaaaaaaaaaaaaaaaa..   b
	// a                             bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb..
	// aaaaaaaaaaaaaaaaaaaaaaaaa..   bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb..
}

func FuzzTruncate(f *testing.F) {
	f.Add("abc", 10)
	f.Add("abcdef", 6)
	f.Add(strings.Repeat("abc", 70), 30)
	f.Add(strings.Repeat("Hello, 世界", 70), 8)

	f.Fuzz(func(t *testing.T, in string, truncLen int) {
		inLen := utf8.RuneCountInString(in)
		if truncLen < 0 {
			truncLen = 0
		}
		wantLen := truncLen
		if inLen < wantLen {
			wantLen = inLen
		}

		out := truncate(in, truncLen)
		outLen := utf8.RuneCountInString(out)
		if outLen > truncLen {
			t.Errorf("RuneCountInString() = %v; want %v", outLen, wantLen)
		}
		if utf8.ValidString(in) && !utf8.ValidString(out) {
			t.Errorf("truncation produced invalid UTF-8 string %q", out)
		}
		if inLen <= truncLen && out != in {
			t.Errorf("truncate(%q, %v) = %q; expected no change", in, truncLen, out)
		}
	})
}
