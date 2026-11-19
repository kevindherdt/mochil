package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"mochil/internal/audio"
)

func TestAudioPlaylistDetailNamesAndAndroidForwarding(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	for _, name := range []string{"Road / trip", "100%", "100%25", "Literal %2F", "日本語 + mix", "Trailing/"} {
		if err := m.audioPlaylists.Put("reader", name, nil); err != nil {
			t.Fatal(err)
		}
		for _, android := range []bool{false, true} {
			response := httptest.NewRecorder()
			if android {
				request := httptest.NewRequest(http.MethodGet, "/api/android/v1/audio-playlists/"+url.PathEscape(name), nil)
				forwardAndroidPath(request, "/api/android/v1/audio-playlists/", "/api/audio-playlists/", m.apiAudioPlaylists, response)
			} else {
				m.apiAudioPlaylists(response, httptest.NewRequest(http.MethodGet, "/api/audio-playlists/"+url.PathEscape(name), nil))
			}
			var got audioPlaylistView
			if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &got) != nil || got.Name != name || got.Items == nil || len(got.Items) != 0 {
				t.Fatalf("name=%q android=%v response=%d %s", name, android, response.Code, response.Body.String())
			}
		}
	}
	if err := m.audioPlaylists.Put("someone-else", "Private", nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Private", "Missing"} {
		response := httptest.NewRecorder()
		m.apiAudioPlaylists(response, httptest.NewRequest(http.MethodGet, "/api/audio-playlists/"+name, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s returned %d", name, response.Code)
		}
	}
}

func TestAudioPlaylistDetailMatchesCollectionOrderAndFiltering(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	root := t.TempDir()
	for _, name := range []string{"First.mp3", "Second.mp3"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("synthetic track"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	store := audio.NewStore("music", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	m.libraries[0].Audio = store
	items := store.Items()
	if len(items) != 2 {
		t.Fatalf("items=%d", len(items))
	}
	entries := []audio.QueueEntry{{LibraryID: "music", ItemID: items[1].ID}, {LibraryID: "removed", ItemID: "gone"}, {LibraryID: "music", ItemID: "missing"}, {LibraryID: "music", ItemID: items[0].ID}}
	if err := m.audioPlaylists.Put("reader", "Ordered", entries); err != nil {
		t.Fatal(err)
	}
	detail := httptest.NewRecorder()
	m.apiAudioPlaylists(detail, httptest.NewRequest(http.MethodGet, "/api/audio-playlists/ordered", nil))
	var got audioPlaylistView
	if err := json.Unmarshal(detail.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 || got.Items[0].ID != items[1].ID || got.Items[1].ID != items[0].ID {
		t.Fatalf("wrong order: %#v", got)
	}
	collection := httptest.NewRecorder()
	m.apiAudioPlaylists(collection, httptest.NewRequest(http.MethodGet, "/api/audio-playlists", nil))
	var lists []audioPlaylistView
	if err := json.Unmarshal(collection.Body.Bytes(), &lists); err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 || !reflect.DeepEqual(got, lists[0]) {
		t.Fatal("detail and collection disagree")
	}
}
