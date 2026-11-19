package catalog

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// blockedOverrideStore opens a store whose save path can never succeed (its
// parent directory is a file instead of a directory), so every attempted
// persist fails and triggers rollback.
func blockedOverrideStore(t *testing.T) *OverrideStore {
	t.Helper()
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenOverrideStore("")
	if err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "overrides.json")
	return store
}

func seedOverrideVictim(store *OverrideStore) {
	store.data.Libraries["victim-library"] = map[string]MetadataOverride{
		"untouched-book": {Title: "Untouched Title", Authors: []string{"Original Author"}},
	}
}

// These tests prove the narrow, per-entry rollback introduced to remove the
// full-store clone is still correct: a failed write undoes exactly the
// attempted change and leaves every other library/book untouched, whether
// or not the library/book already existed.

func TestSetArtworkRollsBackNewlyCreatedLibraryOnFailure(t *testing.T) {
	store := blockedOverrideStore(t)
	seedOverrideVictim(store)

	if err := store.SetArtwork("brand-new-library", "book", "https://example.com/cover.jpg", "provider", "id"); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if _, exists := store.data.Libraries["brand-new-library"]; exists {
		t.Fatal("a failed write left behind a freshly created, empty library")
	}
	if got := store.data.Libraries["victim-library"]["untouched-book"]; got.Title != "Untouched Title" {
		t.Fatalf("an unrelated library was disturbed by a failed write: %#v", got)
	}
}

func TestSetArtworkRollsBackExistingBookWithoutDisturbingSiblings(t *testing.T) {
	store := blockedOverrideStore(t)
	seedOverrideVictim(store)
	before := MetadataOverride{Title: "Keep Title", ArtworkURL: "https://old.example.com/cover.jpg", Provider: "old", ProviderID: "old-id"}
	sibling := MetadataOverride{Title: "Sibling Title"}
	store.data.Libraries["library"] = map[string]MetadataOverride{"book-one": before, "book-two": sibling}

	if err := store.SetArtwork("library", "book-one", "https://new.example.com/cover.jpg", "new", "new-id"); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got := store.data.Libraries["library"]["book-one"]; !reflect.DeepEqual(got, before) {
		t.Fatalf("SetArtwork left a partial write behind: %#v want %#v", got, before)
	}
	if got := store.data.Libraries["library"]["book-two"]; !reflect.DeepEqual(got, sibling) {
		t.Fatalf("a failed write to one book disturbed a sibling book: %#v want %#v", got, sibling)
	}
}

func TestDeleteArtworkRollsBackOnFailure(t *testing.T) {
	store := blockedOverrideStore(t)
	seedOverrideVictim(store)
	before := MetadataOverride{Title: "Keep Title", ArtworkURL: "https://old.example.com/cover.jpg", Provider: "old", ProviderID: "old-id"}
	store.data.Libraries["library"] = map[string]MetadataOverride{"book": before}

	if err := store.DeleteArtwork("library", "book"); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got := store.data.Libraries["library"]["book"]; !reflect.DeepEqual(got, before) {
		t.Fatalf("DeleteArtwork left a partial write behind: %#v want %#v", got, before)
	}
}

func TestApplyWithUndoRollsBackNewlyCreatedLibraryOnFailure(t *testing.T) {
	store := blockedOverrideStore(t)
	seedOverrideVictim(store)

	if _, err := store.ApplyWithUndo("brand-new-library", map[string]map[string]string{"book": {"title": "New Title"}}); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if _, exists := store.data.Libraries["brand-new-library"]; exists {
		t.Fatal("a failed write left behind a freshly created, empty library")
	}
	if got := store.data.Libraries["victim-library"]["untouched-book"]; got.Title != "Untouched Title" {
		t.Fatalf("an unrelated library was disturbed by a failed write: %#v", got)
	}
}

func TestApplyWithUndoRollsBackPartialBatchOnValidationFailureWithoutDisturbingOtherBooks(t *testing.T) {
	store, _ := OpenOverrideStore("")
	before := MetadataOverride{Title: "Original"}
	sibling := MetadataOverride{Title: "Sibling"}
	store.data.Libraries["library"] = map[string]MetadataOverride{"book-one": before, "book-two": sibling}

	// book-one would be changed first (map order is nondeterministic, so try
	// the invalid field on the book that must remain untouched) -- instead
	// assert directly that an invalid field anywhere rolls back everything.
	_, err := store.ApplyWithUndo("library", map[string]map[string]string{
		"book-one": {"title": "Changed"},
		"book-two": {"rating": "not-a-number"},
	})
	if err == nil {
		t.Fatal("expected an invalid rating to be rejected")
	}
	if got := store.data.Libraries["library"]["book-one"]; !reflect.DeepEqual(got, before) {
		t.Fatalf("ApplyWithUndo left a partial write behind after a mid-batch validation failure: %#v want %#v", got, before)
	}
	if got := store.data.Libraries["library"]["book-two"]; !reflect.DeepEqual(got, sibling) {
		t.Fatalf("ApplyWithUndo disturbed a sibling book after a mid-batch validation failure: %#v want %#v", got, sibling)
	}
}

func TestApplyWithUndoRollsBackOnFailureWithoutDisturbingOtherBooks(t *testing.T) {
	store := blockedOverrideStore(t)
	seedOverrideVictim(store)
	before := MetadataOverride{Title: "Original"}
	sibling := MetadataOverride{Title: "Sibling"}
	store.data.Libraries["library"] = map[string]MetadataOverride{"book-one": before, "book-two": sibling}

	if _, err := store.ApplyWithUndo("library", map[string]map[string]string{"book-one": {"title": "Changed"}}); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got := store.data.Libraries["library"]["book-one"]; !reflect.DeepEqual(got, before) {
		t.Fatalf("ApplyWithUndo left a partial write behind: %#v want %#v", got, before)
	}
	if got := store.data.Libraries["library"]["book-two"]; !reflect.DeepEqual(got, sibling) {
		t.Fatalf("a failed write to one book disturbed a sibling book: %#v want %#v", got, sibling)
	}
}

func TestApplyWithUndoFunctionRestoresPriorStateAndTolerateSubsequentFailure(t *testing.T) {
	store, err := OpenOverrideStore("")
	if err != nil {
		t.Fatal(err)
	}
	before := MetadataOverride{Title: "Original"}
	store.data.Libraries["library"] = map[string]MetadataOverride{"book": before}

	undo, err := store.ApplyWithUndo("library", map[string]map[string]string{"book": {"title": "Changed"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := store.data.Libraries["library"]["book"].Title; got != "Changed" {
		t.Fatalf("ApplyWithUndo did not apply the change: %q", got)
	}

	// Now block persistence and verify the undo function itself rolls back
	// on a failed save, leaving the applied (not pre-apply) state in place.
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "overrides.json")
	if err := undo(); err == nil {
		t.Fatal("expected the blocked undo save to fail")
	}
	if got := store.data.Libraries["library"]["book"].Title; got != "Changed" {
		t.Fatalf("a failed undo left a partial write behind: %q want %q", got, "Changed")
	}
}

func TestResetRollsBackOnFailure(t *testing.T) {
	store := blockedOverrideStore(t)
	seedOverrideVictim(store)
	before := MetadataOverride{Title: "Keep Title", Authors: []string{"Keep Author"}}
	store.data.Libraries["library"] = map[string]MetadataOverride{"book": before}

	if _, err := store.Reset("library", "book", "title"); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got := store.data.Libraries["library"]["book"]; !reflect.DeepEqual(got, before) {
		t.Fatalf("Reset left a partial write behind: %#v want %#v", got, before)
	}
}

func TestResetRemovesEntryWhenEverythingIsClearedThenRestoresOnFailure(t *testing.T) {
	store := blockedOverrideStore(t)
	before := MetadataOverride{Title: "Only Field Set"}
	store.data.Libraries["library"] = map[string]MetadataOverride{"book": before}

	if _, err := store.Reset("library", "book"); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got, ok := store.data.Libraries["library"]["book"]; !ok || !reflect.DeepEqual(got, before) {
		t.Fatalf("a failed Reset should restore the pre-reset entry even though it would have been deleted: %#v", got)
	}
}
