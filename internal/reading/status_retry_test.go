package reading

import (
	"errors"
	"reflect"
	"testing"
)

func TestDismissalRetryPreservesActiveReadingRevision(t *testing.T) {
	store, _ := Open("")
	first, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .2,
		Locators: map[string]string{"readium": "location"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	dismissed, err := store.SetDismissed("reader", "books", "one", true)
	if err != nil || dismissed.Revision != first.Revision+1 {
		t.Fatalf("dismiss: %+v %v", dismissed, err)
	}
	history := store.History("reader", "books", "one")
	store.path = t.TempDir()
	retry, err := store.SetDismissed("reader", "books", "one", true)
	if err != nil || !reflect.DeepEqual(retry, dismissed) {
		t.Fatalf("retry: %+v %v", retry, err)
	}
	retry.Locators["readium"] = "mutated response"
	stored, _ := store.Get("reader", "books", "one")
	if !reflect.DeepEqual(stored, dismissed) || !reflect.DeepEqual(history, store.History("reader", "books", "one")) {
		t.Fatal("retry changed stored location or history")
	}
	store.path = ""
	dismissed.Fraction = .4
	if _, err := store.Put("reader", dismissed, dismissed.Revision); err != nil {
		t.Fatalf("retry invalidated reader: %v", err)
	}
}

func TestRejectedInitialPositionDoesNotCreateAccount(t *testing.T) {
	store, _ := Open("")
	_, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .2}, 42)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("error: %v", err)
	}
	if len(store.data.Accounts) != 0 || len(store.data.History) != 0 {
		t.Fatal("rejected write created state")
	}
}
