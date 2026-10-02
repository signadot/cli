package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/signadot/cli/internal/utils/system"
)

const (
	credentialsFileName = "credentials"
)

func storeAuthInPlainText(auth *Auth) error {
	// Get the sigandot dir and ensure it exists
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return err
	}
	if err := system.CreateDirIfNotExist(signadotDir); err != nil {
		return err
	}

	credentialsPath := filepath.Join(signadotDir, credentialsFileName)

	authJson, err := json.Marshal(auth)
	if err != nil {
		return err
	}

	// Write the credentials file with restricted permissions (0600),
	// atomically: concurrent readers (other CLI processes, the MCP server,
	// locald refreshing tokens) must never see a partial file.
	tmp, err := os.CreateTemp(signadotDir, credentialsFileName+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(authJson); err != nil {
		tmp.Close()
		return err
	}
	// on disk before it is renamed into place, so a crash cannot leave an
	// empty credentials file behind the rename
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), credentialsPath)
}

// ErrInvalidCredentialsFile is a plain-text credentials file that cannot be
// read as credentials. It is kept, since it may be valid credentials this CLI
// failed to read; logging out removes it.
var ErrInvalidCredentialsFile = errors.New("invalid credentials file")

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
		// Don't delete the file: it may be valid credentials we failed to
		// read (e.g. an older CLI writing it non-atomically).
		return nil, fmt.Errorf("%w %s (run 'signadot auth login' to replace it): %w",
			ErrInvalidCredentialsFile, credentialsPath, err)
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
