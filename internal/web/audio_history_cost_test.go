package web

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mochil/internal/audio"
)

func audioHistoryFixture(t testing.TB, kind string, history int) *multiServer {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Episode.mp3"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	store := audio.NewStore("shelf", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	progress, err := audio.OpenProgressStore("")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < history; i++ {
		if err := progress.Put("default", audio.Progress{LibraryID: "shelf", ItemID: fmt.Sprint(i), Position: 20, Duration: 200}); err != nil {
			t.Fatal(err)
		}
	}
	if err := progress.Put("default", audio.Progress{LibraryID: "shelf", ItemID: store.Items()[0].ID, Position: 42, Duration: 200}); err != nil {
		t.Fatal(err)
	}
	return &multiServer{libraries: []Library{{ID: "shelf", Kind: kind, Audio: store}}, audioProgress: progress}
}

func TestAudioHistorySnapshotIsNotRequiredForVisiblePageProgress(t *testing.T) {
	for _, kind := range []string{"music", "audiobooks", "podcasts"} {
		m := audioHistoryFixture(t, kind, 0)
		for _, filter := range []string{"", "continue", "unplayed", "played"} {
			response := httptest.NewRecorder()
			m.apiAudio(response, httptest.NewRequest("GET", "/api/audio?library=shelf&played="+filter, nil))
			var items []audioView
			if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &items) != nil {
				t.Fatalf("%s/%s: %d %s", kind, filter, response.Code, response.Body.String())
			}
			if filter == "played" {
				if len(items) != 0 {
					t.Fatal("unfinished item matched played")
				}
				continue
			}
			if len(items) != 1 {
				t.Fatalf("%s/%s: expected one item", kind, filter)
			}
			if kind == "music" {
				if items[0].Progress != nil {
					t.Fatal("music exposed per-track resume progress")
				}
			} else if items[0].Progress == nil || items[0].Progress.Position != 42 {
				t.Fatal("spoken media lost visible progress")
			}
		}
	}
}

func BenchmarkAudioPlainPageWithHistory(b *testing.B) {
	m := audioHistoryFixture(b, "music", 5000)
	request := httptest.NewRequest("GET", "/api/audio?library=shelf", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := httptest.NewRecorder()
		m.apiAudio(response, request)
		if response.Code != 200 {
			b.Fatal(response.Code)
		}
	}
}
