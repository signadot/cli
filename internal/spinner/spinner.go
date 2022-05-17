// Package spinner displays a progress indicator with a live-updating status message.
//
// If the destination io.Writer is a terminal, the spinner will use color and
// will display progress updates on a single, animated line. Otherwise, color
// and animation will be disabled and each update will be printed on a new line.
package spinner

import (
	"fmt"
	"io"
	"time"

	"github.com/theckman/yacspin"
)

type T struct {
	*yacspin.Spinner
}

func New(out io.Writer, title string) *T {
	s, err := yacspin.New(yacspin.Config{
		Writer:            out,
		Frequency:         100 * time.Millisecond,
		CharSet:           yacspin.CharSets[14],
		StopCharacter:     "✓",
		StopColors:        []string{"fgGreen"},
		StopFailCharacter: "✗",
		StopFailColors:    []string{"fgRed"},
		Message:           "...",
		Suffix:            fmt.Sprintf(" %s: ", title),
	})
	if err != nil {
		panic(err)
	}
	return &T{Spinner: s}
}

// Start creates a new spinner and starts it immediately.
func Start(out io.Writer, title string) *T {
	s := New(out, title)
	if err := s.Start(); err != nil {
		panic(err)
	}
	return s
}

func (t *T) Message(msg string) {
	// Set the latest message in the underlying spinner.
	t.Spinner.Message(msg)
	// Whenever we update the message, also set that value as the one to display
	// if we fail. That way we default to showing the most recent value on error.
	t.Spinner.StopFailMessage(msg)
}

func (t *T) Messagef(format string, args ...interface{}) {
	t.Message(fmt.Sprintf(format, args...))
}
