package accounts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenBootstrapsLegacyAccountKeyAsAdministrator(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	store, err := Open(path, "Owner", "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := store.Authenticate(" owner ", "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if identity.ID != "Owner" || identity.Username != "Owner" || !identity.Administrator() {
		t.Fatalf("identity=%+v", identity)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "correct horse battery staple") {
		t.Fatal("account state persisted a plaintext password")
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode/error=%v %v", info.Mode().Perm(), err)
	}
}

func TestOpenReloadsWithoutReplacingPersistedPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	if _, err := Open(path, "owner", "first password"); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path, "owner", "different environment password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate("owner", "first password"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate("owner", "different environment password"); err == nil {
		t.Fatal("reload silently replaced persisted password")
	}
}

func TestOpenRejectsMalformedDuplicateAndAdministratorlessState(t *testing.T) {
	for name, body := range map[string]string{
		"trailing":  `{"version":1,"accounts":[]} trailing`,
		"duplicate": `{"version":1,"accounts":[{"id":"one","username":"Owner","normalized_username":"owner","password_hash":"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy","role":"administrator"},{"id":"two","username":"owner","normalized_username":"owner","password_hash":"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy","role":"user"}]}`,
		"no_admin":  `{"version":1,"accounts":[{"id":"one","username":"reader","normalized_username":"reader","password_hash":"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy","role":"user"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "accounts.json")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(path, "owner", "secret"); err == nil {
				t.Fatal("malformed account state was accepted")
			}
		})
	}
}

func TestAccountLifecyclePreservesStableIdentityAndLastAdministrator(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	store, err := Open(path, "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := store.Create("Reader", "reader password", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if reader.ID == "Reader" || reader.ID == "owner" {
		t.Fatalf("new account received a mutable ID: %+v", reader)
	}
	if _, err := store.Create("reader", "another password", RoleUser); err != ErrConflict {
		t.Fatalf("duplicate error=%v", err)
	}
	if err := store.SetDisabled("owner", true); err != ErrLastAdministrator {
		t.Fatalf("last administrator disable error=%v", err)
	}
	if err := store.SetDisabled(reader.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate("Reader", "reader password"); err != ErrInvalidCredentials {
		t.Fatalf("disabled authentication error=%v", err)
	}
	if err := store.SetDisabled(reader.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := store.ResetPassword(reader.ID, "replacement password"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate("Reader", "reader password"); err != ErrInvalidCredentials {
		t.Fatal("old password remained valid")
	}
	reloaded, err := Open(path, "owner", "ignored password")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := reloaded.Authenticate("reader", "replacement password")
	if err != nil || identity.ID != reader.ID {
		t.Fatalf("reloaded identity=%+v error=%v", identity, err)
	}
}

func TestEnvironmentRecoveryRequiresPersistedAdministratorPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	store, err := Open(path, "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	if !store.RecoveryEnabled() {
		t.Fatal("new migration did not retain its recovery path")
	}
	if err := store.ConfirmRecovery("owner", "wrong password"); err != ErrInvalidCredentials {
		t.Fatalf("wrong confirmation error=%v", err)
	}
	if err := store.ConfirmRecovery("owner", "owner password"); err != nil {
		t.Fatal(err)
	}
	if store.RecoveryEnabled() {
		t.Fatal("confirmed recovery remained enabled")
	}
	reloaded, err := Open(path, "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.RecoveryEnabled() {
		t.Fatal("disabled recovery was not durable")
	}
}

func TestDeleteDisabledAccountIsDurableAndRequiresDisabledState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	store, err := Open(path, "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := store.Create("reader", "reader password", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.DeleteDisabled(reader.ID); !errors.Is(err, ErrAccountMustBeDisabled) {
		t.Fatalf("active delete = %v", err)
	}
	if err = store.SetDisabled(reader.ID, true); err != nil {
		t.Fatal(err)
	}
	if err = store.DeleteDisabled(reader.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.IdentityByUsername("reader"); ok {
		t.Fatal("deleted reader still resolves")
	}
	reopened, err := Open(path, "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.List()) != 1 || reopened.List()[0].Username != "owner" {
		t.Fatalf("accounts = %#v", reopened.List())
	}
}
