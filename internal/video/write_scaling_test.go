package video

import (
	"fmt"
	"runtime"
	"testing"
)

// seedUnrelatedVideoAccounts populates many accounts with progress the write
// under test never touches, so a regression that starts cloning the whole
// store again (as internal/reading's store once did) would show up here.
func seedUnrelatedVideoAccounts(store *ProgressStore, n int) {
	for index := 0; index < n; index++ {
		account := fmt.Sprintf("unrelated-%d", index)
		key := progressKey("library", "item")
		store.data.Accounts[account] = map[string]Progress{
			key: {LibraryID: "library", ItemID: "item", Position: 120, Duration: 3600, Revision: 4},
		}
	}
}

func measureVideoAllocatedBytes(t *testing.T, fn func()) uint64 {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// TestPutCostDoesNotScaleWithUnrelatedAccounts guards against a regression
// that would make video progress writes clone the whole store like
// internal/reading's used to.
func TestPutCostDoesNotScaleWithUnrelatedAccounts(t *testing.T) {
	small, err := OpenProgressStore("")
	if err != nil {
		t.Fatal(err)
	}
	big, err := OpenProgressStore("")
	if err != nil {
		t.Fatal(err)
	}
	seedUnrelatedVideoAccounts(big, 6000)

	smallCost := measureVideoAllocatedBytes(t, func() {
		if err := small.Put("measured", Progress{LibraryID: "library", ItemID: "item", Position: 1, Duration: 100}); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureVideoAllocatedBytes(t, func() {
		if err := big.Put("measured", Progress{LibraryID: "library", ItemID: "item", Position: 1, Duration: 100}); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("Put allocated %d bytes with 6000 unrelated accounts present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size", bigCost, smallCost, limit)
	}
}
