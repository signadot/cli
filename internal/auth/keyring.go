package auth

import (
	"encoding/json"
	"errors"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "signadot-cli"
	authKey        = "auth"
)

func storeAuthInKeyring(auth *Auth) error {
	authJson, err := json.Marshal(auth)
	if err != nil {
		return err
	}
	return keyring.Set(keyringService, authKey, string(authJson))
}

func getAuthFromKeyring() (*Auth, error) {
	authJson, err := keyring.Get(keyringService, authKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// decode the auth
	var auth Auth
	err = json.Unmarshal([]byte(authJson), &auth)
	if err != nil {
		// this is an unlikely state. Remove the entry from the keyring and
		// allow the user to log in again.
		return nil, deleteAuthFromKeyring()
	}
	return &auth, nil
}

func deleteAuthFromKeyring() error {
	return keyring.Delete(keyringService, authKey)
}
