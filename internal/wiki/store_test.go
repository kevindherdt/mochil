package wiki

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenRejectsMalformedState(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "wiki.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"accounts":{}} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("wiki state with trailing JSON was accepted")
	}
	invalid := filepath.Join(root, "invalid.json")
	if err := os.WriteFile(invalid, []byte(`{"version":1,"accounts":{"alice":null}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(invalid); err == nil {
		t.Fatal("wiki state with a null account was accepted")
	}
}

func TestVersionedLifecycleAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wiki.json")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	p, e := s.Create("alice", "Start", "one")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := s.Create("alice", "Linking", "See [[Start]]"); e != nil {
		t.Fatal(e)
	}
	if links := s.Backlinks("alice", "Start"); len(links) != 1 || links[0].Title != "Linking" {
		t.Fatalf("backlinks: %#v", links)
	}
	if _, e = s.Update("alice", p.ID, "Changed", "two", 99); !errors.Is(e, ErrConflict) {
		t.Fatalf("wanted conflict, got %v", e)
	}
	p, e = s.Update("alice", p.ID, "Changed", "two", 1)
	if e != nil || p.Revision != 2 {
		t.Fatalf("update: %#v %v", p, e)
	}
	h, e := s.History("alice", p.ID)
	if e != nil || len(h) != 1 || h[0].Content != "one" {
		t.Fatalf("history: %#v %v", h, e)
	}
	p, e = s.Delete("alice", p.ID, 2)
	if e != nil || p.DeletedAt == nil {
		t.Fatalf("delete: %#v %v", p, e)
	}
	if len(s.List("alice", false)) != 1 || len(s.List("alice", true)) != 1 {
		t.Fatal("trash filtering failed")
	}
	p, e = s.Restore("alice", p.ID, 3)
	if e != nil || p.DeletedAt != nil {
		t.Fatalf("restore: %#v %v", p, e)
	}
	p, e = s.RestoreVersion("alice", p.ID, 1, 4)
	if e != nil || p.Content != "one" || p.Revision != 5 {
		t.Fatalf("restore version: %#v %v", p, e)
	}
	reopened, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	got, ok := reopened.Get("alice", p.ID)
	if !ok || got.Content != "one" {
		t.Fatalf("not persistent: %#v", got)
	}
	if _, ok := reopened.Get("bob", p.ID); ok {
		t.Fatal("page leaked between accounts")
	}
}

func TestPurgeRequiresTrashRetentionAndRevision(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "wiki.json"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := store.Create("alice", "Temporary", "content")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Purge("alice", page.ID, page.Revision, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("active page purge error=%v", err)
	}
	page, err = store.Delete("alice", page.ID, page.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Purge("alice", page.ID, page.Revision-1, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale purge error=%v", err)
	}
	if err = store.Purge("alice", page.ID, page.Revision, 7*24*time.Hour); !errors.Is(err, ErrRetention) {
		t.Fatalf("early purge error=%v", err)
	}
	if err = store.Purge("alice", page.ID, page.Revision, 0); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get("alice", page.ID); ok {
		t.Fatal("purged page remained")
	}
	if _, err = store.History("alice", page.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("history remained: %v", err)
	}
}

func TestNestedFolderPersistsAndIsVersioned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wiki.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	page, err := store.CreateInFolder("alice", "Chapter notes", ` Books\\Fantasy / Rebuild World `, "one")
	if err != nil || page.Folder != "Books/Fantasy/Rebuild World" {
		t.Fatalf("create: %#v %v", page, err)
	}
	page, err = store.UpdateInFolder("alice", page.ID, page.Title, "Books/Finished", "two", page.Revision)
	if err != nil {
		t.Fatal(err)
	}
	history, err := store.History("alice", page.ID)
	if err != nil || len(history) != 1 || history[0].Folder != "Books/Fantasy/Rebuild World" {
		t.Fatalf("history: %#v %v", history, err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := reopened.Get("alice", page.ID)
	if got.Folder != "Books/Finished" {
		t.Fatalf("folder=%q", got.Folder)
	}
	if _, err = store.CreateInFolder("alice", "Bad", "../Private", "x"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid folder: %v", err)
	}
}

func TestEmptyFoldersPersistAndDeduplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wiki.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	folder, err := store.CreateFolder("alice", ` Books / Fantasy `)
	if err != nil || folder != "Books/Fantasy" {
		t.Fatalf("create: %q %v", folder, err)
	}
	if _, err = store.CreateFolder("alice", "books/fantasy"); err != nil {
		t.Fatal(err)
	}
	if got := store.ListFolders("alice"); len(got) != 1 || got[0] != folder {
		t.Fatalf("folders: %#v", got)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.ListFolders("alice"); len(got) != 1 || got[0] != folder {
		t.Fatalf("reopened: %#v", got)
	}
}

func TestStoreRefusesToSavePastItsOwnReadLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wiki.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create("acct", "First", "body"); err != nil {
		t.Fatal(err)
	}
	original := maxStateBytes
	maxStateBytes = 64
	if _, err := store.Create("acct", "Second", "a body that no longer fits"); err == nil {
		t.Fatal("a write that exceeds the read limit must be refused, not saved")
	}
	if pages := store.List("acct", false); len(pages) != 1 {
		t.Fatalf("pages = %d, want 1 (a refused page must not stay in memory)", len(pages))
	}
	maxStateBytes = original
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("a stored file must always be readable again: %v", err)
	}
	if pages := reopened.List("acct", false); len(pages) != 1 {
		t.Fatalf("reopened pages = %d, want 1", len(pages))
	}
}
