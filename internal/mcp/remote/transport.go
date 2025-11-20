package remote

import (
	"net/http"

	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/go-sdk/transport"
)

// authTransport wraps an http.RoundTripper to add auth headers
type authTransport struct {
	http.RoundTripper
	authInfo *auth.ResolvedAuth
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.authInfo != nil {
		if t.authInfo.APIKey != "" {
			req.Header.Set(transport.APIKeyHeader, t.authInfo.APIKey)
		} else if t.authInfo.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+t.authInfo.BearerToken)
		}
	}
	return t.RoundTripper.RoundTrip(req)
}
