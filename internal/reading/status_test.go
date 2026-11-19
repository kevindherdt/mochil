package reading

import (
	"path/filepath"
	"testing"
)

func TestStatusesMoveABookBetweenTheReadersLists(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	set := func(status string) Position {
		t.Helper()
		position, err := store.SetStatus("a", "lib", "book", status)
		if err != nil {
			t.Fatalf("%s: %v", status, err)
		}
		return position
	}
	if p := set("reading"); p.Fraction <= 0 || p.Finished || p.Dismissed {
		t.Fatalf("reading should count as started: %+v", p)
	}
	if !store.Has("a", false) {
		t.Fatal("a started book belongs in the reading list")
	}
	if p := set("read"); !p.Finished || p.Dismissed || p.Status != "" {
		t.Fatalf("read should finish the book: %+v", p)
	}
	if p := set(StatusOnHold); p.Finished || !p.Dismissed || p.Status != StatusOnHold {
		t.Fatalf("on hold hides it from both lists: %+v", p)
	}
	if store.Has("a", false) || store.Has("a", true) {
		t.Fatal("a held book must not appear on the dashboard")
	}
	if p := set(StatusDropped); p.Status != StatusDropped || !p.Dismissed {
		t.Fatalf("dropped: %+v", p)
	}
	if p, _ := store.SetDismissed("a", "lib", "book", false); p.Status != "" {
		t.Fatalf("showing a book again clears its label: %+v", p)
	}
	set("unread")
	if _, ok := store.Get("a", "lib", "book"); ok {
		t.Fatal("unread forgets the place")
	}
	if _, err := store.SetStatus("a", "lib", "book", "nonsense"); err != ErrUnknownStatus {
		t.Fatalf("want ErrUnknownStatus, got %v", err)
	}
}
