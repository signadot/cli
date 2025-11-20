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
	ConfigAuthSource    AuthSource = "config"
	KeyringAuthSource   AuthSource = "keyring"
	PlainTextAuthSource AuthSource = "plaintext"
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

func IsAuthenticated(authInfo *ResolvedAuth) bool {
	if authInfo == nil {
		var err error
		authInfo, err = ResolveAuth()
		if err != nil {
			return false
		}
	}
	if authInfo == nil {
		return false
	}
	return authInfo.ExpiresAt == nil || authInfo.ExpiresAt.After(time.Now())
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

	// try keyring first
	keyringStorage := NewKeyringStorage()
	auth, err := keyringStorage.Get()
	if err != nil {
		// If keyring is not available (e.g., in Docker without dbus),
		// treat it as if keyring has no credentials and fall back to plain text
		// Only return error if we have no fallback option
		auth = nil
	}
	if auth != nil {
		return &ResolvedAuth{
			Source: keyringStorage.Source(),
			Auth:   *auth,
		}, nil
	}

	// fall back to plain text file
	plainTextStorage := NewPlainTextStorage()
	auth, err = plainTextStorage.Get()
	if err != nil {
		return nil, err
	}
	if auth == nil {
		return nil, nil
	}
	return &ResolvedAuth{
		Source: plainTextStorage.Source(),
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
