package video

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRevalidatesRevisionAndContainment(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "film.mp4")
	if err := os.WriteFile(path, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("movies", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	file, opened, err := store.Open(item.ID)
	if err != nil || opened.ID != item.ID {
		t.Fatalf("open indexed video: %#v %v", opened, err)
	}
	file.Close()

	outside := filepath.Join(t.TempDir(), "outside.mp4")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, _, err := store.Open(item.ID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("symlink replacement should be unavailable, got %v", err)
	}
}

func TestOpenRejectsFileChangedSinceSnapshot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "film.mp4")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("movies", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	if err := os.WriteFile(path, []byte("replacement with another size"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Open(item.ID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("changed file should require a rescan, got %v", err)
	}
}
