package reading

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRejectsMalformedAndOversizedState(t *testing.T) {
	root := t.TempDir()
	trailing := filepath.Join(root, "trailing.json")
	if err := os.WriteFile(trailing, []byte(`{"version":1,"accounts":{}} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(trailing); err == nil {
		t.Fatal("reading state with trailing JSON was accepted")
	}
	oversized := filepath.Join(root, "oversized.json")
	file, err := os.Create(oversized)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(maximumStateBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = Open(oversized); err == nil {
		t.Fatal("oversized reading state was accepted")
	}
}

func TestStorePersistenceConflictAndHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reading.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .2, CFI: "first"}, 0)
	if err != nil || first.Revision != 1 {
		t.Fatalf("first: %#v %v", first, err)
	}
	if _, err = s.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .4}, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("wanted conflict, got %v", err)
	}
	second, err := s.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .6, CFI: "second"}, first.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if history := s.History("reader", "books", "one"); len(history) != 2 {
		t.Fatalf("history: %#v", history)
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Get("reader", "books", "one")
	if !ok || got.Revision != second.Revision || got.CFI != "second" {
		t.Fatalf("reopened: %#v %v", got, ok)
	}
}

func TestStoreTreatsEquivalentStalePositionAsIdempotent(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	current, err := store.Put("reader", Position{
		LibraryID: "books", BookID: "one", Fraction: .5, DeviceID: "tablet",
		Locators: map[string]string{"readium": "authoritative"},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}

	acknowledged, err := store.Put("reader", Position{
		LibraryID: "books", BookID: "one", Fraction: .5015, DeviceID: "phone",
		Locators: map[string]string{"readium": "stale"},
	}, 0)
	if err != nil {
		t.Fatalf("equivalent stale write should be acknowledged: %v", err)
	}
	if acknowledged.Revision != current.Revision || acknowledged.DeviceID != "tablet" || acknowledged.Locators["readium"] != "authoritative" {
		t.Fatalf("equivalent stale write changed authoritative state: %#v", acknowledged)
	}
	if history := store.History("reader", "books", "one"); len(history) != 1 {
		t.Fatalf("idempotent acknowledgement created history: %#v", history)
	}

	if _, err = store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .51}, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("materially different stale write should conflict, got %v", err)
	}
}

func TestStoreNeverTreatsAdjacentFixedLayoutPagesAsEquivalent(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	current, err := store.Put("reader", Position{
		LibraryID: "comics", BookID: "one", Fraction: .5015,
		Locators: map[string]string{"comic": `{"version":1,"page":501,"page_count":1001}`},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Put("reader", Position{
		LibraryID: "comics", BookID: "one", Fraction: .5,
		Locators: map[string]string{"comic": `{"version":1,"page":500,"page_count":1001}`},
	}, 0)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("adjacent comic page should conflict, got %v", err)
	}
	acknowledged, err := store.Put("reader", Position{
		LibraryID: "comics", BookID: "one", Fraction: .5,
		Locators: map[string]string{"pdf": `{"version":1,"page":501,"page_count":1001}`},
	}, 0)
	if err != nil || acknowledged.Revision != current.Revision {
		t.Fatalf("same fixed-layout page should be acknowledged across legacy locator names: %#v %v", acknowledged, err)
	}
	_, err = store.Put("reader", Position{LibraryID: "comics", BookID: "one", Fraction: .5015}, 0)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("a fixed-layout position without an exact page should conflict, got %v", err)
	}
}

func TestStoreAcknowledgesStaleSemanticReadingLocation(t *testing.T) {
	for _, test := range []struct {
		name    string
		current Position
		stale   Position
	}{
		{
			name:    "cfi",
			current: Position{CFI: "epubcfi(/6/4!/4/2:7)"},
			stale:   Position{CFI: "epubcfi(/6/4!/4/2:7)"},
		},
		{
			name:    "koreader xpointer",
			current: Position{KoReaderLocator: "/body/DocFragment[4]/body/div[1]/p[3]/text().2"},
			stale:   Position{KoReaderLocator: "/body/DocFragment[4]/body/div[1]/p[3]/text().2"},
		},
		{
			name:    "contextual text anchor",
			current: Position{WebAnchor: &Anchor{SpineIndex: 3, Prefix: "before", Exact: "the same sentence", Suffix: "after", Progress: .2}},
			stale:   Position{WebAnchor: &Anchor{SpineIndex: 3, Prefix: "before", Exact: "the same sentence", Suffix: "after", Progress: .8}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, err := Open("")
			if err != nil {
				t.Fatal(err)
			}
			current := test.current
			current.LibraryID, current.BookID, current.Fraction, current.DeviceID = "books", "one", .25, "tablet"
			saved, err := store.Put("reader", current, 0)
			if err != nil {
				t.Fatal(err)
			}
			stale := test.stale
			stale.LibraryID, stale.BookID, stale.Fraction, stale.DeviceID = "books", "one", .75, "phone"
			acknowledged, err := store.Put("reader", stale, 0)
			if err != nil || acknowledged.Revision != saved.Revision || acknowledged.DeviceID != "tablet" || acknowledged.Fraction != .25 {
				t.Fatalf("semantic acknowledgement = %#v, %v", acknowledged, err)
			}
		})
	}
}

func TestEquivalentPositionRejectsAmbiguousOrDifferentAnchors(t *testing.T) {
	base := Position{Fraction: .2, WebAnchor: &Anchor{SpineIndex: 1, Prefix: "before", Exact: "repeated words", Suffix: "after"}}
	for _, different := range []Position{
		{Fraction: .8, WebAnchor: &Anchor{SpineIndex: 2, Prefix: "before", Exact: "repeated words", Suffix: "after"}},
		{Fraction: .8, WebAnchor: &Anchor{SpineIndex: 1, Exact: "repeated words"}},
		{Fraction: .8, WebAnchor: &Anchor{SpineIndex: 1, Prefix: "elsewhere", Exact: "repeated words", Suffix: "after"}},
	} {
		if EquivalentPosition(base, different) {
			t.Fatalf("different anchor treated as equivalent: %#v", different)
		}
	}
}

func TestAccountExportIncludesDismissedPositionsAndHistoryWithoutOtherAccounts(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .2, CFI: "first"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .5, CFI: "second"}, first.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.SetDismissed("reader", "books", "one", true); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Put("other", Position{LibraryID: "books", BookID: "private", Fraction: .9}, 0); err != nil {
		t.Fatal(err)
	}
	export := store.ExportAccount("reader")
	if export.Version != 1 || len(export.Items) != 1 || export.Items[0].Position.BookID != "one" || !export.Items[0].Position.Dismissed {
		t.Fatalf("export = %#v", export)
	}
	if len(export.Items[0].History) < 2 || export.Items[0].History[0].Revision < second.Revision {
		t.Fatalf("history = %#v", export.Items[0].History)
	}
}

func TestStoreRejectsInvalidDirectCaller(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: 2}, 0); !errors.Is(err, ErrInvalid) {
		t.Fatalf("wanted invalid, got %v", err)
	}
}

func TestStorePreservesPerClientLocators(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	android, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .2, Locators: map[string]string{"readium": `{"href":"chapter1"}`}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	web, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .3, CFI: "epubcfi(/6/2)", Locators: map[string]string{"foliate": "epubcfi(/6/2)"}}, android.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if web.Locators["readium"] == "" || web.Locators["foliate"] == "" {
		t.Fatalf("client locators were not merged: %#v", web.Locators)
	}
	again, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .4, Locators: map[string]string{"readium": `{"href":"chapter2"}`}}, web.Revision)
	if err != nil || again.Locators["foliate"] != "epubcfi(/6/2)" || again.Locators["readium"] != `{"href":"chapter2"}` {
		t.Fatalf("android update replaced another client's locator: %#v %v", again, err)
	}
	if again.CFI != "" {
		t.Fatalf("a position that moved without a CFI must not keep the old CFI: %#v", again)
	}
	still, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .4005, Locators: map[string]string{"readium": `{"href":"chapter2"}`}}, again.Revision)
	if err != nil || still.CFI != "" {
		t.Fatalf("unexpected: %#v %v", still, err)
	}
	pinned, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .5, CFI: "epubcfi(/6/8)"}, still.Revision)
	if err != nil {
		t.Fatal(err)
	}
	same, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .5005}, pinned.Revision)
	if err != nil || same.CFI != "epubcfi(/6/8)" {
		t.Fatalf("a nudge below the equivalence threshold keeps the CFI: %#v %v", same, err)
	}
}

func TestStoreSnapshotsAndWritesDoNotExposeNestedPositionState(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	anchor := &Anchor{Exact: "original"}
	locators := map[string]string{"readium": "original"}
	written, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .2, WebAnchor: anchor, Locators: locators}, 0)
	if err != nil {
		t.Fatal(err)
	}
	linked, err := store.LinkKoSync("reader", "books", "one", "document")
	if err != nil {
		t.Fatal(err)
	}
	anchor.Exact = "mutated input"
	locators["readium"] = "mutated input"
	written.WebAnchor.Exact = "mutated return"
	written.Locators["readium"] = "mutated return"
	linked.KoSyncDocuments[0] = "mutated return"

	get, _ := store.Get("reader", "books", "one")
	list := store.List("reader", false)
	history := store.History("reader", "books", "one")
	exported := store.ExportAccount("reader")
	get.WebAnchor.Exact = "mutated get"
	get.Locators["readium"] = "mutated get"
	get.KoSyncDocuments[0] = "mutated get"
	list[0].Locators["readium"] = "mutated list"
	history[0].WebAnchor.Exact = "mutated history"
	exported.Items[0].Position.Locators["readium"] = "mutated export"

	fresh, _ := store.Get("reader", "books", "one")
	freshHistory := store.History("reader", "books", "one")
	if fresh.WebAnchor.Exact != "original" || fresh.Locators["readium"] != "original" || fresh.KoSyncDocuments[0] != "document" {
		t.Fatalf("stored position was mutated through an API boundary: %+v", fresh)
	}
	if freshHistory[0].WebAnchor.Exact != "original" {
		t.Fatalf("stored history was mutated through an API boundary: %+v", freshHistory)
	}
}

func TestStorePublishesAndPreservesValidatedAndroidKOReaderLocator(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	native := "/body/DocFragment[4]/body/div[1]/p[3]"
	first, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .2, KoReaderLocator: native}, 0)
	if err != nil || first.KoReaderLocator != native {
		t.Fatalf("native locator was not stored: %#v %v", first, err)
	}
	second, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .3}, first.Revision)
	if err != nil || second.KoReaderLocator != native {
		t.Fatalf("native locator was not preserved: %#v %v", second, err)
	}
}

func TestStoreRetainsMultipleKoSyncBinaryAliases(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	for index, document := range []string{"binary-a", "binary-b"} {
		if _, err = store.LinkKoSync("reader", "books", "one", document); err != nil {
			t.Fatalf("link alias %d: %v", index, err)
		}
	}
	for _, document := range []string{"binary-a", "binary-b"} {
		if position, found := store.FindKoSyncDocument("reader", document); !found || position.BookID != "one" {
			t.Fatalf("alias %q was not retained: %#v", document, position)
		}
	}
	position, _ := store.Get("reader", "books", "one")
	updated, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .3}, position.Revision)
	if err != nil || len(updated.KoSyncDocuments) != 2 {
		t.Fatalf("ordinary update discarded aliases: %#v %v", updated, err)
	}
}

func TestStoreBoundsKoSyncBinaryAliases(t *testing.T) {
	store, _ := Open("")
	for index := range 12 {
		if _, err := store.LinkKoSync("reader", "books", "one", fmt.Sprintf("binary-%d", index)); err != nil {
			t.Fatal(err)
		}
	}
	position, _ := store.Get("reader", "books", "one")
	if len(position.KoSyncDocuments) != 8 || position.KoSyncDocuments[0] != "binary-4" {
		t.Fatalf("aliases were not bounded to the newest eight: %#v", position.KoSyncDocuments)
	}
}

// TestStoreReportsAFailedPersistWithoutDroppingTheChange replaces the old
// rollback test: progress is acknowledged in memory so page loads never wait
// for a whole-file write, which means a failed write can no longer be undone.
// It must be reported instead (Flush/LastError) and kept for the next attempt.
func TestStoreReportsAFailedPersistWithoutDroppingTheChange(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocker, "reading.json")
	position := Position{LibraryID: "books", BookID: "one", Fraction: .25}
	saved, err := store.Put("alice", position, 0)
	if err != nil {
		t.Fatalf("a deferred save must be acknowledged, got %v", err)
	}
	if err := store.Flush(); err == nil {
		t.Fatal("write through an invalid state path succeeded")
	}
	if got, exists := store.Get("alice", "books", "one"); !exists || got.Revision != saved.Revision {
		t.Fatal("failed write dropped the acknowledged change")
	}
	if history := store.History("alice", "books", "one"); len(history) != 1 {
		t.Fatalf("failed write dropped the history entry: %+v", history)
	}
	if store.LastError() == nil {
		t.Fatal("a failed write is not visible through LastError")
	}
}

func TestStoreDropsATextAnchorOnceThePositionMovesWithoutANewOne(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .4, WebAnchor: &Anchor{Exact: "the quoted place at forty percent"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	moved, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .2, CFI: "epubcfi(/6/2)"}, first.Revision)
	if err != nil || moved.WebAnchor != nil {
		t.Fatalf("an anchor from 40%% must not describe a position at 20%%: %#v %v", moved, err)
	}
	anchored, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .5, WebAnchor: &Anchor{Exact: "the quoted place at fifty percent"}}, moved.Revision)
	if err != nil {
		t.Fatal(err)
	}
	nudged, err := store.Put("reader", Position{LibraryID: "books", BookID: "one", Fraction: .5005}, anchored.Revision)
	if err != nil || nudged.WebAnchor == nil {
		t.Fatalf("a nudge inside the equivalence threshold keeps the anchor: %#v %v", nudged, err)
	}
}

func TestKoSyncUpdateWithoutTextDropsTheOldPlace(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Put("reader", Position{
		LibraryID: "books", BookID: "one", Fraction: .10, CFI: "epubcfi(/6/4)",
		WebAnchor: &Anchor{Exact: "an old place", SpineIndex: 3}, Locators: map[string]string{"readium": "old"},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	// A chapter-start position from KOReader cannot be turned into text, so it carries no anchor.
	got, err := s.RecordKoSync("reader", "books", "one", SyncProgress{
		Document: "doc", Progress: "/body/DocFragment[15].0", Percentage: .26, Device: "boox", Timestamp: first.UpdatedAt/1000 + 60,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Fraction != .26 || got.WebAnchor != nil || got.CFI != "" || len(got.Locators) != 0 {
		t.Fatalf("stale place survived a newer KOReader position: %#v", got)
	}
	exact := &Anchor{Exact: "the new place", SpineIndex: 14}
	got, err = s.RecordKoSync("reader", "books", "one", SyncProgress{
		Document: "doc", Progress: "/body/DocFragment[15]/body/p[2]/text().0", Percentage: .27, Device: "boox", Timestamp: first.UpdatedAt/1000 + 120,
	}, exact)
	if err != nil || got.WebAnchor == nil || got.WebAnchor.Exact != "the new place" {
		t.Fatalf("text anchor not stored: %#v %v", got, err)
	}
}

func TestOpenDropsAnAnchorThatDisagreesWithThePercentage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reading.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	stale := Position{LibraryID: "books", BookID: "stale", Fraction: .26, CFI: "epubcfi(/6/4)", WebAnchor: &Anchor{Exact: "old", SpineIndex: 6, Progress: .096}, Locators: map[string]string{"readium": "old"}}
	fine := Position{LibraryID: "books", BookID: "fine", Fraction: .26, WebAnchor: &Anchor{Exact: "here", SpineIndex: 14, Progress: .25}}
	for _, p := range []Position{stale, fine} {
		if _, err = s.Put("reader", p, 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := reopened.Get("reader", "books", "stale"); got.WebAnchor != nil || got.CFI != "" || len(got.Locators) != 0 {
		t.Fatalf("stale place kept: %#v", got)
	}
	if got, _ := reopened.Get("reader", "books", "fine"); got.WebAnchor == nil {
		t.Fatalf("a matching anchor was dropped: %#v", got)
	}
}
