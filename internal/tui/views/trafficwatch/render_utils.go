package trafficwatch

import (
	"bufio"
	"io"
	"mime"
	"net/http"
	"strings"
)

func truncateURL(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Heuristic: is this content type "renderable text"?
func isTextContentType(ct string) bool {
	if ct == "" {
		return false
	}

	mt, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return false // be conservative on parse errors
	}

	// Text/* is text.
	if strings.HasPrefix(mt, "text/") {
		return true
	}

	// If a charset is explicitly present, it's almost always text you can render.
	if _, ok := params["charset"]; ok {
		return true
	}

	// Common "application/*" types that are text.
	switch mt {
	case "application/json",
		"application/xml",
		"application/javascript",
		"application/ecmascript",
		"application/x-www-form-urlencoded":
		return true
	}

	// RFC 6838 structured suffixes — JSON/XML are typically renderable.
	if strings.HasSuffix(mt, "+json") || strings.HasSuffix(mt, "+xml") {
		return true
	}

	return false
}

// Fallback: sniff first bytes (without consuming the body).
func looksTextBySniff(peek []byte) bool {
	// http.DetectContentType implements a small sniffing algorithm.
	detected := http.DetectContentType(peek)
	return isTextContentType(detected)
}

func isRespBodyRenderable(resp *http.Response) bool {
	// Honor Content-Disposition: attachment (usually a download).
	if cd := resp.Header.Get("Content-Disposition"); strings.Contains(cd, "attachment") {
		return false
	}

	// If Content-Type looks text-like, render.
	if isTextContentType(resp.Header.Get("Content-Type")) {
		return true
	}

	// Otherwise, sniff the first 512 bytes without consuming the body.
	br := bufio.NewReader(resp.Body)
	resp.Body = io.NopCloser(br)
	peek, _ := br.Peek(512) // ok if fewer bytes available
	return looksTextBySniff(peek)
}

func isReqBodyRenderable(req *http.Request) bool {
	// If Content-Type looks text-like, render.
	if isTextContentType(req.Header.Get("Content-Type")) {
		return true
	}

	// Otherwise, sniff the first 512 bytes without consuming the body.
	br := bufio.NewReader(req.Body)
	req.Body = io.NopCloser(br)
	peek, _ := br.Peek(512) // ok if fewer bytes available
	return looksTextBySniff(peek)
}
