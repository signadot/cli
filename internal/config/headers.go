package config

import (
	"fmt"
	"net/http"
	"net/textproto"
	"strings"

	"golang.org/x/net/http/httpguts"
)

// deniedHeaders are the headers --header may not set: the CLI or Go's HTTP
// transport owns them, and letting a caller set one would change who a request
// authenticates as, or break the request or the reading of its response.
var deniedHeaders = map[string]bool{
	// Who the request is from.
	"Authorization":          true,
	"Proxy-Authorization":    true,
	"Signadot-Api-Key":       true,
	"Signadot-Cluster-Token": true,
	"Cookie":                 true,
	"User-Agent":             true,
	// What the request and response are. Accept-Encoding in particular turns
	// off the transport's transparent decompression, so the SDK would be
	// handed gzipped bytes; Accept would go out next to the one it sets.
	"Accept":          true,
	"Accept-Encoding": true,
	"Content-Type":    true,
	"Content-Length":  true,
	// Framing and connection management, which belong to the transport.
	"Host":              true,
	"Transfer-Encoding": true,
	"Connection":        true,
	"Keep-Alive":        true,
	"Proxy-Connection":  true,
	"Te":                true,
	"Trailer":           true,
	"Upgrade":           true,
	"Expect":            true,
}

// parseHeaders turns repeated --header "Name: value" flags into headers to add
// to the requests made through the CLI's API client (see API.GetBaseTransport).
// A name given more than once keeps every value, in order, as HTTP allows for a
// repeated header. Anything Go's HTTP transport would refuse is refused here,
// so a bad flag fails before any request is sent.
func parseHeaders(flags []string) (http.Header, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	h := http.Header{}
	for _, f := range flags {
		name, value, ok := strings.Cut(f, ":")
		if !ok {
			return nil, fmt.Errorf("invalid --header %q: want \"Name: value\"", f)
		}
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if !httpguts.ValidHeaderFieldName(name) {
			return nil, fmt.Errorf("invalid --header %q: %q is not a valid header name", f, name)
		}
		if !httpguts.ValidHeaderFieldValue(value) {
			return nil, fmt.Errorf("invalid --header %q: the value contains a control character", f)
		}
		canonical := textproto.CanonicalMIMEHeaderKey(name)
		if deniedHeaders[canonical] {
			return nil, fmt.Errorf("invalid --header %q: %s is set by the CLI and cannot be overridden", f, canonical)
		}
		h.Add(canonical, value)
	}
	return h, nil
}

// headerTransport adds the --header headers to every request it carries. The SDK sets
// User-Agent in a wrapper around its own transport, which it does not use once
// it is handed an http.Client with a transport of its own, so this sets
// User-Agent as well.
type headerTransport struct {
	inner     http.RoundTripper
	userAgent string
	headers   http.Header
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for name, values := range t.headers {
		for _, v := range values {
			req.Header.Add(name, v)
		}
	}
	if t.userAgent != "" {
		req.Header.Set("User-Agent", t.userAgent)
	}
	return t.inner.RoundTrip(req)
}
