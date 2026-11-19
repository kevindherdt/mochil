package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHoldsOnlyNarratedBooks(t *testing.T) {
	touch := func(root, name string) {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	epubs := t.TempDir()
	touch(epubs, "Author/Book.epub")
	if !holdsOnlyNarratedBooks(epubs) {
		t.Fatal("a folder of EPUBs only is a book shelf")
	}
	mixed := t.TempDir()
	touch(mixed, "Author/Book.epub")
	touch(mixed, "Other/Story.M4B")
	if holdsOnlyNarratedBooks(mixed) {
		t.Fatal("loose audio keeps it an audiobook library")
	}
	if holdsOnlyNarratedBooks(t.TempDir()) {
		t.Fatal("an empty folder stays as configured")
	}
}
