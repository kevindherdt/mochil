package customcolumns

import (
	"fmt"
	"runtime"
	"testing"
)

// seedUnrelatedLibraries populates the store with n libraries, each with one
// book's column values, that the measured write below never touches. A
// write to one library's one book's cells must cost roughly the same
// whether these unrelated libraries exist or not: per-write cost must scale
// with the record being touched, not with everyone else's data.
func seedUnrelatedLibraries(store *Store, column Column, n int) {
	for index := 0; index < n; index++ {
		library := fmt.Sprintf("unrelated-library-%d", index)
		store.data.Values[library] = map[string]map[string]string{
			"book": {column.ID: "unrelated value"},
		}
	}
}

func measureAllocatedBytes(t *testing.T, fn func()) uint64 {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// TestSetCostDoesNotScaleWithUnrelatedLibraries proves that setting one
// book's one column value does not allocate proportionally to the total
// number of libraries/books/columns tracked. Today Set (via cloneLocked)
// deep-clones every library's every book's every column on every edit, so
// this fails against a store holding many unrelated libraries.
func TestSetCostDoesNotScaleWithUnrelatedLibraries(t *testing.T) {
	small, _ := Open("")
	column, err := small.Add("Status", Text)
	if err != nil {
		t.Fatal(err)
	}
	big, _ := Open("")
	big.data.Columns = append([]Column{}, small.data.Columns...)
	seedUnrelatedLibraries(big, column, 6000)

	changes := map[string]map[string]string{"measured-book": {column.ID: "measured value"}}

	smallCost := measureAllocatedBytes(t, func() {
		if _, err := small.Set("measured-library", changes); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if _, err := big.Set("measured-library", changes); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("Set allocated %d bytes with 6000 unrelated libraries present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size, not with the one record written",
			bigCost, smallCost, limit)
	}
}

// TestSetUndoCostDoesNotScaleWithUnrelatedLibraries is the same proof for
// the undo function Set returns.
func TestSetUndoCostDoesNotScaleWithUnrelatedLibraries(t *testing.T) {
	setup := func(n int) (*Store, func() error) {
		store, _ := Open("")
		column, err := store.Add("Status", Text)
		if err != nil {
			t.Fatal(err)
		}
		seedUnrelatedLibraries(store, column, n)
		undo, err := store.Set("measured-library", map[string]map[string]string{"measured-book": {column.ID: "measured value"}})
		if err != nil {
			t.Fatal(err)
		}
		return store, undo
	}

	_, smallUndo := setup(0)
	_, bigUndo := setup(6000)

	smallCost := measureAllocatedBytes(t, func() {
		if err := smallUndo(); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if err := bigUndo(); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("Set's undo allocated %d bytes with 6000 unrelated libraries present vs %d bytes on an empty store "+
			"(limit %d): undo cost scales with total store size", bigCost, smallCost, limit)
	}
}
