package annotations

import (
	"fmt"
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
	store.path = filepath.Join(blocker, "annotations.json")
	return store
}

// These tests prove the narrow, per-publication rollback introduced to
// remove the full-store clone is still correct: a failed write undoes
// exactly the attempted change and leaves every other account/book
// untouched, whether or not the account/publication already existed.
func TestApplyRollsBackNewlyCreatedAccountOnFailure(t *testing.T) {
	store := blockedStore(t)
	store.data.Accounts["victim"] = map[string]publication{
		publicationKey("books", "untouched"): {Revision: 2, Items: map[string]Annotation{"x": {ID: "x", Kind: "bookmark", Revision: 2}}},
	}

	if _, err := store.Apply("brand-new-reader", "books", "one", []Change{{Annotation: Annotation{ID: "first", Kind: "bookmark"}}}); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if _, exists := store.data.Accounts["brand-new-reader"]; exists {
		t.Fatal("a failed write left behind a freshly created, empty account")
	}
	if got := store.data.Accounts["victim"][publicationKey("books", "untouched")]; len(got.Items) != 1 || got.Revision != 2 {
		t.Fatalf("an unrelated account was disturbed by a failed write: %#v", got)
	}
}

func TestApplyRollsBackExistingPublicationWithoutDisturbingOtherBooks(t *testing.T) {
	store := blockedStore(t)
	key := publicationKey("books", "one")
	otherKey := publicationKey("books", "other")
	original := publication{Revision: 1, Items: map[string]Annotation{"first": {ID: "first", Kind: "bookmark", Revision: 1}}}
	otherOriginal := publication{Revision: 3, Items: map[string]Annotation{"other-item": {ID: "other-item", Kind: "note", Note: "keep me", Revision: 3}}}
	store.data.Accounts["reader"] = map[string]publication{
		key:      {Revision: original.Revision, Items: map[string]Annotation{"first": original.Items["first"]}},
		otherKey: {Revision: otherOriginal.Revision, Items: map[string]Annotation{"other-item": otherOriginal.Items["other-item"]}},
	}

	if _, err := store.Apply("reader", "books", "one", []Change{{BaseRevision: 1, Annotation: Annotation{ID: "first", Kind: "bookmark", Note: "changed"}}}); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got := store.data.Accounts["reader"][key]; !reflect.DeepEqual(got, original) {
		t.Fatalf("Apply left a partial write behind: %#v want %#v", got, original)
	}
	if got := store.data.Accounts["reader"][otherKey]; !reflect.DeepEqual(got, otherOriginal) {
		t.Fatalf("a failed write to one publication disturbed a sibling publication: %#v want %#v", got, otherOriginal)
	}
}

func TestApplyRollsBackPartialBatchWhenPerBookLimitExceeded(t *testing.T) {
	store, _ := Open("")
	items := map[string]Annotation{}
	for i := 0; i < MaximumItemsPerBook; i++ {
		id := fmt.Sprintf("item-%d", i)
		items[id] = Annotation{ID: id, Kind: "bookmark", Revision: 1}
	}
	key := publicationKey("books", "full")
	store.data.Accounts["reader"] = map[string]publication{key: {Revision: int64(len(items)), Items: items}}
	before := store.data.Accounts["reader"][key]
	beforeCount := len(before.Items)

	_, err := store.Apply("reader", "books", "full", []Change{{Annotation: Annotation{ID: "brand-new", Kind: "bookmark"}}})
	if err != ErrInvalid {
		t.Fatalf("expected ErrInvalid when the per-book item limit is exceeded, got %v", err)
	}
	got := store.data.Accounts["reader"][key]
	if len(got.Items) != beforeCount || got.Revision != before.Revision {
		t.Fatalf("Apply left a partial write behind after a mid-batch abort: %d items, revision %d", len(got.Items), got.Revision)
	}
	if _, exists := got.Items["brand-new"]; exists {
		t.Fatal("Apply left the rejected new item behind after a mid-batch abort")
	}
}

func TestDeleteAccountRollsBackOnFailureWithoutDisturbingOtherAccounts(t *testing.T) {
	store := blockedStore(t)
	store.data.Accounts["victim"] = map[string]publication{
		publicationKey("books", "untouched"): {Revision: 2, Items: map[string]Annotation{"x": {ID: "x", Kind: "bookmark", Revision: 2}}},
	}
	doomed := map[string]publication{
		publicationKey("books", "one"): {Revision: 1, Items: map[string]Annotation{"first": {ID: "first", Kind: "bookmark", Revision: 1}}},
	}
	store.data.Accounts["doomed"] = doomed

	if _, err := store.DeleteAccount("doomed"); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got, ok := store.data.Accounts["doomed"][publicationKey("books", "one")]; !ok || len(got.Items) != 1 {
		t.Fatalf("DeleteAccount did not restore the account it removed: %#v", store.data.Accounts["doomed"])
	}
	if got := store.data.Accounts["victim"][publicationKey("books", "untouched")]; len(got.Items) != 1 {
		t.Fatalf("an unrelated account was disturbed by a failed write: %#v", got)
	}
}
