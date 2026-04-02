package print

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/jclem/sseparser"
)

type sseEvent struct {
	Event string `sse:"event"`
	Data  string `sse:"data"`
}

type sseMessage struct {
	Message string `json:"message"`
	Cursor  string `json:"cursor"`
}

// ParseSSEStream reads SSE events and writes message content to out.
// Returns the last cursor and any error.
func ParseSSEStream(reader io.Reader, out io.Writer) (string, error) {
	scanner := sseparser.NewStreamScanner(reader)
	var lastCursor string

	for {
		var e sseEvent
		_, err := scanner.UnmarshalNext(&e)
		if err != nil {
			if errors.Is(err, sseparser.ErrStreamEOF) {
				err = nil
			}
			return lastCursor, err
		}

		switch e.Event {
		case "message":
			var m sseMessage
			err = json.Unmarshal([]byte(e.Data), &m)
			if err != nil {
				return lastCursor, err
			}
			if m.Message == "" {
				continue
			}
			out.Write([]byte(m.Message))
			lastCursor = m.Cursor
		case "error":
			return lastCursor, errors.New(string(e.Data))
		case "signal":
			switch e.Data {
			case "EOF":
				return lastCursor, nil
			case "RESTART":
				out.Write([]byte("\n\n-------------------------------------------------------------------------------\n"))
				out.Write([]byte("WARNING: The execution has been restarted...\n\n"))
			}
		}
	}
}
