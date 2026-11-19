package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/audio"
)

func TestAudioPlaylistSummariesBoundArtworkAndPreserveCounts(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	root := t.TempDir()
	for i := 0; i < 20; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("Track %02d.mp3", i)), []byte("synthetic audio"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	store := audio.NewStore("music", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	m.libraries[0].Audio = store
	entries := []audio.QueueEntry{{LibraryID: "missing", ItemID: "gone"}, {LibraryID: "music", ItemID: "gone"}}
	for _, item := range store.Items() {
		entries = append(entries, audio.QueueEntry{LibraryID: "music", ItemID: item.ID})
	}
	if err := m.audioPlaylists.Put("reader", "Saved", entries); err != nil {
		t.Fatal(err)
	}
	if err := m.audioPlaylists.Put("reader", "Empty", nil); err != nil {
		t.Fatal(err)
	}
	if err := m.audioPlaylists.Put("another", "Private", entries); err != nil {
		t.Fatal(err)
	}
	full := httptest.NewRecorder()
	m.apiAudioPlaylists(full, httptest.NewRequest(http.MethodGet, "/api/audio-playlists", nil))
	var lists []audioPlaylistView
	if err := json.Unmarshal(full.Body.Bytes(), &lists); err != nil {
		t.Fatal(err)
	}
	for _, android := range []bool{false, true} {
		response := httptest.NewRecorder()
		if android {
			r := httptest.NewRequest(http.MethodGet, "/api/android/v1/audio-playlists?view=summary", nil)
			forwardAndroidPath(r, "/api/android/v1/audio-playlists", "/api/audio-playlists", m.apiAudioPlaylists, response)
		} else {
			m.apiAudioPlaylists(response, httptest.NewRequest(http.MethodGet, "/api/audio-playlists?view=summary", nil))
		}
		var got audioPlaylistSummaries
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &got) != nil {
			t.Fatalf("summary=%d %s", response.Code, response.Body.String())
		}
		if got.Version != 1 || len(got.Playlists) != 2 {
			t.Fatalf("wrong overview: %#v", got)
		}
		if strings.Contains(response.Body.String(), `"items"`) {
			t.Fatal("summary exposes items field")
		}
		for i, list := range got.Playlists {
			if list.Name != lists[i].Name || list.ItemCount != len(lists[i].Items) || list.ArtworkURLs == nil {
				t.Fatalf("summary mismatch: %#v", list)
			}
			if list.ItemCount == 0 && len(list.ArtworkURLs) != 0 {
				t.Fatal("empty playlist artwork")
			}
			if list.ItemCount > 4 && len(list.ArtworkURLs) != 4 {
				t.Fatal("artwork must be bounded at four")
			}
			for j, url := range list.ArtworkURLs {
				if url != lists[i].Items[j].ArtworkURL {
					t.Fatal("artwork order changed")
				}
			}
		}
		if response.Body.Len() >= full.Body.Len()/2 {
			t.Fatal("summary unexpectedly approaches full response size")
		}
		t.Logf("synthetic 20-track playlist plus empty playlist: full=%d bytes summary=%d bytes", full.Body.Len(), response.Body.Len())
	}
}

func TestAudioPlaylistSummariesEmptyAccount(t *testing.T) {
	m := newAudioCollectionTestServer(t)
	response := httptest.NewRecorder()
	m.apiAudioPlaylists(response, httptest.NewRequest(http.MethodGet, "/api/audio-playlists?view=summary", nil))
	if strings.TrimSpace(response.Body.String()) != `{"version":1,"playlists":[]}` {
		t.Fatalf("empty response: %s", response.Body.String())
	}
}
