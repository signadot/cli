package print

import "strings"

// FirstLine returns the first line of s, trimmed and truncated to 80 chars.
func FirstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 80 {
		s = s[:77] + "..."
	}
	return s
}
