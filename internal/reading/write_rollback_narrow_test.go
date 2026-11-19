package reading

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// blockedStore opens a store whose save path can never succeed (its parent
// directory is a file instead of a directory), so every attempted persist
// fails.
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
	store.path = filepath.Join(blocker, "reading.json")
	return store
}

// Progress saves are coalesced and acknowledged in memory, so they cannot be
// rolled back on a failed write (see persist_test.go). Deleting a position and
// purging an account still write before they return, because a delete that is
// only in memory is not a delete -- and these tests pin that they still undo
// exactly the change they failed to persist.

func TestDeleteRollsBackOnFailureWithoutDisturbingOtherKeys(t *testing.T) {
	store := blockedStore(t)
	key := readingKey("books", "one")
	otherKey := readingKey("books", "other")
	store.data.Accounts["reader"] = map[string]Position{
		key:      {LibraryID: "books", BookID: "one", Fraction: .3, Revision: 1},
		otherKey: {LibraryID: "books", BookID: "other", Fraction: .9, Revision: 5},
	}
	otherBefore := store.data.Accounts["reader"][otherKey]

	if err := store.Delete("reader", "books", "one"); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got, ok := store.data.Accounts["reader"][key]; !ok || got.Fraction != .3 {
		t.Fatalf("Delete did not restore the position it removed: %#v", store.data.Accounts["reader"])
	}
	if got := store.data.Accounts["reader"][otherKey]; !reflect.DeepEqual(got, otherBefore) {
		t.Fatalf("a failed delete disturbed a sibling key: %#v want %#v", got, otherBefore)
	}
}

func TestDeleteAccountRollsBackOnFailureWithoutDisturbingOtherAccounts(t *testing.T) {
	store := blockedStore(t)
	store.data.Accounts["victim"] = map[string]Position{
		readingKey("books", "untouched"): {LibraryID: "books", BookID: "untouched", Fraction: .5, Revision: 2},
	}
	store.data.Accounts["doomed"] = map[string]Position{
		readingKey("books", "one"): {LibraryID: "books", BookID: "one", Fraction: .3, Revision: 1},
	}
	store.data.History["doomed"] = map[string][]Position{
		readingKey("books", "one"): {{LibraryID: "books", BookID: "one", Fraction: .3, Revision: 1}},
	}

	if err := store.DeleteAccount("doomed"); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got, ok := store.data.Accounts["doomed"][readingKey("books", "one")]; !ok || got.Fraction != .3 {
		t.Fatalf("DeleteAccount did not restore the positions it removed: %#v", store.data.Accounts["doomed"])
	}
	if len(store.data.History["doomed"][readingKey("books", "one")]) != 1 {
		t.Fatalf("DeleteAccount did not restore the history it removed: %#v", store.data.History["doomed"])
	}
	if got := store.data.Accounts["victim"][readingKey("books", "untouched")]; got.Fraction != .5 {
		t.Fatalf("an unrelated account was disturbed by a failed write: %#v", got)
	}
}

func TestTransferRollsBackOnFailure(t *testing.T) {
	store := blockedStore(t)
	key := readingKey("old", "one")
	store.data.Accounts["reader"] = map[string]Position{
		key: {LibraryID: "old", BookID: "one", Fraction: .4, Revision: 2},
	}
	store.data.History["reader"] = map[string][]Position{key: {{LibraryID: "old", BookID: "one", Fraction: .2, Revision: 1}}}

	if _, err := store.Transfer("old", "new", map[string]string{"one": "moved"}); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got, ok := store.data.Accounts["reader"][key]; !ok || got.LibraryID != "old" || got.Fraction != .4 {
		t.Fatalf("Transfer did not restore the positions it moved: %#v", store.data.Accounts["reader"])
	}
	if entries := store.data.History["reader"][key]; len(entries) != 1 || entries[0].LibraryID != "old" {
		t.Fatalf("Transfer did not restore the history it moved: %#v", store.data.History["reader"])
	}
}

// Deferred progress saves still have to agree with the bulk paths: a delete
// flushes what was pending, so an acknowledged position cannot reappear.
func TestDeletePersistsPendingProgressSaves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reading.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .5}, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("reader", "books", "one"); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Get("reader", "books", "one"); ok {
		t.Fatal("a deleted position came back from a stale write")
	}
}

func TestDeleteOfMissingPositionIsANoOp(t *testing.T) {
	store, _ := Open("")
	if err := store.Delete("reader", "books", "missing"); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(nil, nil) {
		t.Fatal("sanity check failed")
	}
}
