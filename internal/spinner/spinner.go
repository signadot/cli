// Package spinner displays a progress indicator with a live-updating status message.
//
// If the destination io.Writer is a terminal, the spinner will use color and
// will display progress updates on a single, animated line. Otherwise, color
// and animation will be disabled and each update will be printed on a new line.
package spinner

import (
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/theckman/yacspin"
	"golang.org/x/term"
)

type T struct {
	*yacspin.Spinner

	config *yacspin.Config
	// prefixWidth is the width of the fixed prefix before each message line.
	prefixWidth int
}

func New(out io.Writer, title string) *T {
	cfg := yacspin.Config{
		Writer:            out,
		Frequency:         100 * time.Millisecond,
		CharSet:           yacspin.CharSets[14],
		StopCharacter:     "✓",
		StopColors:        []string{"fgGreen"},
		StopFailCharacter: "✗",
		StopFailColors:    []string{"fgRed"},
		Message:           "...",
		Suffix:            fmt.Sprintf(" %s: ", title),
	}
	s, err := yacspin.New(cfg)
	if err != nil {
		panic(err)
	}
	return &T{
		Spinner:     s,
		config:      &cfg,
		prefixWidth: utf8.RuneCountInString(cfg.CharSet[0] + cfg.Suffix),
	}
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
	// Whenever we update the message, also set that value as the one to display
	// if we fail. That way we default to showing the most recent value on error.
	t.Spinner.StopFailMessage(msg)

	// See if we need to truncate the message to avoid wrapping to a second line.
	// The spinner currently doesn't support such wrapping.
	termWidth := t.termWidth()
	if termWidth > t.prefixWidth {
		msg = sdtab.Truncate(msg, termWidth-t.prefixWidth)
	}

	// Set the latest message in the underlying spinner.
	t.Spinner.Message(msg)
}

func (t *T) Messagef(format string, args ...interface{}) {
	t.Message(fmt.Sprintf(format, args...))
}

// termWidth returns the terminal width, if known, or 0 if unknown.
func (t *T) termWidth() int {
	file, ok := t.config.Writer.(*os.File)
	if !ok {
		return 0
	}
	width, _, err := term.GetSize(int(file.Fd()))
	if err != nil {
		return 0
	}
	return width
}
