package auth

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func setupStorages(t *testing.T) {
	t.Helper()
	keyring.MockInit()
	t.Setenv("HOME", t.TempDir())
}

func TestStoreExclusiveMakesNewLoginActive(t *testing.T) {
	setupStorages(t)
	if err := StoreExclusive(NewKeyringStorage(), &Auth{APIKey: "a", OrgName: "org-a"}); err != nil {
		t.Fatal(err)
	}
	if err := StoreExclusive(NewPlainTextStorage(), &Auth{APIKey: "b", OrgName: "org-b"}); err != nil {
		t.Fatal(err)
	}
	got, err := loadAuth()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.OrgName != "org-b" || got.Source != PlainTextAuthSource {
		t.Fatalf("got %+v, want plaintext org-b", got)
	}

	// and back
	if err := StoreExclusive(NewKeyringStorage(), &Auth{APIKey: "c", OrgName: "org-c"}); err != nil {
		t.Fatal(err)
	}
	if a, _ := NewPlainTextStorage().Get(); a != nil {
		t.Fatalf("plaintext credentials not removed: %+v", a)
	}
}

func TestDeleteAllLeavesNoCredentials(t *testing.T) {
	setupStorages(t)
	if err := NewKeyringStorage().Store(&Auth{APIKey: "a", OrgName: "org-a"}); err != nil {
		t.Fatal(err)
	}
	if err := NewPlainTextStorage().Store(&Auth{APIKey: "b", OrgName: "org-b"}); err != nil {
		t.Fatal(err)
	}
	if err := DeleteAll(); err != nil {
		t.Fatal(err)
	}
	got, err := loadAuth()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("still logged in after DeleteAll: %+v", got)
	}
	// deleting again is not an error
	if err := DeleteAll(); err != nil {
		t.Fatalf("second DeleteAll: %v", err)
	}
}
