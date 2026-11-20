package reading

import (
	"testing"
	"time"
)

// The same publication can sit in two libraries, and both copies can be linked
// to the same KoSync document. A push carries only the document hash, so the
// copy it lands on has to be the one the reader was last in — not whichever one
// the account map happened to yield first. Highlights, notes and the progress a
// device pulls back all follow this lookup.
func TestFindKoSyncDocumentPicksTheCopyLastRead(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	const document = "0123456789abcdef0123456789abcdef"
	revisions := map[string]int64{}
	for _, ids := range [][2]string{{"one", "copy-a"}, {"two", "copy-b"}} {
		linked, err := store.LinkKoSync("reader", ids[0], ids[1], document)
		if err != nil {
			t.Fatalf("link %s: %v", ids[0], err)
		}
		revisions[ids[0]] = linked.Revision
	}
	if _, err := store.Put("reader", Position{LibraryID: "one", BookID: "copy-a", Fraction: .5}, revisions["one"]); err != nil {
		t.Fatalf("save copy-a: %v", err)
	}
	// UpdatedAt is stamped from the clock, so let the millisecond pass before the
	// second save; that save is then the one the reader was last in.
	time.Sleep(3 * time.Millisecond)
	if _, err := store.Put("reader", Position{LibraryID: "two", BookID: "copy-b", Fraction: .8}, revisions["two"]); err != nil {
		t.Fatalf("save copy-b: %v", err)
	}

	first, found := store.FindKoSyncDocument("reader", document)
	if !found {
		t.Fatal("document is linked but no copy was found")
	}
	if first.LibraryID != "two" || first.BookID != "copy-b" {
		t.Fatalf("resolved to %s/%s at %.2f, want two/copy-b at 0.80", first.LibraryID, first.BookID, first.Fraction)
	}
	for call := 0; call < 200; call++ {
		again, _ := store.FindKoSyncDocument("reader", document)
		if again.LibraryID != first.LibraryID || again.BookID != first.BookID {
			t.Fatalf("call %d resolved to %s/%s, call 1 to %s/%s: the same push lands on a different copy every time",
				call, again.LibraryID, again.BookID, first.LibraryID, first.BookID)
		}
	}
}
