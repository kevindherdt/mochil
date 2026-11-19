package web

import (
	"mochil/internal/audio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVisiblePlaylistEditRetainsUnavailableReferences(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	original := []audio.QueueEntry{{LibraryID: "music", ItemID: "was-visible"}, {LibraryID: "removed-library", ItemID: "unseen"}}
	if err := m.audioPlaylists.Put("reader", "Saved", original); err != nil {
		t.Fatal(err)
	}
	list, _ := m.audioPlaylists.Get("reader", "Saved")
	revision := audio.PlaylistRevision(list)
	get := httptest.NewRecorder()
	m.audioPlaylist(get, httptest.NewRequest(http.MethodGet, "/api/audio-playlists/Saved", nil), "reader", "Saved")
	if get.Header().Get("X-Mochil-Playlist-Edit") != "visible-v1" {
		t.Fatal("missing visible edit capability")
	}
	body := `{"visible_entries":[{"library_id":"music","item_id":"was-visible"}],"entries":[]}`
	request := func(token, payload string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPut, "/api/audio-playlists/Saved?edit=visible", strings.NewReader(payload))
		r.Header.Set("If-Match", token)
		w := httptest.NewRecorder()
		m.audioPlaylist(w, r, "reader", "Saved")
		return w
	}
	if w := request("", body); w.Code != 428 {
		t.Fatalf("missing condition: %d", w.Code)
	}
	if w := request(revision, `{"visible_entries":[],"entries":[{"library_id":"foreign","item_id":"new"}]}`); w.Code != 400 {
		t.Fatalf("unknown insert: %d", w.Code)
	}
	w := request(revision, body)
	if w.Code != 204 {
		t.Fatalf("edit: %d %s", w.Code, w.Body.String())
	}
	list, _ = m.audioPlaylists.Get("reader", "Saved")
	if len(list.Entries) != 1 || list.Entries[0] != original[1] {
		t.Fatalf("unseen reference lost: %v", list.Entries)
	}
	if w.Header().Get("ETag") != audio.PlaylistRevision(list) {
		t.Fatal("wrong stored revision")
	}
	if w := request(revision, body); w.Code != 412 {
		t.Fatalf("stale edit: %d", w.Code)
	}
}
