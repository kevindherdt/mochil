package video

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenSidecarForKeepsShowAndSeasonScopesSeparate(t *testing.T) {
	root := t.TempDir()
	seasonDirectory := filepath.Join(root, "Example Show", "Season 01")
	if err := os.MkdirAll(seasonDirectory, 0o750); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		filepath.Join(root, "Example Show", "poster.jpg"): "show",
		filepath.Join(seasonDirectory, "poster.jpg"):      "season",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	item, ok := Identify("tv", "Example Show/Season 01/Example Show S01E01.mkv", 5, time.Now())
	if !ok {
		t.Fatal("not identified")
	}
	InferEpisode(&item)
	store := NewStore("tv", root, "")
	store.publish([]Item{item})

	assertSidecarContents(t, store, item.ID, ArtworkShow, "show")
	assertSidecarContents(t, store, item.ID, ArtworkSeason, "season")
	if file, _, found := store.OpenSidecarFor(item.ID, ArtworkItem); found {
		file.Close()
		t.Fatal("episode item scope reused season artwork")
	}
}

func TestOpenSidecarForEpisodeUsesFileSpecificArtwork(t *testing.T) {
	root := t.TempDir()
	seasonDirectory := filepath.Join(root, "Example Show", "Season 01")
	if err := os.MkdirAll(seasonDirectory, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seasonDirectory, "Example Show S01E01.jpg"), []byte("episode"), 0o640); err != nil {
		t.Fatal(err)
	}
	item, _ := Identify("tv", "Example Show/Season 01/Example Show S01E01.mkv", 5, time.Now())
	InferEpisode(&item)
	store := NewStore("tv", root, "")
	store.publish([]Item{item})
	assertSidecarContents(t, store, item.ID, ArtworkItem, "episode")
}

func TestOpenSidecarForSeasonSupportsShowRootConvention(t *testing.T) {
	root := t.TempDir()
	seasonDirectory := filepath.Join(root, "Example Show", "Season 01")
	if err := os.MkdirAll(seasonDirectory, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Example Show", "season01-poster.webp"), []byte("season-root"), 0o640); err != nil {
		t.Fatal(err)
	}
	item, _ := Identify("tv", "Example Show/Season 01/Example Show S01E01.mkv", 5, time.Now())
	InferEpisode(&item)
	store := NewStore("tv", root, "")
	store.publish([]Item{item})
	assertSidecarContents(t, store, item.ID, ArtworkSeason, "season-root")
}

func TestOpenSidecarForShowDoesNotFallIntoSeason(t *testing.T) {
	root := t.TempDir()
	seasonDirectory := filepath.Join(root, "Example Show", "Season 01")
	if err := os.MkdirAll(seasonDirectory, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seasonDirectory, "poster.jpg"), []byte("season"), 0o640); err != nil {
		t.Fatal(err)
	}
	item, _ := Identify("tv", "Example Show/Season 01/Example Show S01E01.mkv", 5, time.Now())
	InferEpisode(&item)
	store := NewStore("tv", root, "")
	store.publish([]Item{item})
	if file, _, found := store.OpenSidecarFor(item.ID, ArtworkShow); found {
		file.Close()
		t.Fatal("show scope used season artwork")
	}
}

func assertSidecarContents(t *testing.T, store *Store, id string, scope ArtworkScope, want string) {
	t.Helper()
	file, _, found := store.OpenSidecarFor(id, scope)
	if !found {
		t.Fatalf("scope %d sidecar not found", scope)
	}
	defer file.Close()
	contents, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != want {
		t.Fatalf("scope %d contents = %q, want %q", scope, contents, want)
	}
}
