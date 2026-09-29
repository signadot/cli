package auth

import (
	"errors"
	"fmt"
)

type Storage interface {
	Store(auth *Auth) error
	Get() (*Auth, error)
	Delete() error
	Source() AuthSource
}

// KeyringStorage implements Storage using the system keyring
type KeyringStorage struct{}

func NewKeyringStorage() *KeyringStorage {
	return &KeyringStorage{}
}

func (k *KeyringStorage) Store(auth *Auth) error {
	return storeAuthInKeyring(auth)
}

func (k *KeyringStorage) Get() (*Auth, error) {
	return getAuthFromKeyring()
}

func (k *KeyringStorage) Delete() error {
	return deleteAuthFromKeyring()
}

func (k *KeyringStorage) Source() AuthSource {
	return KeyringAuthSource
}

// PlainTextStorage implements Storage using a plain text file
type PlainTextStorage struct{}

func NewPlainTextStorage() *PlainTextStorage {
	return &PlainTextStorage{}
}

func (p *PlainTextStorage) Store(auth *Auth) error {
	return storeAuthInPlainText(auth)
}

func (p *PlainTextStorage) Get() (*Auth, error) {
	return getAuthFromPlainText()
}

func (p *PlainTextStorage) Delete() error {
	return deleteAuthFromPlainText()
}

func (p *PlainTextStorage) Source() AuthSource {
	return PlainTextAuthSource
}

// StoreExclusive stores auth in s and removes any credentials held by the
// other storage, so that the stored credentials are the ones ResolveAuth
// returns (it prefers the keyring over the plain text file).
func StoreExclusive(s Storage, auth *Auth) error {
	if err := s.Store(auth); err != nil {
		return err
	}
	for _, other := range allStorages() {
		if other.Source() == s.Source() {
			continue
		}
		// best effort: the keyring may be unavailable (e.g. no dbus), in
		// which case it can't hold credentials that take precedence either.
		_ = other.Delete()
	}
	return nil
}

// DeleteAll removes credentials from all storages. Errors from the keyring
// are ignored when there is nothing to delete or it is unavailable.
func DeleteAll() error {
	var errs []error
	for _, s := range allStorages() {
		err := s.Delete()
		if err == nil {
			continue
		}
		if s.Source() == KeyringAuthSource {
			if a, gerr := s.Get(); gerr != nil || a == nil {
				// not found, or keyring unavailable
				continue
			}
		}
		errs = append(errs, fmt.Errorf("%s: %w", s.Source(), err))
	}
	return errors.Join(errs...)
}

// StorageFor returns the storage for a stored auth source, or nil for
// sources that are not stored by the CLI (e.g. config).
func StorageFor(source AuthSource) Storage {
	for _, s := range allStorages() {
		if s.Source() == source {
			return s
		}
	}
	return nil
}

func allStorages() []Storage {
	return []Storage{NewKeyringStorage(), NewPlainTextStorage()}
}
