package catalog

import (
	"fmt"
	"runtime"
	"testing"
)

// seedUnrelatedLibraries populates the override store with n libraries, each
// with an overridden book, that the measured write below never touches. A
// write to one library's one book override must cost roughly the same
// whether these unrelated libraries exist or not: per-write cost must scale
// with the record being touched, not with everyone else's data.
func seedUnrelatedLibraries(store *OverrideStore, n int) {
	for index := 0; index < n; index++ {
		libraryID := fmt.Sprintf("unrelated-library-%d", index)
		store.data.Libraries[libraryID] = map[string]MetadataOverride{
			"book": {
				Title: "Some Unrelated Title", Authors: []string{"Author One", "Author Two"},
				Tags: []string{"tag-a", "tag-b"}, Identifiers: map[string]string{"isbn": "978-0-00-000000-0"},
			},
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

// TestSetArtworkCostDoesNotScaleWithUnrelatedLibraries proves that setting
// one book's artwork override does not allocate proportionally to the total
// number of libraries/books tracked. Today SetArtwork (via
// cloneOverrideLibraries) deep-clones every library's every override on
// every write, so this fails against a store holding many unrelated
// libraries.
func TestSetArtworkCostDoesNotScaleWithUnrelatedLibraries(t *testing.T) {
	small, _ := OpenOverrideStore("")
	big, _ := OpenOverrideStore("")
	seedUnrelatedLibraries(big, 6000)

	smallCost := measureAllocatedBytes(t, func() {
		if err := small.SetArtwork("measured-library", "measured-book", "https://example.com/cover.jpg", "provider", "id"); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if err := big.SetArtwork("measured-library", "measured-book", "https://example.com/cover.jpg", "provider", "id"); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("SetArtwork allocated %d bytes with 6000 unrelated libraries present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size, not with the one record written",
			bigCost, smallCost, limit)
	}
}

// TestApplyWithUndoCostDoesNotScaleWithUnrelatedLibraries is the same proof
// for ApplyWithUndo.
func TestApplyWithUndoCostDoesNotScaleWithUnrelatedLibraries(t *testing.T) {
	small, _ := OpenOverrideStore("")
	big, _ := OpenOverrideStore("")
	seedUnrelatedLibraries(big, 6000)

	changes := map[string]map[string]string{"measured-book": {"title": "Measured Title"}}

	smallCost := measureAllocatedBytes(t, func() {
		if _, err := small.ApplyWithUndo("measured-library", changes); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if _, err := big.ApplyWithUndo("measured-library", changes); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("ApplyWithUndo allocated %d bytes with 6000 unrelated libraries present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size", bigCost, smallCost, limit)
	}
}

// TestResetCostDoesNotScaleWithUnrelatedLibraries is the same proof for
// Reset.
func TestResetCostDoesNotScaleWithUnrelatedLibraries(t *testing.T) {
	setup := func(n int) *OverrideStore {
		store, _ := OpenOverrideStore("")
		seedUnrelatedLibraries(store, n)
		if err := store.SetArtwork("measured-library", "measured-book", "https://example.com/cover.jpg", "provider", "id"); err != nil {
			t.Fatal(err)
		}
		if err := store.Apply("measured-library", map[string]map[string]string{"measured-book": {"title": "Measured Title"}}); err != nil {
			t.Fatal(err)
		}
		return store
	}

	small := setup(0)
	big := setup(6000)

	smallCost := measureAllocatedBytes(t, func() {
		if _, err := small.Reset("measured-library", "measured-book", "title"); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if _, err := big.Reset("measured-library", "measured-book", "title"); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("Reset allocated %d bytes with 6000 unrelated libraries present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size", bigCost, smallCost, limit)
	}
}
