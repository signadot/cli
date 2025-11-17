package auth

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/signadot/cli/internal/utils/system"
)

const (
	credentialsFileName = "credentials"
)

func storeAuthInPlainText(auth *Auth) error {
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return err
	}

	// Ensure the directory exists
	if err := system.CreateDirIfNotExist(signadotDir); err != nil {
		return err
	}

	credentialsPath := filepath.Join(signadotDir, credentialsFileName)

	authJson, err := json.Marshal(auth)
	if err != nil {
		return err
	}

	// Write credentials file with restricted permissions (0600)
	return os.WriteFile(credentialsPath, authJson, 0600)
}

func getAuthFromPlainText() (*Auth, error) {
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return nil, err
	}

	credentialsPath := filepath.Join(signadotDir, credentialsFileName)

	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		return nil, nil
	}

	authJson, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, err
	}

	var auth Auth
	if err := json.Unmarshal(authJson, &auth); err != nil {
		// If unmarshaling fails, remove the corrupted file
		_ = deleteAuthFromPlainText()
		return nil, err
	}

	return &auth, nil
}

func deleteAuthFromPlainText() error {
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return err
	}

	credentialsPath := filepath.Join(signadotDir, credentialsFileName)

	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(credentialsPath)
}
