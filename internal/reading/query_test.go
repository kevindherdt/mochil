package reading

import (
	"fmt"
	"testing"
)

func TestListLimitReturnsOnlyNewestPositionsDeterministically(t *testing.T) {
	store, _ := Open("")
	store.data.Accounts["reader"] = map[string]Position{
		readingKey("books", "old"):   {LibraryID: "books", BookID: "old", UpdatedAt: 1},
		readingKey("books", "beta"):  {LibraryID: "books", BookID: "beta", UpdatedAt: 3},
		readingKey("books", "alpha"): {LibraryID: "books", BookID: "alpha", UpdatedAt: 3},
		readingKey("books", "done"):  {LibraryID: "books", BookID: "done", UpdatedAt: 9, Finished: true},
	}
	positions := store.ListLimit("reader", false, 2)
	if len(positions) != 2 || positions[0].BookID != "alpha" || positions[1].BookID != "beta" {
		t.Fatalf("limited positions = %+v", positions)
	}
	positions[0].BookID = "mutated"
	if stored, _ := store.Get("reader", "books", "alpha"); stored.BookID != "alpha" {
		t.Fatal("limited list exposed mutable store state")
	}
}

func TestListLibraryLimitFiltersBeforeApplyingLimit(t *testing.T) {
	store, _ := Open("")
	store.data.Accounts["reader"] = map[string]Position{
		readingKey("other", "newest"): {LibraryID: "other", BookID: "newest", UpdatedAt: 9},
		readingKey("books", "new"):    {LibraryID: "books", BookID: "new", UpdatedAt: 7},
		readingKey("books", "old"):    {LibraryID: "books", BookID: "old", UpdatedAt: 3},
	}
	positions := store.ListLibraryLimit("reader", false, "books", 2)
	if len(positions) != 2 || positions[0].BookID != "new" || positions[1].BookID != "old" {
		t.Fatalf("library positions = %+v", positions)
	}
}

func BenchmarkListLimitLargeAccount(b *testing.B) {
	store, _ := Open("")
	positions := make(map[string]Position, 100_000)
	for index := range 100_000 {
		id := fmt.Sprintf("book-%06d", index)
		positions[readingKey("books", id)] = Position{LibraryID: "books", BookID: id, UpdatedAt: int64(index)}
	}
	store.data.Accounts["reader"] = positions
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if got := store.ListLimit("reader", false, 100); len(got) != 100 {
			b.Fatal(len(got))
		}
	}
}
