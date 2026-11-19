package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/accounts"
	"mochil/internal/appconfig"
)

func TestAndroidHermesIsForAdministratorDevicesOnly(t *testing.T) {
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer hermes.Close()
	directory := t.TempDir()
	store, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := store.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := store.Authenticate("owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	devices, err := newAndroidDeviceStore(filepath.Join(directory, "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{accounts: store, androidDevices: devices, cfg: MultiConfig{Username: "owner", Hermes: appconfig.Hermes{URL: hermes.URL + "/", Key: "secret"}}}
	readerToken, _, _ := devices.issue(reader.ID, "Reader phone")
	ownerToken, _, _ := devices.issue(owner.ID, "Owner phone")

	call := func(token, method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		m.androidAPI(w, r)
		return w
	}
	chat := `{"session":"c1234567890abcdef","message":"ping"}`
	if got := call("", http.MethodPost, "/api/android/v1/hermes/chat", chat); got.Code != http.StatusUnauthorized {
		t.Fatalf("no device token: %d", got.Code)
	}
	for _, path := range []string{"/api/android/v1/hermes/status", "/api/android/v1/hermes/chats", "/api/android/v1/hermes/memories"} {
		if got := call(readerToken, http.MethodGet, path, ""); got.Code != http.StatusForbidden || strings.Contains(got.Body.String(), "secret") {
			t.Fatalf("ordinary account device on %s: %d %s", path, got.Code, got.Body.String())
		}
	}
	if got := call(readerToken, http.MethodPost, "/api/android/v1/hermes/chat", chat); got.Code != http.StatusForbidden {
		t.Fatalf("ordinary account chat: %d", got.Code)
	}
	if got := call(ownerToken, http.MethodPost, "/api/android/v1/hermes/chat", chat); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"hi"`) {
		t.Fatalf("administrator chat: %d %s", got.Code, got.Body.String())
	}
	status := call(ownerToken, http.MethodGet, "/api/android/v1/hermes/status", "")
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), `"configured":true`) || strings.Contains(status.Body.String(), "secret") {
		t.Fatalf("administrator status: %d %s", status.Code, status.Body.String())
	}
}

func TestAndroidAvatarIsForAdministratorDevicesOnly(t *testing.T) {
	avatars, root := avatarFixture(t)
	directory := t.TempDir()
	store, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := store.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := store.Authenticate("owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	devices, err := newAndroidDeviceStore(filepath.Join(directory, "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{accounts: store, androidDevices: devices, cfg: MultiConfig{Username: "owner", Avatar: appconfig.Avatar{Directory: root}, AvatarState: avatars.cfg.AvatarState}}
	readerToken, _, _ := devices.issue(reader.ID, "Reader phone")
	ownerToken, _, _ := devices.issue(owner.ID, "Owner phone")
	get := func(token, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		m.androidAPI(w, r)
		return w
	}
	if got := get(readerToken, "/api/android/v1/avatar/models"); got.Code != http.StatusForbidden {
		t.Fatalf("ordinary account device: %d", got.Code)
	}
	if got := get(readerToken, "/api/android/v1/avatar/core"); got.Code != http.StatusForbidden {
		t.Fatalf("ordinary account device, core: %d", got.Code)
	}
	if got := get(ownerToken, "/api/android/v1/avatar/models"); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"models"`) {
		t.Fatalf("administrator device: %d %s", got.Code, got.Body.String())
	}
}
