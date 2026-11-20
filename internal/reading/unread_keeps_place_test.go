package reading

import (
	"path/filepath"
	"testing"
)

func openReading(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

// "Unread" is a label about the book, not an instruction to forget where you were. It used to delete the
// position and the history with it, so marking a book unread and then opening it again started from the
// first page — the one thing a reader who pressed it by accident cannot undo.
func TestMarkingABookUnreadKeepsThePlace(t *testing.T) {
	store := openReading(t)
	account := "reader"
	if _, err := store.Put(account, Position{LibraryID: "books", BookID: "one", CFI: "chapter-9", Fraction: .42, UpdatedAt: 1000}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetStatus(account, "books", "one", "unread"); err != nil {
		t.Fatal(err)
	}

	kept, ok := store.Get(account, "books", "one")
	if !ok {
		t.Fatal("the position was deleted")
	}
	if kept.CFI != "chapter-9" || kept.Fraction != .42 {
		t.Fatalf("the place was lost: cfi=%q fraction=%v", kept.CFI, kept.Fraction)
	}
	if kept.Finished {
		t.Fatal("an unread book cannot be finished")
	}
	if kept.Status != "unread" {
		t.Fatalf("status=%q, want unread so every surface can label it", kept.Status)
	}
	if !kept.Dismissed {
		t.Fatal("an unread book must not sit in the continue-reading rails")
	}
}

// The "on hold" shelf is for books the reader parked deliberately; unread is not one of them.
func TestAnUnreadBookIsNotOnTheHeldShelf(t *testing.T) {
	store := openReading(t)
	account := "reader"
	if _, err := store.Put(account, Position{LibraryID: "books", BookID: "one", Fraction: .3, UpdatedAt: 1000}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(account, Position{LibraryID: "books", BookID: "two", Fraction: .5, UpdatedAt: 900}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetStatus(account, "books", "one", "unread"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetStatus(account, "books", "two", "on_hold"); err != nil {
		t.Fatal(err)
	}

	held := store.ListHeld(account, 10)
	if len(held) != 1 || held[0].BookID != "two" {
		t.Fatalf("held=%v, want only the parked book", held)
	}
}

// Reading it again puts it back in the ordinary flow.
func TestReadingAnUnreadBookAgainClearsTheLabel(t *testing.T) {
	store := openReading(t)
	account := "reader"
	if _, err := store.Put(account, Position{LibraryID: "books", BookID: "one", Fraction: .4, UpdatedAt: 1000}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetStatus(account, "books", "one", "unread"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetStatus(account, "books", "one", "reading"); err != nil {
		t.Fatal(err)
	}
	position, ok := store.Get(account, "books", "one")
	if !ok || position.Status != "" || position.Dismissed {
		t.Fatalf("still labelled after reading resumed: %+v", position)
	}
	if position.Fraction != .4 {
		t.Fatalf("fraction=%v, want the kept .4", position.Fraction)
	}
}
