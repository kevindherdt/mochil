package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func sidecarStore(t *testing.T) (*Store, Book, string) {
	t.Helper()
	root := t.TempDir()
	directory := filepath.Join(root, "Series")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "Book 1.epub"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return &Store{library: root}, Book{Path: "Series/Book 1.epub"}, directory
}

func TestSidecarCoverPrefersTheBookNamedImageThenCoverJpg(t *testing.T) {
	store, book, directory := sidecarStore(t)
	if _, err := store.SidecarCover(context.Background(), book); err == nil {
		t.Fatal("no image beside the book should be an error")
	}
	os.WriteFile(filepath.Join(directory, "cover.jpg"), []byte("shared"), 0o644)
	if data, err := store.SidecarCover(context.Background(), book); err != nil || string(data) != "shared" {
		t.Fatalf("cover.jpg: %q %v", data, err)
	}
	os.WriteFile(filepath.Join(directory, "Book 1.png"), []byte("own"), 0o644)
	if data, err := store.SidecarCover(context.Background(), book); err != nil || string(data) != "own" {
		t.Fatalf("book-named image should win: %q %v", data, err)
	}
}

func TestSidecarCoverRefusesSymbolicLinksAndEmptyFiles(t *testing.T) {
	store, book, directory := sidecarStore(t)
	outside := filepath.Join(t.TempDir(), "secret.jpg")
	os.WriteFile(outside, []byte("secret"), 0o644)
	if err := os.Symlink(outside, filepath.Join(directory, "cover.jpg")); err != nil {
		t.Skip("symlinks unavailable")
	}
	os.WriteFile(filepath.Join(directory, "Book 1.jpg"), nil, 0o644)
	if data, err := store.SidecarCover(context.Background(), book); err == nil {
		t.Fatalf("a symlink or empty file must not be served: %q", data)
	}
}
