package print

import "testing"

func TestFormatAttachValue(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "<nil>"},
		{"string", "hello world", "hello world"},
		{"empty string", "", ""},
		// Numbers come back from JSON as float64; JSON marshal renders
		// integral floats without a trailing decimal.
		{"int-as-float64", float64(200), "200"},
		{"true", true, "true"},
		{"false", false, "false"},
		{"map", map[string]any{"check": "frontend / returns 200"},
			`{"check":"frontend / returns 200"}`},
		{"slice", []any{"a", "b"}, `["a","b"]`},
		{"nested", map[string]any{
			"response": map[string]any{"statusCode": float64(200)},
		}, `{"response":{"statusCode":200}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := formatAttachValue(c.in)
			if got != c.want {
				t.Errorf("formatAttachValue(%#v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
