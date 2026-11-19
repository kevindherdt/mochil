package video

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenSidecarUsesEpisodeSpecificArtwork(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "Show", "Season 01")
	if err := os.MkdirAll(folder, 0o750); err != nil {
		t.Fatal(err)
	}
	videoPath := filepath.Join(folder, "episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Show", "poster.jpg"), []byte("show"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "episode.jpg"), []byte("episode"), 0o640); err != nil {
		t.Fatal(err)
	}
	store := NewStore("tv", root, "")
	item, ok := Identify("tv", "Show/Season 01/episode.mkv", 5, time.Now())
	if !ok {
		t.Fatal("not identified")
	}
	item.Show = "Show"
	item.Season = 1
	store.publish([]Item{item})
	file, extension, found := store.OpenSidecar(item.ID)
	if !found {
		t.Fatal("sidecar not found")
	}
	defer file.Close()
	data, _ := os.ReadFile(file.Name())
	if extension != ".jpg" || string(data) != "episode" {
		t.Fatalf("extension=%s data=%q", extension, data)
	}
}

func TestArtworkKeyChangesWithRevision(t *testing.T) {
	store := &ArtworkStore{}
	first := Item{LibraryID: "a", ID: "b", Revision: "one"}
	second := first
	second.Revision = "two"
	if store.key(first) == store.key(second) {
		t.Fatal("revision did not change artwork key")
	}
}

func TestPosterSeekAvoidsSharedEpisodeIntroFrame(t *testing.T) {
	if got := posterSeekSeconds(1_480); got != 148 {
		t.Fatalf("episode seek = %v", got)
	}
	if got := posterSeekSeconds(20); got != 5 {
		t.Fatalf("short seek = %v", got)
	}
	if got := posterSeekSeconds(10_000); got != 600 {
		t.Fatalf("long seek = %v", got)
	}
}
