package ingest

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/opf"
)

func TestSidecarsAreWrittenOnlyForBooksInsideTheManagedLibrary(t *testing.T) {
	executor, _, managed, _ := newTestExecutor(t, 1<<20)
	book := filepath.Join(managed, "Series", "Series - Vol 01.epub")
	os.MkdirAll(filepath.Dir(book), 0o755)
	os.WriteFile(book, []byte("book"), 0o644)
	meta := opf.Metadata{Title: "Vol 1", Authors: []string{"A"}}

	written, err := executor.WriteSidecar("Series/Series - Vol 01.epub", meta, false)
	if err != nil || written != "Series/metadata.opf" {
		t.Fatalf("%q %v", written, err)
	}
	if _, err := executor.WriteSidecar("Series/Series - Vol 01.epub", opf.Metadata{Title: "Other"}, false); !errors.Is(err, opf.ErrExists) {
		t.Fatalf("an existing sidecar was not protected: %v", err)
	}
	if content, _ := os.ReadFile(filepath.Join(managed, "Series", "metadata.opf")); !strings.Contains(string(content), "Vol 1") {
		t.Fatal("the existing sidecar changed")
	}
	if got, _ := os.ReadFile(book); string(got) != "book" {
		t.Fatal("the book itself changed")
	}

	for name, path := range map[string]string{
		"parent escape": "../outside.epub",
		"absolute":      "/etc/passwd.epub",
		"journal area":  ".ingest/x/y.epub",
		"not a book":    "Series/notes.txt",
		"missing":       "Series/Nope.epub",
		"windows style": `Series\..\x.epub`,
	} {
		if _, err := executor.WriteSidecar(path, meta, false); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestASymbolicLinkCannotCarryASidecarOutOfTheManagedLibrary(t *testing.T) {
	executor, _, managed, _ := newTestExecutor(t, 1<<20)
	outside := t.TempDir()
	os.WriteFile(filepath.Join(outside, "elsewhere.epub"), []byte("x"), 0o644)
	if err := os.Symlink(outside, filepath.Join(managed, "link")); err != nil {
		t.Skip("symbolic links are not available here")
	}
	if _, err := executor.WriteSidecar("link/elsewhere.epub", opf.Metadata{Title: "T"}, false); err == nil {
		t.Fatal("a sidecar was written through a link")
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 1 {
		t.Fatalf("something was written outside the library: %v", entries)
	}
}
