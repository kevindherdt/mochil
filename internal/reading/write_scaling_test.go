package reading

import (
	"fmt"
	"runtime"
	"testing"
)

// seedUnrelatedAccounts populates the store with n accounts, each with one
// book position and a short history, that the measured write below never
// touches. A write to one account's one book must cost roughly the same
// whether these unrelated accounts exist or not: per-write cost must scale
// with the record being touched, not with everyone else's data.
func seedUnrelatedAccounts(store *Store, n int) {
	for index := 0; index < n; index++ {
		account := fmt.Sprintf("unrelated-%d", index)
		key := readingKey("books", "book")
		position := Position{
			LibraryID: "books", BookID: "book", Revision: 3, Fraction: .42,
			Locators:  map[string]string{"readium": "chapter-location"},
			WebAnchor: &Anchor{SpineIndex: 2, Exact: "some passage", Prefix: "before", Suffix: "after"},
		}
		store.data.Accounts[account] = map[string]Position{key: position}
		store.data.History[account] = map[string][]Position{
			key: {clonePosition(position), clonePosition(position), clonePosition(position)},
		}
	}
}

// measureAllocatedBytes returns the heap bytes allocated by fn, with a GC
// just beforehand so unrelated garbage does not pollute the measurement.
func measureAllocatedBytes(t *testing.T, fn func()) uint64 {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// TestPutCostDoesNotScaleWithUnrelatedAccounts proves that saving one
// account's one book position does not allocate proportionally to the total
// number of tracked accounts/books in the store. Today Put (via
// cloneStoreData) deep-clones every account and every history entry on every
// write, so this fails against a store holding many unrelated accounts.
func TestPutCostDoesNotScaleWithUnrelatedAccounts(t *testing.T) {
	small, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	big, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	seedUnrelatedAccounts(big, 6000)

	smallCost := measureAllocatedBytes(t, func() {
		if _, err := small.Put("measured", Position{LibraryID: "books", BookID: "one", Fraction: .3}, 0); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if _, err := big.Put("measured", Position{LibraryID: "books", BookID: "one", Fraction: .3}, 0); err != nil {
			t.Fatal(err)
		}
	})

	// Generous slack for incidental allocation noise; a real O(n) clone of
	// 6000 unrelated accounts allocates orders of magnitude more than this.
	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("Put allocated %d bytes with 6000 unrelated accounts present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size, not with the one record written",
			bigCost, smallCost, limit)
	}
}

// TestSetDismissedCostDoesNotScaleWithUnrelatedAccounts is the same proof for
// SetDismissed, which also went through a full-store clone before persisting.
func TestSetDismissedCostDoesNotScaleWithUnrelatedAccounts(t *testing.T) {
	small, _ := Open("")
	if _, err := small.Put("measured", Position{LibraryID: "books", BookID: "one", Fraction: .3}, 0); err != nil {
		t.Fatal(err)
	}
	big, _ := Open("")
	seedUnrelatedAccounts(big, 6000)
	if _, err := big.Put("measured", Position{LibraryID: "books", BookID: "one", Fraction: .3}, 0); err != nil {
		t.Fatal(err)
	}

	smallCost := measureAllocatedBytes(t, func() {
		if _, err := small.SetDismissed("measured", "books", "one", true); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if _, err := big.SetDismissed("measured", "books", "one", true); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("SetDismissed allocated %d bytes with 6000 unrelated accounts present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size", bigCost, smallCost, limit)
	}
}

// TestRestoreHistoryCostDoesNotScaleWithUnrelatedAccounts is the same proof
// for RestoreHistory.
func TestRestoreHistoryCostDoesNotScaleWithUnrelatedAccounts(t *testing.T) {
	setup := func(n int) (*Store, int64) {
		store, _ := Open("")
		seedUnrelatedAccounts(store, n)
		first, err := store.Put("measured", Position{LibraryID: "books", BookID: "one", Fraction: .3, CFI: "a"}, 0)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Put("measured", Position{LibraryID: "books", BookID: "one", Fraction: .6, CFI: "b"}, first.Revision); err != nil {
			t.Fatal(err)
		}
		return store, first.UpdatedAt
	}

	small, updatedAt := setup(0)
	big, bigUpdatedAt := setup(6000)

	smallCost := measureAllocatedBytes(t, func() {
		if _, err := small.RestoreHistory("measured", "books", "one", updatedAt); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if _, err := big.RestoreHistory("measured", "books", "one", bigUpdatedAt); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("RestoreHistory allocated %d bytes with 6000 unrelated accounts present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size", bigCost, smallCost, limit)
	}
}
