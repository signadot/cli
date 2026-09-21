package print

import (
	"strings"
	"testing"
)

func TestQuoteIfNeeded(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "plain", "plain"},
		{"empty", "", `""`},
		{"space", "a b", `"a b"`},
		{"equals", "k=v", `"k=v"`},
		{"tab", "a\tb", `"a\tb"`},
		{"newline", "a\nb", `"a\nb"`},
		{"carriage_return", "a\rb", `"a\rb"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := quoteIfNeeded(tc.in)
			if got != tc.want {
				t.Errorf("quoteIfNeeded(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if strings.ContainsRune(got, '\r') {
				t.Errorf("quoteIfNeeded(%q) leaked a raw carriage return: %q", tc.in, got)
			}
		})
	}
}

func TestFormatTextLogTrimsTrailingCRLF(t *testing.T) {
	out := formatText(AttachEvent{Type: "log", Msg: "hello\r\n"})
	if strings.ContainsRune(out, '\r') {
		t.Errorf("formatText leaked a raw carriage return: %q", out)
	}
	if !strings.Contains(out, "msg=hello") {
		t.Errorf("formatText(%q) = %q, want it to contain msg=hello", "hello\r\n", out)
	}
}

func TestFormatTextLogEscapesEmbeddedCR(t *testing.T) {
	out := formatText(AttachEvent{Type: "log", Msg: "line1\rline2"})
	if strings.ContainsRune(out, '\r') {
		t.Errorf("formatText leaked a raw carriage return: %q", out)
	}
	if !strings.Contains(out, `msg="line1\rline2"`) {
		t.Errorf("formatText(%q) = %q, want the message quoted with an escaped CR", "line1\rline2", out)
	}
}

func TestFormatTextLogPlainMsgUnquoted(t *testing.T) {
	out := formatText(AttachEvent{Type: "log", Msg: "plain"})
	if !strings.Contains(out, "msg=plain") {
		t.Errorf("formatText(%q) = %q, want unquoted msg=plain", "plain", out)
	}
}
