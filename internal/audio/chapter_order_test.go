package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// quietProber returns no tags at all, which is the shape of a folder-inferred audiobook: every file gets the book's own
// volume as its track, so the chapter title is the only thing that can order them.
type quietProber struct{}

func (quietProber) Probe(string) (Metadata, error) { return Metadata{}, nil }

func TestChaptersWithoutTrackTagsAreOrderedNaturally(t *testing.T) {
	root := t.TempDir()
	book := filepath.Join(root, "Some Book")
	if err := os.MkdirAll(book, 0o750); err != nil {
		t.Fatal(err)
	}
	for chapter := 1; chapter <= 12; chapter++ {
		path := filepath.Join(book, fmt.Sprintf("Chapter %d.mp3", chapter))
		if err := os.WriteFile(path, []byte("audio"), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore("audio", root, filepath.Join(t.TempDir(), "audio.json"))
	store.SetProber(quietProber{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	items := store.Items()
	if len(items) != 12 {
		t.Fatalf("scanned %d chapters, want 12", len(items))
	}
	// A plain string compare puts Chapter 10 straight after Chapter 1, which is how "next chapter" played the wrong file.
	for index, item := range items {
		want := fmt.Sprintf("Chapter %d", index+1)
		if item.Title != want {
			t.Fatalf("position %d is %q, want %q (order: %v)", index, item.Title, want, titles(items))
		}
	}
}

func titles(items []Item) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Title)
	}
	return out
}
