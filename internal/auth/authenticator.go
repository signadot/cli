package auth

import (
	"os"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

type t struct{}

// TODO use config
func GetApiKey() string {
	return os.Getenv("SIGNADOT_API_KEY")
}

func (t t) AuthenticateRequest(req runtime.ClientRequest, _ strfmt.Registry) error {
	req.SetHeaderParam("signadot-api-key", GetApiKey())
	return nil
}

// this is useful for using the sdk.
// returns a thing which adds auth credentials to client
// request
func Authenticator() runtime.ClientAuthInfoWriter {
	return t{}
}
