package annotations

import (
	"fmt"
	"runtime"
	"testing"

	"mochil/internal/reading"
)

// seedUnrelatedAccounts populates the store with n accounts, each with one
// publication holding a few annotations, that the measured write below never
// touches. A write to one account's one publication must cost roughly the
// same whether these unrelated accounts exist or not: per-write cost must
// scale with the record being touched, not with everyone else's data.
func seedUnrelatedAccounts(store *Store, n int) {
	for index := 0; index < n; index++ {
		account := fmt.Sprintf("unrelated-%d", index)
		books := map[string]publication{}
		items := map[string]Annotation{}
		for j := 0; j < 5; j++ {
			id := fmt.Sprintf("item-%d", j)
			items[id] = Annotation{
				ID: id, Kind: "highlight", Quote: "some unrelated passage",
				Anchor:   &reading.Anchor{Exact: "some unrelated passage"},
				Locators: map[string]string{"readium": "chapter-location"},
			}
		}
		books[publicationKey("books", "book")] = publication{Revision: 5, Items: items}
		store.data.Accounts[account] = books
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

// TestApplyCostDoesNotScaleWithUnrelatedAccounts proves that applying
// changes to one account's one publication does not allocate proportionally
// to the total number of tracked accounts/publications in the store. Today
// Apply (via cloneData) deep-clones every account's every annotation on
// every batch, so this fails against a store holding many unrelated
// accounts.
func TestApplyCostDoesNotScaleWithUnrelatedAccounts(t *testing.T) {
	small, _ := Open("")
	big, _ := Open("")
	seedUnrelatedAccounts(big, 6000)

	change := func() []Change {
		return []Change{{Annotation: Annotation{ID: "measured", Kind: "highlight", Quote: "measured quote"}}}
	}

	smallCost := measureAllocatedBytes(t, func() {
		if _, err := small.Apply("measured", "books", "one", change()); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if _, err := big.Apply("measured", "books", "one", change()); err != nil {
			t.Fatal(err)
		}
	})

	// Generous slack for incidental allocation noise; a real O(n) clone of
	// 6000 unrelated accounts allocates orders of magnitude more than this.
	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("Apply allocated %d bytes with 6000 unrelated accounts present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size, not with the one record written",
			bigCost, smallCost, limit)
	}
}

// TestDeleteAccountCostDoesNotScaleWithUnrelatedAccounts is the same proof
// for DeleteAccount.
func TestDeleteAccountCostDoesNotScaleWithUnrelatedAccounts(t *testing.T) {
	setup := func(n int) *Store {
		store, _ := Open("")
		seedUnrelatedAccounts(store, n)
		if _, err := store.Apply("measured", "books", "one", []Change{{Annotation: Annotation{ID: "first", Kind: "bookmark"}}}); err != nil {
			t.Fatal(err)
		}
		return store
	}

	small := setup(0)
	big := setup(6000)

	smallCost := measureAllocatedBytes(t, func() {
		if _, err := small.DeleteAccount("measured"); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if _, err := big.DeleteAccount("measured"); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("DeleteAccount allocated %d bytes with 6000 unrelated accounts present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size", bigCost, smallCost, limit)
	}
}
