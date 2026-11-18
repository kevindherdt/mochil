package externallibrary

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
	store.path = filepath.Join(blocker, "external-library.json")
	return store
}

func seedVictim(store *Store) {
	store.data.Accounts["victim"] = accountData{
		Titles:   map[string]Title{"untouched-title": {ID: "untouched-title", Name: "Untouched", ExecutionLocation: ExecutionServer, UpdatedAt: 1}},
		Progress: map[string]map[string]Progress{"untouched-title": {"chapter": {TitleID: "untouched-title", RemoteChapterID: "chapter", Fraction: .5, Revision: 1}}},
	}
}

func TestPutRollsBackNewlyCreatedAccountOnFailure(t *testing.T) {
	store := blockedStore(t)
	seedVictim(store)

	if _, err := store.Put("brand-new-reader", Title{ProviderID: "p", SourceID: "s", RemoteTitleID: "new", Name: "New Book", ExecutionLocation: ExecutionServer}); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if _, exists := store.data.Accounts["brand-new-reader"]; exists {
		t.Fatal("a failed write left behind a freshly created, empty account")
	}
	if got := store.data.Accounts["victim"].Titles["untouched-title"]; got.Name != "Untouched" {
		t.Fatalf("an unrelated account was disturbed by a failed write: %#v", got)
	}
}

func TestPutRollsBackExistingTitleWithoutDisturbingOtherTitles(t *testing.T) {
	store := blockedStore(t)
	seedVictim(store)
	before := Title{ID: "measured-title", ProviderID: "p", SourceID: "s", RemoteTitleID: "measured", Name: "Original Name", ExecutionLocation: ExecutionServer, AddedAt: 100, UpdatedAt: 100}
	other := Title{ID: "other-title", Name: "Other", ExecutionLocation: ExecutionServer, UpdatedAt: 50}
	store.data.Accounts["reader"] = accountData{
		Titles:   map[string]Title{before.ID: before, other.ID: other},
		Progress: map[string]map[string]Progress{},
	}

	if _, err := store.Put("reader", Title{ProviderID: "p", SourceID: "s", RemoteTitleID: "measured", Name: "Changed Name", ExecutionLocation: ExecutionServer}); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got := store.data.Accounts["reader"].Titles[before.ID]; !reflect.DeepEqual(got, before) {
		t.Fatalf("Put left a partial write behind: %#v want %#v", got, before)
	}
	if got := store.data.Accounts["reader"].Titles[other.ID]; !reflect.DeepEqual(got, other) {
		t.Fatalf("a failed write to one title disturbed a sibling title: %#v want %#v", got, other)
	}
}

func TestRemoveRollsBackTitleAndItsProgressOnFailure(t *testing.T) {
	store := blockedStore(t)
	seedVictim(store)
	titleID := "measured-title"
	title := Title{ID: titleID, Name: "Measured", ExecutionLocation: ExecutionServer}
	progress := map[string]Progress{"chapter-one": {TitleID: titleID, RemoteChapterID: "chapter-one", Fraction: .4, Revision: 2}}
	other := Title{ID: "other-title", Name: "Other", ExecutionLocation: ExecutionServer}
	store.data.Accounts["reader"] = accountData{
		Titles:   map[string]Title{titleID: title, other.ID: other},
		Progress: map[string]map[string]Progress{titleID: progress},
	}

	if err := store.Remove("reader", titleID); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got, ok := store.data.Accounts["reader"].Titles[titleID]; !ok || !reflect.DeepEqual(got, title) {
		t.Fatalf("Remove did not restore the title it removed: %#v", got)
	}
	if got, ok := store.data.Accounts["reader"].Progress[titleID]; !ok || len(got) != 1 {
		t.Fatalf("Remove did not restore the progress it removed: %#v", got)
	}
	if got := store.data.Accounts["reader"].Titles[other.ID]; !reflect.DeepEqual(got, other) {
		t.Fatalf("Remove disturbed a sibling title: %#v want %#v", got, other)
	}
}

func TestDeleteAccountRollsBackOnFailureWithoutDisturbingOtherAccounts(t *testing.T) {
	store := blockedStore(t)
	seedVictim(store)
	store.data.Accounts["doomed"] = accountData{
		Titles:   map[string]Title{"one": {ID: "one", Name: "Doomed Book", ExecutionLocation: ExecutionServer}},
		Progress: map[string]map[string]Progress{"one": {"c": {TitleID: "one", RemoteChapterID: "c", Fraction: .1, Revision: 1}}},
	}

	if err := store.DeleteAccount("doomed"); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got, ok := store.data.Accounts["doomed"].Titles["one"]; !ok || got.Name != "Doomed Book" {
		t.Fatalf("DeleteAccount did not restore the account it removed: %#v", store.data.Accounts["doomed"])
	}
	if got := store.data.Accounts["victim"].Titles["untouched-title"]; got.Name != "Untouched" {
		t.Fatalf("an unrelated account was disturbed by a failed write: %#v", got)
	}
}

func TestPutProgressRollsBackOnFailureWithoutDisturbingOtherChapters(t *testing.T) {
	store := blockedStore(t)
	seedVictim(store)
	titleID := "measured-title"
	before := Progress{TitleID: titleID, RemoteChapterID: "chapter-one", Fraction: .3, Revision: 1}
	other := Progress{TitleID: titleID, RemoteChapterID: "chapter-two", Fraction: .9, Revision: 5}
	store.data.Accounts["reader"] = accountData{
		Titles: map[string]Title{titleID: {ID: titleID, Name: "Measured", ExecutionLocation: ExecutionServer}},
		Progress: map[string]map[string]Progress{titleID: {
			"chapter-one": before,
			"chapter-two": other,
		}},
	}

	if _, err := store.PutProgress("reader", titleID, Progress{RemoteChapterID: "chapter-one", Fraction: .6}, 1); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if got := store.data.Accounts["reader"].Progress[titleID]["chapter-one"]; !reflect.DeepEqual(got, before) {
		t.Fatalf("PutProgress left a partial write behind: %#v want %#v", got, before)
	}
	if got := store.data.Accounts["reader"].Progress[titleID]["chapter-two"]; !reflect.DeepEqual(got, other) {
		t.Fatalf("PutProgress disturbed a sibling chapter: %#v want %#v", got, other)
	}
}

func TestPutProgressRollsBackFreshlyCreatedChapterMapOnFailure(t *testing.T) {
	store := blockedStore(t)
	seedVictim(store)
	titleID := "measured-title"
	store.data.Accounts["reader"] = accountData{
		Titles:   map[string]Title{titleID: {ID: titleID, Name: "Measured", ExecutionLocation: ExecutionServer}},
		Progress: map[string]map[string]Progress{},
	}

	if _, err := store.PutProgress("reader", titleID, Progress{RemoteChapterID: "first-chapter", Fraction: .1}, 0); err == nil {
		t.Fatal("expected the blocked save to fail")
	}
	if _, exists := store.data.Accounts["reader"].Progress[titleID]; exists {
		t.Fatal("a failed write left behind a freshly created, empty chapter map")
	}
}
