package auth

import "github.com/zalando/go-keyring"

const (
	keyringService = "signadot-cli"
	tokenKey       = "token"
	orgKey         = "org"
)

// StoreToken stores the auth token securely in the system keyring
func StoreToken(token string) error {
	return keyring.Set(keyringService, tokenKey, token)
}

// GetToken retrieves the auth token from the system keyring
func GetToken() (string, error) {
	return keyring.Get(keyringService, tokenKey)
}

// DeleteToken removes the auth token from the system keyring
func DeleteToken() error {
	return keyring.Delete(keyringService, tokenKey)
}

func StoreOrg(org string) error {
	return keyring.Set(keyringService, orgKey, org)
}

func GetOrg() (string, error) {
	return keyring.Get(keyringService, orgKey)
}

func DeleteOrg() error {
	return keyring.Delete(keyringService, orgKey)
}
