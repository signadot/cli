package config

import (
	"fmt"
	"net/http"
	"net/textproto"
	"strings"
)

// deniedHeaders are the headers --header may not set: the CLI owns them, and
// letting a caller replace one would change who a request authenticates as, or
// break the request itself.
var deniedHeaders = map[string]bool{
	"Authorization":          true,
	"Proxy-Authorization":    true,
	"Signadot-Api-Key":       true,
	"Signadot-Cluster-Token": true,
	"Cookie":                 true,
	"User-Agent":             true,
	"Host":                   true,
	"Content-Length":         true,
	"Content-Type":           true,
	"Transfer-Encoding":      true,
	"Connection":             true,
}

// parseHeaders turns repeated --header "Name: value" flags into headers to add
// to every API request. A name given more than once keeps every value, in
// order, as HTTP allows for a repeated header.
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
		if !validHeaderName(name) {
			return nil, fmt.Errorf("invalid --header %q: %q is not a valid header name", f, name)
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("invalid --header %q: the value cannot span lines", f)
		}
		canonical := textproto.CanonicalMIMEHeaderKey(name)
		if deniedHeaders[canonical] {
			return nil, fmt.Errorf("invalid --header %q: %s is set by the CLI and cannot be overridden", f, canonical)
		}
		h.Add(canonical, value)
	}
	return h, nil
}

// validHeaderName reports whether s is an RFC 9110 field name (a token).
func validHeaderName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", r):
		default:
			return false
		}
	}
	return true
}

// headerTransport adds the --header headers to every request. The SDK sets
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
