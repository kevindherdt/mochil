package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/accounts"
)

// The website refuses server-wide changes to ordinary accounts; the Android routes reach the same handlers and must refuse
// them in the same way, not only the Hermes ones.
func TestAndroidDevicesOfOrdinaryAccountsCannotChangeServerWideSettings(t *testing.T) {
	directory := t.TempDir()
	store, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := store.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := store.Authenticate("owner", "owner password")
	devices, err := newAndroidDeviceStore(filepath.Join(directory, "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{androidDevices: devices, cfg: MultiConfig{Username: "owner"}}
	m.accountsStoreField.Store(store)
	readerToken, _, _ := devices.issue(reader.ID, "Reader phone")
	ownerToken, _, _ := devices.issue(owner.ID, "Owner phone")
	call := func(token, method, path string) (code int) {
		defer func() {
			if recover() != nil {
				code = http.StatusInternalServerError // the handler ran against a bare test server: it got past the gate
			}
		}()
		r := httptest.NewRequest(method, path, strings.NewReader(`{}`))
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		m.androidAPI(w, r)
		return w.Code
	}
	for _, change := range [][2]string{
		{http.MethodPost, "/api/android/v1/podcast-feeds"},
		{http.MethodDelete, "/api/android/v1/podcast-feeds"},
		{http.MethodPost, "/api/android/v1/narration/default-voice"},
		{http.MethodPut, "/api/android/v1/narration/default-voice"},
		{http.MethodDelete, "/api/android/v1/narration/voices/someone"},
		{http.MethodPatch, "/api/android/v1/narration/voices/someone"},
		{http.MethodPost, "/api/android/v1/narration/transcribe"},
		{http.MethodPost, "/api/android/v1/narration/voices"},
		{http.MethodPut, "/api/android/v1/vaults/notes/pages/a.md"},
	} {
		if got := call(readerToken, change[0], change[1]); got != http.StatusForbidden {
			t.Errorf("ordinary device %s %s = %d, want 403", change[0], change[1], got)
		}
		if got := call(ownerToken, change[0], change[1]); got == http.StatusForbidden {
			t.Errorf("administrator device %s %s was refused", change[0], change[1])
		}
	}
}
