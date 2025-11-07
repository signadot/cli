package auth

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
