package externallibrary

import (
	"fmt"
	"runtime"
	"testing"
)

// seedUnrelatedAccounts populates the store with n accounts, each with one
// title and some chapter progress, that the measured write below never
// touches. A write to one account's one title/chapter must cost roughly the
// same whether these unrelated accounts exist or not: per-write cost must
// scale with the record being touched, not with everyone else's data.
func seedUnrelatedAccounts(store *Store, n int) {
	for index := 0; index < n; index++ {
		account := fmt.Sprintf("unrelated-%d", index)
		title := Title{
			ID: StableID("provider", "source", "remote-title"), ProviderID: "provider", SourceID: "source",
			RemoteTitleID: "remote-title", Name: "Some Unrelated Book", ExecutionLocation: ExecutionServer,
			AddedAt: 1000, UpdatedAt: 1000,
		}
		progress := map[string]Progress{
			"chapter-one": {TitleID: title.ID, RemoteChapterID: "chapter-one", Fraction: .3, Revision: 1, UpdatedAt: 1000},
		}
		store.data.Accounts[account] = accountData{
			Titles:   map[string]Title{title.ID: title},
			Progress: map[string]map[string]Progress{title.ID: progress},
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

// TestPutCostDoesNotScaleWithUnrelatedAccounts proves that saving one
// account's one title does not allocate proportionally to the total number
// of tracked accounts/titles in the store. Today Put (via cloneData)
// deep-clones every account's every title and chapter progress on every
// write, so this fails against a store holding many unrelated accounts.
func TestPutCostDoesNotScaleWithUnrelatedAccounts(t *testing.T) {
	small, _ := Open("")
	big, _ := Open("")
	seedUnrelatedAccounts(big, 6000)

	title := func() Title {
		return Title{ProviderID: "p", SourceID: "s", RemoteTitleID: "measured-title", Name: "Measured Book", ExecutionLocation: ExecutionServer}
	}

	smallCost := measureAllocatedBytes(t, func() {
		if _, err := small.Put("measured", title()); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if _, err := big.Put("measured", title()); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("Put allocated %d bytes with 6000 unrelated accounts present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size, not with the one record written",
			bigCost, smallCost, limit)
	}
}

// TestPutProgressCostDoesNotScaleWithUnrelatedAccounts is the same proof for
// PutProgress, which is hit on every reading-progress update for an
// externally sourced title.
func TestPutProgressCostDoesNotScaleWithUnrelatedAccounts(t *testing.T) {
	setup := func(n int) *Store {
		store, _ := Open("")
		seedUnrelatedAccounts(store, n)
		if _, err := store.Put("measured", Title{ProviderID: "p", SourceID: "s", RemoteTitleID: "measured-title", Name: "Measured Book", ExecutionLocation: ExecutionServer}); err != nil {
			t.Fatal(err)
		}
		return store
	}
	titleID := StableID("p", "s", "measured-title")

	small := setup(0)
	big := setup(6000)

	smallCost := measureAllocatedBytes(t, func() {
		if _, err := small.PutProgress("measured", titleID, Progress{RemoteChapterID: "chapter-one", Fraction: .4}, 0); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if _, err := big.PutProgress("measured", titleID, Progress{RemoteChapterID: "chapter-one", Fraction: .4}, 0); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("PutProgress allocated %d bytes with 6000 unrelated accounts present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size", bigCost, smallCost, limit)
	}
}

// TestRemoveCostDoesNotScaleWithUnrelatedAccounts is the same proof for
// Remove.
func TestRemoveCostDoesNotScaleWithUnrelatedAccounts(t *testing.T) {
	setup := func(n int) (*Store, string) {
		store, _ := Open("")
		seedUnrelatedAccounts(store, n)
		title, err := store.Put("measured", Title{ProviderID: "p", SourceID: "s", RemoteTitleID: "measured-title", Name: "Measured Book", ExecutionLocation: ExecutionServer})
		if err != nil {
			t.Fatal(err)
		}
		return store, title.ID
	}

	small, smallID := setup(0)
	big, bigID := setup(6000)

	smallCost := measureAllocatedBytes(t, func() {
		if err := small.Remove("measured", smallID); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if err := big.Remove("measured", bigID); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("Remove allocated %d bytes with 6000 unrelated accounts present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size", bigCost, smallCost, limit)
	}
}

// TestDeleteAccountCostDoesNotScaleWithUnrelatedAccounts is the same proof
// for DeleteAccount.
func TestDeleteAccountCostDoesNotScaleWithUnrelatedAccounts(t *testing.T) {
	setup := func(n int) *Store {
		store, _ := Open("")
		seedUnrelatedAccounts(store, n)
		if _, err := store.Put("measured", Title{ProviderID: "p", SourceID: "s", RemoteTitleID: "measured-title", Name: "Measured Book", ExecutionLocation: ExecutionServer}); err != nil {
			t.Fatal(err)
		}
		return store
	}

	small := setup(0)
	big := setup(6000)

	smallCost := measureAllocatedBytes(t, func() {
		if err := small.DeleteAccount("measured"); err != nil {
			t.Fatal(err)
		}
	})
	bigCost := measureAllocatedBytes(t, func() {
		if err := big.DeleteAccount("measured"); err != nil {
			t.Fatal(err)
		}
	})

	if limit := smallCost*10 + 64*1024; bigCost > limit {
		t.Fatalf("DeleteAccount allocated %d bytes with 6000 unrelated accounts present vs %d bytes on an empty store "+
			"(limit %d): write cost scales with total store size", bigCost, smallCost, limit)
	}
}
