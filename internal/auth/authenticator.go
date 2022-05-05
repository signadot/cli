package auth

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

type t struct {
	apiKey string
}

func (t t) AuthenticateRequest(req runtime.ClientRequest, _ strfmt.Registry) error {
	req.SetHeaderParam("signadot-api-key", t.apiKey)
	return nil
}

// this is useful for using the sdk.
// returns a thing which adds auth credentials to client
// request
func Authenticator(apiKey string) runtime.ClientAuthInfoWriter {
	return t{
		apiKey: apiKey,
	}
}
