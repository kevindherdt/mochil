package vaultwriter

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestWriterCreatesObsidianCompatibleFoldersAndMarkdown(t *testing.T) {
	root := t.TempDir()
	writer, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = writer.CreateFolder("Projects"); err != nil {
		t.Fatal(err)
	}
	content := []byte("---\ntags: [work]\n---\n\n# Plan\n\n[[Home]]\n")
	revision, err := writer.CreatePage("Projects/Plan.md", content)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(root, "Projects", "Plan.md"))
	if err != nil || string(stored) != string(content) || revision != Revision(content) {
		t.Fatalf("stored=%q revision=%q err=%v", stored, revision, err)
	}
	if _, err = writer.CreatePage("Projects/Plan.md", nil); !errors.Is(err, ErrExists) {
		t.Fatalf("duplicate create error = %v", err)
	}
}

func TestWriterRejectsTraversalHiddenConfigurationAndNonMarkdown(t *testing.T) {
	root := t.TempDir()
	writer, _ := New(root)
	for _, path := range []string{"../outside.md", ".obsidian/plugins.md", "Notes.txt", "/absolute.md"} {
		if _, err := writer.CreatePage(path, nil); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("%q error = %v", path, err)
		}
	}
}

func TestWriterUsesRevisionConflictsAndAtomicReplacement(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Note.md")
	if err := os.WriteFile(path, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	writer, _ := New(root)
	if _, err := writer.UpdatePage("Note.md", Revision([]byte("stale")), []byte("wrong")); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update error = %v", err)
	}
	revision, err := writer.UpdatePage("Note.md", Revision([]byte("first")), []byte("second"))
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := os.ReadFile(path)
	if string(stored) != "second" || revision != Revision(stored) {
		t.Fatalf("stored=%q revision=%q", stored, revision)
	}
}

func TestWriterMovesPageWithoutClobberingOrAcceptingStaleContent(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Archive"), 0o750); err != nil {
		t.Fatal(err)
	}
	content := []byte("# Plan")
	if err := os.WriteFile(filepath.Join(root, "Plan.md"), content, 0o640); err != nil {
		t.Fatal(err)
	}
	writer, _ := New(root)
	if err := writer.MovePage("Plan.md", "Archive/Plan.md", Revision([]byte("stale"))); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale move error = %v", err)
	}
	if err := writer.MovePage("Plan.md", "Archive/Plan.md", Revision(content)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "Plan.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("source still exists: %v", err)
	}
	if stored, err := os.ReadFile(filepath.Join(root, "Archive", "Plan.md")); err != nil || string(stored) != string(content) {
		t.Fatalf("moved content = %q, %v", stored, err)
	}
	if err := os.WriteFile(filepath.Join(root, "Other.md"), []byte("other"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := writer.MovePage("Archive/Plan.md", "Other.md", Revision(content)); !errors.Is(err, ErrExists) {
		t.Fatalf("clobber move error = %v", err)
	}
}

func TestWriterMovesDeletedPagesToRecoverableVaultTrash(t *testing.T) {
	root := t.TempDir()
	content := []byte("keep me")
	if err := os.WriteFile(filepath.Join(root, "Note.md"), content, 0o640); err != nil {
		t.Fatal(err)
	}
	writer, _ := New(root)
	if err := writer.TrashPage("Note.md", Revision([]byte("stale"))); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale trash error = %v", err)
	}
	if err := writer.TrashPage("Note.md", Revision(content)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "Note.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("visible page still exists: %v", err)
	}
	if stored, err := os.ReadFile(filepath.Join(root, ".trash", "Note.md")); err != nil || string(stored) != string(content) {
		t.Fatalf("trashed content = %q, %v", stored, err)
	}
}

// A note saved as .markdown is a Markdown note everywhere else in the app, so the writer must
// accept it as a page rather than refusing it as a non-Markdown file.
func TestWriterAcceptsBothMarkdownExtensionsForPages(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Notes"), 0o750); err != nil {
		t.Fatal(err)
	}
	writer, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	original := []byte("# Longform\r\n\r\nA note with the other extension.\r\n")
	if _, err = writer.CreatePage("Notes/Longform.markdown", original); err != nil {
		t.Fatalf("create .markdown page = %v", err)
	}
	path := filepath.Join(root, "Notes", "Longform.markdown")
	if stored, err := os.ReadFile(path); err != nil || string(stored) != string(original) {
		t.Fatalf("stored = %q, %v", stored, err)
	}
	updated := []byte("# Longform\r\n\r\nEdited.\r\n")
	revision, err := writer.UpdatePage("Notes/Longform.markdown", Revision(original), updated)
	if err != nil || revision != Revision(updated) {
		t.Fatalf("update .markdown page = %v, revision %q", err, revision)
	}
	if err = writer.MovePage("Notes/Longform.markdown", "Notes/Renamed.markdown", revision); err != nil {
		t.Fatalf("move .markdown page = %v", err)
	}
	if err = writer.TrashPage("Notes/Renamed.markdown", revision); err != nil {
		t.Fatalf("trash .markdown page = %v", err)
	}
}
