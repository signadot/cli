package types

import "time"

// Auth represents the authentication information stored in the keyring
type Auth struct {
	APIKey       string     `json:"api_key,omitempty"`
	BearerToken  string     `json:"bearer_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	OrgName      string     `json:"org_name,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}
