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

func TestTransferMovesPublicationsAndUndoTakesThemBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "annotations.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Apply("reader", "books", "one", []Change{{Annotation: Annotation{ID: "h1", Kind: "highlight", Quote: "First"}}}); err != nil {
		t.Fatal(err)
	}
	undo, err := store.Transfer("books", "shelf", map[string]string{"one": "one-new"})
	if err != nil {
		t.Fatal(err)
	}
	if moved := store.List("reader", "shelf", "one-new", 0); len(moved.Applied) != 1 || moved.Applied[0].Quote != "First" {
		t.Fatalf("transfer did not carry the highlight: %#v", moved.Applied)
	}
	if left := store.List("reader", "books", "one", 0); len(left.Applied) != 0 {
		t.Fatalf("transfer left the source behind: %#v", left.Applied)
	}
	// Written after the move, so the undo has to carry it back rather than
	// restore an older snapshot.
	if _, err := store.Apply("reader", "shelf", "one-new", []Change{{Annotation: Annotation{ID: "h2", Kind: "note", Note: "Later"}}}); err != nil {
		t.Fatal(err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	back := store.List("reader", "books", "one", 0)
	if len(back.Applied) != 2 {
		t.Fatalf("undo should bring both highlights back: %#v", back.Applied)
	}
	if after := store.List("reader", "shelf", "one-new", 0); len(after.Applied) != 0 {
		t.Fatalf("undo should leave nothing at the target: %#v", after.Applied)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if persisted := reopened.List("reader", "books", "one", 0); len(persisted.Applied) != 2 {
		t.Fatalf("the undo was not saved: %#v", persisted.Applied)
	}
}

// A book moved onto an id that already had notes in the target library must not
// destroy them: the undo puts the earlier target work back where it was.
func TestTransferKeepsWorkAlreadyStoredAtTheTargetAndItsUndoRestoresIt(t *testing.T) {
	store, _ := Open("")
	if _, err := store.Apply("reader", "books", "one", []Change{{Annotation: Annotation{ID: "moved", Kind: "note", Note: "moved"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Apply("reader", "shelf", "one", []Change{{Annotation: Annotation{ID: "resident", Kind: "note", Note: "resident"}}}); err != nil {
		t.Fatal(err)
	}
	undo, err := store.Transfer("books", "shelf", map[string]string{"one": "one"})
	if err != nil {
		t.Fatal(err)
	}
	if atTarget := store.List("reader", "shelf", "one", 0); len(atTarget.Applied) != 1 || atTarget.Applied[0].ID != "moved" {
		t.Fatalf("the moved publication should replace the target key: %#v", atTarget.Applied)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	resident := store.List("reader", "shelf", "one", 0)
	if len(resident.Applied) != 1 || resident.Applied[0].ID != "resident" {
		t.Fatalf("the undo should put the target's own note back: %#v", resident.Applied)
	}
	if source := store.List("reader", "books", "one", 0); len(source.Applied) != 1 || source.Applied[0].ID != "moved" {
		t.Fatalf("the undo should return the moved note: %#v", source.Applied)
	}
}

func TestTransferRefusesOneLibraryAndDifferentIdsOfTheSameName(t *testing.T) {
	store, _ := Open("")
	if _, err := store.Transfer("books", "books", map[string]string{"one": "two"}); err == nil {
		t.Fatal("a transfer between one library and itself must be refused")
	}
	if _, err := store.Transfer("", "shelf", map[string]string{"one": "two"}); err == nil {
		t.Fatal("a transfer needs a source library")
	}
}
