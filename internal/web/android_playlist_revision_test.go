package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/audio"
)

func TestAndroidPlaylistRevisionSurvivesAuthenticatedForwarding(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	devices, err := newAndroidDeviceStore(filepath.Join(t.TempDir(), "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.androidDevices = devices
	account := m.readingAccount(nil)
	token, id, err := devices.issue(account, "Fixture Android")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.audioPlaylists.Create(account, "Saved"); err != nil {
		t.Fatal(err)
	}
	request := func(method, credential, revision string) *httptest.ResponseRecorder {
		t.Helper()
		body := "[]"
		if method == http.MethodPatch {
			body = `{"name":"Saved"}`
		}
		r := httptest.NewRequest(method, "/api/android/v1/audio-playlists/Saved", strings.NewReader(body))
		if credential != "" {
			r.Header.Set("Authorization", "Bearer "+credential)
		}
		if revision != "" {
			r.Header.Set("If-Match", revision)
		}
		w := httptest.NewRecorder()
		m.androidAPI(w, r)
		return w
	}
	if got := request(http.MethodGet, "", ""); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized GET: %d", got.Code)
	}
	detail := request(http.MethodGet, token, "")
	revision := detail.Header().Get("ETag")
	if detail.Code != http.StatusOK || revision == "" {
		t.Fatalf("detail: %d", detail.Code)
	}
	if detail.Header().Get("X-Mochil-Playlist-Edit") != "visible-v1" {
		t.Fatal("capability lost in Android forwarding")
	}
	if err := m.audioPlaylists.Add(account, "Saved", audio.QueueEntry{LibraryID: "music", ItemID: "other-device"}); err != nil {
		t.Fatal(err)
	}
	if got := request(http.MethodPut, token, revision); got.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale PUT: %d", got.Code)
	}
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		if got := request(method, token, revision); got.Code != http.StatusPreconditionFailed {
			t.Fatalf("stale %s: %d", method, got.Code)
		}
	}
	current, _ := m.audioPlaylists.Get(account, "Saved")
	if len(current.Entries) != 1 {
		t.Fatal("stale Android PUT erased append")
	}
	latest := request(http.MethodGet, token, "")
	visible := httptest.NewRequest(http.MethodPut, "/api/android/v1/audio-playlists/Saved?edit=visible", strings.NewReader(`{"visible_entries":[],"entries":[]}`))
	visible.Header.Set("Authorization", "Bearer "+token)
	visible.Header.Set("If-Match", latest.Header().Get("ETag"))
	visibleResponse := httptest.NewRecorder()
	m.androidAPI(visibleResponse, visible)
	if visibleResponse.Code != http.StatusNoContent {
		t.Fatalf("visible edit: %d", visibleResponse.Code)
	}
	preserved, _ := m.audioPlaylists.Get(account, "Saved")
	if len(preserved.Entries) != 1 {
		t.Fatal("Android visible edit lost unseen track")
	}
	if got := request(http.MethodPut, token, latest.Header().Get("ETag")); got.Code != http.StatusNoContent || got.Header().Get("ETag") != revision {
		t.Fatalf("fresh PUT: %d", got.Code)
	}
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		if got := request(method, token, revision); got.Code != http.StatusNoContent {
			t.Fatalf("fresh %s: %d", method, got.Code)
		}
	}
	if err := devices.revoke(account, id); err != nil {
		t.Fatal(err)
	}
	if got := request(http.MethodPut, token, revision); got.Code != http.StatusUnauthorized {
		t.Fatalf("revoked PUT: %d", got.Code)
	}
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		if got := request(method, token, revision); got.Code != http.StatusUnauthorized {
			t.Fatalf("revoked %s: %d", method, got.Code)
		}
	}
}
