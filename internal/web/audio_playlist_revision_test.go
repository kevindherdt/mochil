package web

import (
	"mochil/internal/audio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlaylistConditionalStructureHTTP(t *testing.T) {
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			m := newAudioCollectionTestServer(t)
			if err := m.audioPlaylists.Create("reader", "Saved"); err != nil {
				t.Fatal(err)
			}
			list, _ := m.audioPlaylists.Get("reader", "Saved")
			old := audio.PlaylistRevision(list)
			if err := m.audioPlaylists.Add("reader", "Saved", audio.QueueEntry{LibraryID: "music", ItemID: "new"}); err != nil {
				t.Fatal(err)
			}
			request := func(revision string) int {
				r := httptest.NewRequest(method, "/api/audio-playlists/Saved", strings.NewReader(`{"name":"Renamed"}`))
				r.Header.Set("If-Match", revision)
				w := httptest.NewRecorder()
				m.audioPlaylist(w, r, "reader", "Saved")
				return w.Code
			}
			if status := request(old); status != http.StatusPreconditionFailed {
				t.Fatalf("stale status %d", status)
			}
			list, found := m.audioPlaylists.Get("reader", "Saved")
			if !found || len(list.Entries) != 1 {
				t.Fatal("stale request lost playlist")
			}
			if status := request(audio.PlaylistRevision(list)); status != http.StatusNoContent {
				t.Fatalf("fresh status %d", status)
			}
		})
	}
}

func TestPlaylistConditionalReplaceHTTP(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	if err := m.audioPlaylists.Create("reader", "Saved"); err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRecorder()
	m.audioPlaylist(get, httptest.NewRequest("GET", "/api/audio-playlists/Saved", nil), "reader", "Saved")
	revision := get.Header().Get("ETag")
	if revision == "" {
		t.Fatal("missing revision")
	}
	// A second client appends a stored entry after the first client's read.
	if err := m.audioPlaylists.Add("reader", "Saved", audio.QueueEntry{LibraryID: "music", ItemID: "new"}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("PUT", "/api/audio-playlists/Saved", strings.NewReader("[]"))
	request.Header.Set("If-Match", revision)
	response := httptest.NewRecorder()
	m.audioPlaylist(response, request, "reader", "Saved")
	if response.Code != http.StatusPreconditionFailed {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	list, _ := m.audioPlaylists.Get("reader", "Saved")
	if len(list.Entries) != 1 {
		t.Fatal("concurrent append erased")
	}
	request = httptest.NewRequest("PUT", "/api/audio-playlists/Saved", strings.NewReader("[]"))
	request.Header.Set("If-Match", audio.PlaylistRevision(list))
	response = httptest.NewRecorder()
	m.audioPlaylist(response, request, "reader", "Saved")
	if response.Code != http.StatusNoContent || response.Header().Get("ETag") != revision {
		t.Fatalf("conditional replace: %d", response.Code)
	}
}
