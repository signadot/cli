package auth

import "github.com/zalando/go-keyring"

const (
	keyringService = "signadot-cli"
)

// GetCurrentUser returns the current user identifier for keyring operations
// This can be extended later to support multiple profiles
func GetCurrentUser() string {
	// TODO: Make this configurable through config file or environment variable
	return "default"
}

// StoreToken stores the auth token securely in the system keyring
func StoreToken(token string) error {
	return keyring.Set(keyringService, GetCurrentUser(), token)
}

// GetToken retrieves the auth token from the system keyring
func GetToken() (string, error) {
	return keyring.Get(keyringService, GetCurrentUser())
}

// DeleteToken removes the auth token from the system keyring
func DeleteToken() error {
	return keyring.Delete(keyringService, GetCurrentUser())
}
