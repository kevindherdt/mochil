package annotations

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

func TestStoreMergesIndependentItemsAndPersistsTombstones(t *testing.T) {
	path := filepath.Join(t.TempDir(), "annotations.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Apply("reader", "books", "one", []Change{
		{Annotation: Annotation{ID: "first", Kind: "highlight", Quote: "First quote"}},
		{Annotation: Annotation{ID: "second", Kind: "note", Note: "Second note"}},
	})
	if err != nil || result.Revision != 2 || len(result.Applied) != 2 {
		t.Fatalf("initial apply: %#v %v", result, err)
	}
	first := result.Applied[0]
	deleted, err := store.Apply("reader", "books", "one", []Change{{BaseRevision: first.Revision, Annotation: Annotation{ID: first.ID, Kind: first.Kind, Deleted: true, CreatedAt: first.CreatedAt}}})
	if err != nil || !deleted.Applied[0].Deleted {
		t.Fatalf("delete: %#v %v", deleted, err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	changes := reopened.List("reader", "books", "one", 2)
	if len(changes.Applied) != 1 || !changes.Applied[0].Deleted || changes.Revision != 3 {
		t.Fatalf("persisted changes: %#v", changes)
	}
}

func TestStoreRejectsSameItemConflictWithoutBlockingOtherChanges(t *testing.T) {
	store, _ := Open("")
	created, _ := store.Apply("reader", "books", "one", []Change{{Annotation: Annotation{ID: "shared", Kind: "note", Note: "original"}}})
	base := created.Applied[0]
	first, _ := store.Apply("reader", "books", "one", []Change{{BaseRevision: base.Revision, Annotation: Annotation{ID: base.ID, Kind: base.Kind, Note: "device one"}}})
	result, err := store.Apply("reader", "books", "one", []Change{
		{BaseRevision: base.Revision, Annotation: Annotation{ID: base.ID, Kind: base.Kind, Note: "device two"}},
		{Annotation: Annotation{ID: "independent", Kind: "bookmark"}},
	})
	if err != nil || len(result.Conflicts) != 1 || len(result.Applied) != 1 || result.Conflicts[0].Current.Note != "device one" || result.Revision != first.Revision+1 {
		t.Fatalf("merge result: %#v %v", result, err)
	}
}

func TestStoreRejectsInvalidUpdates(t *testing.T) {
	store, _ := Open("")
	_, err := store.Apply("reader", "books", "one", []Change{{Annotation: Annotation{ID: "bad", Kind: "unknown", Quote: "x"}}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid annotation accepted: %v", err)
	}
}

func TestStoreExportsOnlyRequestedAccountDeterministically(t *testing.T) {
	store, _ := Open("")
	_, _ = store.Apply("reader", "z-library", "second", []Change{{Annotation: Annotation{ID: "z", Kind: "bookmark"}}})
	_, _ = store.Apply("reader", "a-library", "first", []Change{{Annotation: Annotation{ID: "a", Kind: "highlight", Quote: "Keep me"}}})
	_, _ = store.Apply("other", "private", "book", []Change{{Annotation: Annotation{ID: "secret", Kind: "note", Note: "Do not export"}}})

	payload, err := store.ExportAccount("reader")
	if err != nil {
		t.Fatal(err)
	}
	var exported Export
	if err := json.Unmarshal(payload, &exported); err != nil {
		t.Fatal(err)
	}
	if exported.Version != 1 || len(exported.Publications) != 2 || exported.Publications[0].LibraryID != "a-library" || exported.Publications[0].Items[0].ID != "a" {
		t.Fatalf("unexpected export: %#v", exported)
	}
	if string(payload) == "" || json.Valid(payload) == false {
		t.Fatalf("invalid export: %q", payload)
	}
}

func TestStoreDeleteAccountIsIsolatedAndPersistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "annotations.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.Apply("reader", "books", "one", []Change{{Annotation: Annotation{ID: "first", Kind: "bookmark"}}})
	_, _ = store.Apply("other", "books", "one", []Change{{Annotation: Annotation{ID: "second", Kind: "bookmark"}}})
	deleted, err := store.DeleteAccount("reader")
	if err != nil || deleted != 1 {
		t.Fatalf("delete = %d, %v", deleted, err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.List("reader", "books", "one", 0); got.Revision != 0 || len(got.Applied) != 0 {
		t.Fatalf("deleted account remains: %#v", got)
	}
	if got := reopened.List("other", "books", "one", 0); len(got.Applied) != 1 || got.Applied[0].ID != "second" {
		t.Fatalf("other account was changed: %#v", got)
	}
}
