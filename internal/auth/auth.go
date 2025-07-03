package auth

import (
	"net/http"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/signadot/go-sdk/transport"
	"github.com/spf13/viper"
)

type AuthSource string

const (
	ConfigAuthSource  AuthSource = "config"
	KeyringAuthSource AuthSource = "keyring"
)

type Auth struct {
	APIKey       string     `json:"apiKey,omitempty"`
	BearerToken  string     `json:"bearerToken,omitempty"`
	RefreshToken string     `json:"refreshToken,omitempty"`
	OrgName      string     `json:"orgName"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
}

type ResolvedAuth struct {
	Auth
	Source AuthSource `json:"source"`
}

func ResolveAuth() (*ResolvedAuth, error) {
	auth, err := loadAuth()
	if err != nil {
		return nil, err
	}

	if auth == nil {
		return nil, nil
	}

	// fall back to config file for org if not defined
	if auth.OrgName == "" {
		auth.OrgName = viper.GetString("org")
	}
	return auth, nil
}

func loadAuth() (*ResolvedAuth, error) {
	// give precedence to config
	apiKey := viper.GetString("api_key")
	if apiKey != "" {
		return &ResolvedAuth{
			Source: ConfigAuthSource,
			Auth: Auth{
				APIKey: apiKey,
			},
		}, nil
	}

	auth, err := GetAuthFromKeyring()
	if err != nil {
		return nil, err
	}
	if auth == nil {
		return nil, nil
	}
	return &ResolvedAuth{
		Source: KeyringAuthSource,
		Auth:   *auth,
	}, nil
}

func GetHeaders() (http.Header, error) {
	authInfo, err := ResolveAuth()
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	if authInfo == nil {
		return headers, nil
	}
	if authInfo.APIKey != "" {
		headers.Set(transport.APIKeyHeader, authInfo.APIKey)
	} else if authInfo.BearerToken != "" {
		headers.Set(runtime.HeaderAuthorization, "Bearer "+authInfo.BearerToken)
	}
	return headers, nil
}
