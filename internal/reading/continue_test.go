package reading

import "testing"

// Kevin's rule for the continue shelf: most recently read first, and a book that was only
// opened on page one as a look never shows up. Finished, dismissed, on-hold and dropped books
// stay off it as well — they are not "in progress".
func TestListContinueIsWhatIsActuallyBeingRead(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	store.data.Accounts["reader"] = map[string]Position{
		readingKey("books", "reading-newest"): {LibraryID: "books", BookID: "reading-newest", Fraction: .42, UpdatedAt: 900},
		readingKey("books", "reading-older"):  {LibraryID: "books", BookID: "reading-older", Fraction: .10, UpdatedAt: 500},
		readingKey("books", "just-opened"):    {LibraryID: "books", BookID: "just-opened", Fraction: 0, UpdatedAt: 950},
		readingKey("books", "finished"):       {LibraryID: "books", BookID: "finished", Fraction: 1, UpdatedAt: 980, Finished: true},
		readingKey("books", "dismissed"):      {LibraryID: "books", BookID: "dismissed", Fraction: .3, UpdatedAt: 990, Dismissed: true},
		readingKey("books", "on-hold"):        {LibraryID: "books", BookID: "on-hold", Fraction: .5, UpdatedAt: 995, Status: StatusOnHold},
		readingKey("books", "dropped"):        {LibraryID: "books", BookID: "dropped", Fraction: .5, UpdatedAt: 996, Status: StatusDropped},
		readingKey("other", "elsewhere"):      {LibraryID: "other", BookID: "elsewhere", Fraction: .7, UpdatedAt: 999},
	}

	positions := store.ListContinue("reader", "books", 0)
	if len(positions) != 2 {
		t.Fatalf("continue = %d entries (%v), want only the 2 books actually being read", len(positions), ids(positions))
	}
	if positions[0].BookID != "reading-newest" || positions[1].BookID != "reading-older" {
		t.Fatalf("order = %v, want most recently read first", ids(positions))
	}
}

func TestListContinueLimitsAfterFiltering(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	store.data.Accounts["reader"] = map[string]Position{
		readingKey("books", "newest"): {LibraryID: "books", BookID: "newest", Fraction: .2, UpdatedAt: 300},
		readingKey("books", "older"):  {LibraryID: "books", BookID: "older", Fraction: .2, UpdatedAt: 200},
	}
	// A finished book with a newer timestamp must not occupy the single slot.
	store.data.Accounts["reader"][readingKey("books", "done")] = Position{LibraryID: "books", BookID: "done", Fraction: 1, UpdatedAt: 400, Finished: true}

	positions := store.ListContinue("reader", "", 1)
	if len(positions) != 1 || positions[0].BookID != "newest" {
		t.Fatalf("limited continue = %v, want [newest]", ids(positions))
	}
}

func ids(positions []Position) []string {
	out := make([]string, 0, len(positions))
	for _, position := range positions {
		out = append(out, position.BookID)
	}
	return out
}
