package print

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-units"
)

// AttachEvent represents a structured event emitted during --attach mode.
type AttachEvent struct {
	Time   time.Time `json:"time"`
	Type   string    `json:"type"`             // "log", "output", "result"
	Step   string    `json:"step,omitempty"`   // for log events; for output events, the step that produced a plan-level output
	Stream string    `json:"stream,omitempty"` // "stdout" or "stderr", for log events
	Msg    string    `json:"msg,omitempty"`    // for log events
	Name   string    `json:"name,omitempty"`   // for output events

	// For output events:
	Kind        string `json:"kind,omitempty"`        // "inline" | "artifact"
	Value       any    `json:"value,omitempty"`       // present for inline outputs
	Size        int64  `json:"size,omitempty"`        // bytes, for artifact outputs
	Ready       *bool  `json:"ready,omitempty"`       // for artifact outputs (always set when Kind=artifact)
	ContentType string `json:"contentType,omitempty"` // for artifact outputs, when metadata.contentType is set

	ID     string `json:"id,omitempty"`     // for created/result events: the execution ID
	PlanID string `json:"planID,omitempty"` // for created events: the source plan
	Phase  string `json:"phase,omitempty"`  // for result events
	Error  string `json:"error,omitempty"`  // for result events (if failed); for output events, an artifact-side error
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
		if e.Step != "" {
			fmt.Fprintf(&b, " step=%s", e.Step)
		}
		switch e.Kind {
		case "artifact":
			fmt.Fprintf(&b, " kind=artifact")
			if e.Size > 0 {
				fmt.Fprintf(&b, " size=%s", units.HumanSize(float64(e.Size)))
			}
			if e.Ready != nil {
				fmt.Fprintf(&b, " ready=%t", *e.Ready)
			}
			if e.ContentType != "" {
				fmt.Fprintf(&b, " content_type=%s", e.ContentType)
			}
			if e.Error != "" {
				fmt.Fprintf(&b, " error=%s", quoteIfNeeded(e.Error))
			}
		case "inline":
			fmt.Fprintf(&b, " kind=inline")
			fmt.Fprintf(&b, " value=%s", quoteIfNeeded(fmt.Sprint(e.Value)))
		default:
			// Older emit sites that didn't set Kind: keep the legacy
			// value=<...> shape so existing log consumers don't break.
			fmt.Fprintf(&b, " value=%s", quoteIfNeeded(fmt.Sprint(e.Value)))
		}
	case "created":
		fmt.Fprintf(&b, " id=%s", e.ID)
		if e.PlanID != "" {
			fmt.Fprintf(&b, " plan_id=%s", e.PlanID)
		}
	case "result":
		fmt.Fprintf(&b, " id=%s", e.ID)
		fmt.Fprintf(&b, " phase=%s", e.Phase)
		if e.Error != "" {
			fmt.Fprintf(&b, " error=%s", quoteIfNeeded(e.Error))
		}
	}

	return b.String()
}

func quoteIfNeeded(s string) string {
	if s == "" || strings.ContainsAny(s, " \t\n\"=") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
