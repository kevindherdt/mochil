package customcolumns

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// blockedStore opens a store whose save path can never succeed (its parent
// directory is a file instead of a directory), so every attempted persist
// fails and triggers rollback.
func blockedStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	return store // caller blocks store.path after any setup that itself needs to save (Add, initial Set)
}

// blockPath points store at a save path that can never succeed.
func blockPath(t *testing.T, store *Store) {
	t.Helper()
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "columns.json")
}

// These tests prove the narrow, per-cell rollback introduced to remove the
// full-store clone is still correct: a failed write undoes exactly the
// attempted change and leaves every other library/book/column untouched,
// cleaning up any freshly created, now-empty book/library entries.

func TestSetRollsBackNewlyCreatedLibraryAndBookOnFailure(t *testing.T) {
	store := blockedStore(t)
	status, err := store.Add("Status", Text)
	if err != nil {
		t.Fatal(err)
	}
	store.data.Values["victim-library"] = map[string]map[string]string{
		"untouched-book": {status.ID: "untouched value"},
	}
	blockPath(t, store)

	if _, err := store.Set("brand-new-library", map[string]map[string]string{"new-book": {status.ID: "new value"}}); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if _, exists := store.data.Values["brand-new-library"]; exists {
		t.Fatal("a failed write left behind a freshly created, empty library")
	}
	if got := store.data.Values["victim-library"]["untouched-book"][status.ID]; got != "untouched value" {
		t.Fatalf("an unrelated library was disturbed by a failed write: %q", got)
	}
}

func TestSetRollsBackExistingCellWithoutDisturbingSiblingCellsOrBooks(t *testing.T) {
	store := blockedStore(t)
	status, err := store.Add("Status", Text)
	if err != nil {
		t.Fatal(err)
	}
	genre, err := store.Add("Genre", Text)
	if err != nil {
		t.Fatal(err)
	}
	store.data.Values["library"] = map[string]map[string]string{
		"book-one": {status.ID: "Reading", genre.ID: "Fiction"},
		"book-two": {status.ID: "Finished"},
	}
	blockPath(t, store)

	if _, err := store.Set("library", map[string]map[string]string{"book-one": {status.ID: "Changed"}}); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got := store.data.Values["library"]["book-one"][status.ID]; got != "Reading" {
		t.Fatalf("Set left a partial write behind: %q want %q", got, "Reading")
	}
	if got := store.data.Values["library"]["book-one"][genre.ID]; got != "Fiction" {
		t.Fatalf("a failed write to one column disturbed a sibling column: %q want %q", got, "Fiction")
	}
	if got := store.data.Values["library"]["book-two"][status.ID]; got != "Finished" {
		t.Fatalf("a failed write to one book disturbed a sibling book: %q want %q", got, "Finished")
	}
}

func TestSetRollsBackFreshlyCreatedBookWithoutDisturbingSiblingBook(t *testing.T) {
	store := blockedStore(t)
	status, err := store.Add("Status", Text)
	if err != nil {
		t.Fatal(err)
	}
	store.data.Values["library"] = map[string]map[string]string{
		"existing-book": {status.ID: "Reading"},
	}
	blockPath(t, store)

	if _, err := store.Set("library", map[string]map[string]string{"brand-new-book": {status.ID: "New"}}); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if _, exists := store.data.Values["library"]["brand-new-book"]; exists {
		t.Fatal("a failed write left behind a freshly created, empty book")
	}
	if got := store.data.Values["library"]["existing-book"][status.ID]; got != "Reading" {
		t.Fatalf("a failed write to a new book disturbed a sibling book: %q want %q", got, "Reading")
	}
}

func TestSetUndoRestoresExactPriorStateIncludingClearedCells(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	status, err := store.Add("Status", Text)
	if err != nil {
		t.Fatal(err)
	}
	genre, err := store.Add("Genre", Text)
	if err != nil {
		t.Fatal(err)
	}
	// book-one has no genre set initially; book-two is untouched throughout.
	if _, err := store.Set("library", map[string]map[string]string{"book-one": {status.ID: "Reading"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Set("library", map[string]map[string]string{"book-two": {status.ID: "On Hold"}}); err != nil {
		t.Fatal(err)
	}

	undo, err := store.Set("library", map[string]map[string]string{"book-one": {status.ID: "Finished", genre.ID: "Sci-Fi"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if got := store.data.Values["library"]["book-one"][status.ID]; got != "Reading" {
		t.Fatalf("undo did not restore the previous status: %q want %q", got, "Reading")
	}
	if _, exists := store.data.Values["library"]["book-one"][genre.ID]; exists {
		t.Fatal("undo left behind a genre value that did not exist before Set")
	}
	if got := store.data.Values["library"]["book-two"][status.ID]; got != "On Hold" {
		t.Fatalf("undo disturbed an unrelated book: %q want %q", got, "On Hold")
	}
}

func TestSetUndoRollsBackOnFailureLeavingTheAppliedStateInPlace(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	status, err := store.Add("Status", Text)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Set("library", map[string]map[string]string{"book": {status.ID: "Original"}}); err != nil {
		t.Fatal(err)
	}
	undo, err := store.Set("library", map[string]map[string]string{"book": {status.ID: "Changed"}})
	if err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "columns.json")

	if err := undo(); err == nil {
		t.Fatal("expected the blocked undo save to fail")
	}
	if got := store.data.Values["library"]["book"][status.ID]; got != "Changed" {
		t.Fatalf("a failed undo left a partial write behind: %q want %q", got, "Changed")
	}
}

func TestSetUndoSkipsARemovedColumnWithoutDisturbingOthers(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	status, err := store.Add("Status", Text)
	if err != nil {
		t.Fatal(err)
	}
	genre, err := store.Add("Genre", Text)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Set("library", map[string]map[string]string{"book": {status.ID: "Reading", genre.ID: "Fiction"}}); err != nil {
		t.Fatal(err)
	}
	undo, err := store.Set("library", map[string]map[string]string{"book": {status.ID: "Finished", genre.ID: "Non-Fiction"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Remove(genre.ID); err != nil {
		t.Fatal(err)
	}

	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if got := store.data.Values["library"]["book"][status.ID]; got != "Reading" {
		t.Fatalf("undo did not restore the surviving column: %q want %q", got, "Reading")
	}
	if got, exists := store.data.Values["library"]["book"][genre.ID]; exists {
		t.Fatalf("undo resurrected a removed column's value: %q", got)
	}
}

func TestSetRollsBackSameFieldsSemanticsAsBeforeRefactor(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	status, err := store.Add("Status", Text)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Set("library", map[string]map[string]string{"book": {status.ID: "Reading"}}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.data.Values["library"]["book"], map[string]string{status.ID: "Reading"}) {
		t.Fatalf("Set did not persist the expected cell: %#v", store.data.Values["library"]["book"])
	}
}
