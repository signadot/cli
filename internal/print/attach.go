package print

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// AttachEvent represents a structured event emitted during --attach mode.
type AttachEvent struct {
	Time   time.Time `json:"time"`
	Type   string    `json:"type"`             // "log", "output", "result"
	Step   string    `json:"step,omitempty"`   // for log events
	Stream string    `json:"stream,omitempty"` // "stdout" or "stderr", for log events
	Msg    string    `json:"msg,omitempty"`    // for log events
	Name   string    `json:"name,omitempty"`   // for output events
	Value  any       `json:"value,omitempty"`  // for output events
	ID     string    `json:"id,omitempty"`     // for result events
	Phase  string    `json:"phase,omitempty"`  // for result events
	Error  string    `json:"error,omitempty"`  // for result events (if failed)
}

// AttachWriter writes structured events to an io.Writer in either
// JSON (one object per line) or slog-style text format.
type AttachWriter struct {
	mu   sync.Mutex
	out  io.Writer
	json bool
}

// NewAttachWriter creates an AttachWriter. If jsonMode is true, events
// are written as JSON lines; otherwise as slog-style text.
func NewAttachWriter(out io.Writer, jsonMode bool) *AttachWriter {
	return &AttachWriter{out: out, json: jsonMode}
}

// Emit writes an event.
func (w *AttachWriter) Emit(e AttachEvent) {
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.json {
		data, _ := json.Marshal(e)
		w.out.Write(data)
		w.out.Write([]byte("\n"))
	} else {
		w.out.Write([]byte(formatText(e)))
		w.out.Write([]byte("\n"))
	}
}

func formatText(e AttachEvent) string {
	var b strings.Builder
	fmt.Fprintf(&b, "time=%s", e.Time.Format(time.TimeOnly))
	fmt.Fprintf(&b, " type=%s", e.Type)

	switch e.Type {
	case "log":
		if e.Step != "" {
			fmt.Fprintf(&b, " step=%s", e.Step)
		}
		if e.Stream != "" {
			fmt.Fprintf(&b, " stream=%s", e.Stream)
		}
		fmt.Fprintf(&b, " msg=%s", quoteIfNeeded(strings.TrimRight(e.Msg, "\n")))
	case "output":
		fmt.Fprintf(&b, " name=%s", e.Name)
		fmt.Fprintf(&b, " value=%s", quoteIfNeeded(formatAttachValue(e.Value)))
	case "result":
		fmt.Fprintf(&b, " id=%s", e.ID)
		fmt.Fprintf(&b, " phase=%s", e.Phase)
		if e.Error != "" {
			fmt.Fprintf(&b, " error=%s", quoteIfNeeded(e.Error))
		}
	}

	return b.String()
}

// formatAttachValue renders an output value for the slog-style text
// stream. Strings stay unquoted (the common case for plan outputs).
// Structured values (maps, slices) are JSON-encoded so the reader
// sees `{"check":"..."}` instead of Go's default `map[check:...]`.
// nil renders as <nil> matching the prior behavior. Numbers and bools
// render as their JSON forms (`200`, `true`).
func formatAttachValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "<nil>"
	case string:
		return x
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(b)
	}
}

func quoteIfNeeded(s string) string {
	if s == "" || strings.ContainsAny(s, " \t\n\"=") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
