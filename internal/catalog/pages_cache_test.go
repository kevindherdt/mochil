package catalog

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writePageArchive(t *testing.T, path string, pages [][2]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, page := range pages {
		entry, createErr := writer.Create(page[0])
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte(page[1])); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCBZPageOrderIsCachedAndRefreshedWhenTheArchiveChanges(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "comic.cbz")
	writePageArchive(t, path, [][2]string{{"page2.jpg", "two"}, {"page1.jpg", "one"}})
	store := NewStore(directory, filepath.Join(t.TempDir(), "cache.json"))
	book := Book{ID: "comic", Path: "comic.cbz", Format: "cbz", PageCount: 2}
	read := func(index int) string {
		t.Helper()
		page, err := store.Page(context.Background(), book, index)
		if err != nil {
			t.Fatal(err)
		}
		defer page.Reader.Close()
		data, err := io.ReadAll(page.Reader)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	if got := read(0); got != "one" {
		t.Fatalf("first page = %q", got)
	}
	if got := read(1); got != "two" {
		t.Fatalf("second page = %q", got)
	}
	// The second read is the cached path: same archive, same order.
	if got := read(0); got != "one" {
		t.Fatalf("cached first page = %q", got)
	}

	// Rewriting the archive changes its size and its modification time, so the
	// cached order must be discarded rather than applied to the new file.
	writePageArchive(t, path, [][2]string{{"page1.jpg", "ONE!"}, {"page2.jpg", "TWO!"}})
	newer := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, newer, newer); err != nil {
		t.Fatal(err)
	}
	if got := read(0); got != "ONE!" {
		t.Fatalf("page after rewrite = %q -- a stale cached order was used", got)
	}
}

func TestNaturalLessASCIIAgreesWithTheRuneComparator(t *testing.T) {
	for _, pair := range [][2]string{
		{"page2.jpg", "page10.jpg"},
		{"Page2", "page10"},
		{"page01", "page1"},
		{"a", "A"},
		{"page", "page2"},
		{"", ""},
		{"", "a"},
		{"z10", "z9"},
	} {
		if got, want := naturalLessASCII(pair[0], pair[1]), naturalLessRunes(pair[0], pair[1]); got != want {
			t.Fatalf("naturalLessASCII(%q, %q) = %v, rune comparator = %v", pair[0], pair[1], got, want)
		}
	}
	// Non-ASCII names never take the byte path, so the rune-aware order still
	// decides them.
	if naturalLess("é1.jpg", "e2.jpg") != naturalLessRunes("é1.jpg", "e2.jpg") {
		t.Fatal("a non-ASCII name did not fall back to the rune comparator")
	}
}
