package web

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/audio"
)

func newAudioCollectionTestServer(t *testing.T) *multiServer {
	t.Helper()
	queue, err := audio.OpenQueueStore("")
	if err != nil {
		t.Fatal(err)
	}
	favorites, err := audio.OpenQueueStore("")
	if err != nil {
		t.Fatal(err)
	}
	playlists, err := audio.OpenPlaylistStore("")
	if err != nil {
		t.Fatal(err)
	}
	return &multiServer{
		cfg:            MultiConfig{Username: "reader"},
		libraries:      []Library{{ID: "music", Kind: "music", Audio: audio.NewStore("music", t.TempDir(), "")}},
		audioQueue:     queue,
		audioFavorites: favorites,
		audioPlaylists: playlists,
	}
}

func TestAudioPlaylistDownloadStreamsStoredTracks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "First.mp3"), []byte("first track"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Second.flac"), []byte("second track"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := audio.NewStore("music", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	playlists, err := audio.OpenPlaylistStore("")
	if err != nil {
		t.Fatal(err)
	}
	items := store.Items()
	entries := make([]audio.QueueEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, audio.QueueEntry{LibraryID: "music", ItemID: item.ID})
	}
	if err = playlists.Put("reader", "Road / trip", entries); err != nil {
		t.Fatal(err)
	}
	m := &multiServer{cfg: MultiConfig{Username: "reader"}, libraries: []Library{{ID: "music", Kind: "music", Audio: store}}, audioPlaylists: playlists}
	response := httptest.NewRecorder()
	m.apiAudioPlaylists(response, httptest.NewRequest(http.MethodGet, "/api/audio-playlists/Road%20%2F%20trip?download=1", nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/zip" || !strings.Contains(response.Header().Get("Content-Disposition"), "Road trip.zip") {
		t.Fatalf("download=%d type=%q disposition=%q body=%q", response.Code, response.Header().Get("Content-Type"), response.Header().Get("Content-Disposition"), response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != 2 {
		t.Fatalf("entries=%d", len(archive.File))
	}
	for _, file := range archive.File {
		if file.Method != zip.Store || !strings.HasPrefix(file.Name, "Road trip/") || strings.Contains(file.Name, "../") {
			t.Fatalf("unsafe or compressed entry: %#v", file.FileHeader)
		}
	}
}

func TestAudioCollectionQueueStartsEmpty(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	response := httptest.NewRecorder()
	m.apiAudioQueue(response, httptest.NewRequest(http.MethodGet, "/api/audio-queue", nil))
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != "[]" {
		t.Fatalf("queue=%d %s", response.Code, response.Body.String())
	}
}

func TestAudioCollectionCreatesAndListsPlaylist(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	create := httptest.NewRecorder()
	m.apiAudioPlaylists(create, httptest.NewRequest(http.MethodPost, "/api/audio-playlists", strings.NewReader(`{"name":"Commute"}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", create.Code, create.Body.String())
	}

	list := httptest.NewRecorder()
	m.apiAudioPlaylists(list, httptest.NewRequest(http.MethodGet, "/api/audio-playlists", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"name":"Commute"`) || !strings.Contains(list.Body.String(), `"items":[]`) {
		t.Fatalf("list=%d %s", list.Code, list.Body.String())
	}
}

func TestAudioCollectionRejectsUnknownFavorite(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	response := httptest.NewRecorder()
	body := `{"library_id":"music","item_id":"missing","favorite":true}`
	m.apiAudioFavorites(response, httptest.NewRequest(http.MethodPut, "/api/audio-favorites", strings.NewReader(body)))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Unknown music item") {
		t.Fatalf("favorite=%d %s", response.Code, response.Body.String())
	}
}

func TestAudioCollectionPreservesInvalidPlaylistEntryError(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	response := httptest.NewRecorder()
	m.apiAudioPlaylists(response, httptest.NewRequest(http.MethodPost, "/api/audio-playlists/Commute", strings.NewReader(`{`)))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Invalid playlist entry") {
		t.Fatalf("playlist entry=%d %s", response.Code, response.Body.String())
	}
}
