package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/opf"
)

// The writer's whole job is to write inside one folder and nowhere else. These are the ways out of a folder that a
// path can take, and every one of them has to be refused before anything is written.
func TestLibraryWriterStaysInsideItsLibrary(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Dune"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Dune", "Dune.epub"), []byte("book"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "out.epub"), []byte("book"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A symlink inside the library that points out of it: the path looks local and is not.
	if err := os.Symlink(filepath.Join(outside, "out.epub"), filepath.Join(root, "Dune", "escape.epub")); err != nil {
		t.Fatal(err)
	}
	writer, err := NewLibraryWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	meta := opf.Metadata{Title: "Dune", Authors: []string{"Frank Herbert"}}
	for _, path := range []string{"../out.epub", "/etc/passwd", `Dune\Dune.epub`, "Dune/escape.epub", "", "Dune/missing.epub"} {
		if _, err := writer.WriteSidecar(path, meta, true); err == nil {
			t.Errorf("%q was accepted", path)
		}
	}
	// A book is the only thing a sidecar is written for: a stray text file is not a book.
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.WriteSidecar("notes.txt", meta, true); err == nil || !strings.Contains(err.Error(), "books only") {
		t.Fatalf("a text file got a sidecar: %v", err)
	}
	// The escape link is a second book in the folder, so it is cleaned up before the ordinary case: one book in a folder
	// is the shape Calibre reads "metadata.opf" from.
	if err := os.Remove(filepath.Join(root, "Dune", "escape.epub")); err != nil {
		t.Fatal(err)
	}
	written, err := writer.WriteSidecar("Dune/Dune.epub", meta, false)
	if err != nil || written != "Dune/metadata.opf" {
		t.Fatalf("written=%q err=%v", written, err)
	}
	if _, err := os.Stat(filepath.Join(root, "Dune", "metadata.opf")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outside, "metadata.opf")); err == nil {
		t.Fatal("a sidecar was written outside the library")
	}
}
