package web

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"mochil/internal/accounts"
	"mochil/internal/catalog"
	"mochil/internal/reading"
)

func TestKoSyncProgressResolvesStableAccountWithoutCrossAccountReplay(t *testing.T) {
	directory := t.TempDir()
	accountStore, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := accountStore.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	readingStore, err := reading.Open(filepath.Join(directory, "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	book := catalog.Book{ID: "book", Path: "Example Book.epub", Title: "Example Book"}
	server := &multiServer{
		cfg:         MultiConfig{Username: "owner"},
		accounts:    accountStore,
		reading:     readingStore,
		readingSync: reading.NewSyncCoordinator(readingStore),
		libraries:   []Library{{ID: "books", Name: "Books", Store: staticBackend{books: []catalog.Book{book}}}},
	}
	document := koSyncFilenameMD5(filepath.Base(book.Path))
	server.recordKoSyncProgress("reader", koSyncProgress{Document: document, Progress: "reader-location", Percentage: .4, Timestamp: 20})
	server.recordKoSyncProgress("owner", koSyncProgress{Document: document, Progress: "owner-location", Percentage: .7, Timestamp: 30})
	server.recordKoSyncProgress("unknown", koSyncProgress{Document: document, Progress: "must-not-save", Percentage: .9, Timestamp: 40})
	readerPosition, readerOK := readingStore.Get(reader.ID, "books", book.ID)
	ownerPosition, ownerOK := readingStore.Get("owner", "books", book.ID)
	if !readerOK || readerPosition.KoReaderLocator != "reader-location" || !ownerOK || ownerPosition.KoReaderLocator != "owner-location" {
		t.Fatalf("reader=%+v owner=%+v", readerPosition, ownerPosition)
	}
	if _, exists := readingStore.Get("unknown", "books", book.ID); exists {
		t.Fatal("unknown KoSync username created account-owned reading state")
	}
}

func TestReadingStatePersistenceConflictFinishAndRestore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reading-state.json")
	store, err := reading.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Put("reader", readingPosition{LibraryID: "books", BookID: "one", CFI: "epubcfi(/6/2)", Fraction: .4, DeviceID: "phone"}, 0)
	if err != nil || first.Revision != 1 {
		t.Fatalf("first save: %+v %v", first, err)
	}
	if _, err := store.Put("reader", readingPosition{LibraryID: "books", BookID: "one", CFI: "old", Fraction: .2}, 0); !errors.Is(err, errReadingConflict) {
		t.Fatalf("stale write returned %v", err)
	}
	finished, err := store.Put("reader", readingPosition{LibraryID: "books", BookID: "one", CFI: "epubcfi(/6/8)", Fraction: .98, DeviceID: "tablet"}, first.Revision)
	if err != nil || !finished.Finished || finished.FinishedAt == 0 {
		t.Fatalf("finish: %+v %v", finished, err)
	}
	if len(store.List("reader", false)) != 0 || len(store.List("reader", true)) != 1 {
		t.Fatal("finished filtering failed")
	}
	dismissed, err := store.SetDismissed("reader", "books", "one", true)
	if err != nil || !dismissed.Dismissed || len(store.List("reader", true)) != 0 {
		t.Fatalf("dismiss: %+v %v", dismissed, err)
	}
	shown, err := store.SetDismissed("reader", "books", "one", false)
	if err != nil || shown.Dismissed || len(store.List("reader", true)) != 1 {
		t.Fatalf("show: %+v %v", shown, err)
	}
	seen, err := store.MarkKoSyncSeen("reader", "books", "one", 12345)
	if err != nil || seen.KoSyncSeenAt != 12345 {
		t.Fatalf("mark KoSync seen: %+v %v", seen, err)
	}
	linked, err := store.LinkKoSync("reader", "books", "one", "0123456789abcdef0123456789abcdef")
	if err != nil || linked.KoSyncDocument == "" {
		t.Fatalf("link KoSync: %+v %v", linked, err)
	}
	accepted, err := store.AcceptKoSync("reader", "books", "one", 23456, readingPosition{CFI: "epubcfi(/6/12)", Fraction: .6, WebAnchor: &webAnchor{Text: "A sufficiently long location anchor", SpineIndex: 5}})
	if err != nil || accepted.CFI != "epubcfi(/6/12)" || accepted.Fraction != .6 || accepted.KoSyncSeenAt != 23456 {
		t.Fatalf("accept KoSync: %+v %v", accepted, err)
	}
	if accepted.UpdatedAt != linked.UpdatedAt {
		t.Fatalf("accepted KOReader position became a newer web edit: %d != %d", accepted.UpdatedAt, linked.UpdatedAt)
	}
	restored, err := store.SetFinished("reader", "books", "one", false)
	if err != nil || restored.Finished || restored.Fraction >= .98 {
		t.Fatalf("restore: %+v %v", restored, err)
	}

	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := reading.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	position, ok := reloaded.Get("reader", "books", "one")
	if !ok || position.CFI != "epubcfi(/6/12)" || position.Revision != 8 || position.KoSyncSeenAt != 23456 || position.KoSyncDocument == "" {
		t.Fatalf("reloaded: %+v", position)
	}
	if _, ok := reloaded.Get("another-account", "books", "one"); ok {
		t.Fatal("reading state leaked between accounts")
	}
}

func TestKoReaderTextAnchorIsStoredWithNativeLocator(t *testing.T) {
	store, err := reading.Open(filepath.Join(t.TempDir(), "reading-state.json"))
	if err != nil {
		t.Fatal(err)
	}
	anchor := &webAnchor{Version: 2, Prefix: "words before", Exact: "a sufficiently distinctive sequence of words at the reading position", Suffix: "words after", SpineIndex: 3, Progress: .42}
	position, err := store.RecordKoSync("reader", "books", "one", reading.SyncProgress{Document: "0123456789abcdef0123456789abcdef", Progress: "/body/DocFragment[4]/body/p[2]/text().7", Percentage: .42, Timestamp: 123}, anchor)
	if err != nil {
		t.Fatal(err)
	}
	if position.KoReaderLocator == "" || position.WebAnchor == nil || position.WebAnchor.Exact != anchor.Exact {
		t.Fatalf("hybrid position was not stored: %+v", position)
	}
}

func TestReadingPositionHistoryKeepsAndRestoresRecentLocations(t *testing.T) {
	store, err := reading.Open(filepath.Join(t.TempDir(), "reading-state.json"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Put("reader", readingPosition{LibraryID: "books", BookID: "one", CFI: "first", Fraction: .2, DeviceName: "Laptop"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	second, err := store.Put("reader", readingPosition{LibraryID: "books", BookID: "one", CFI: "second", Fraction: .6, DeviceName: "Tablet"}, first.Revision)
	if err != nil {
		t.Fatal(err)
	}
	history := store.History("reader", "books", "one")
	if len(history) != 2 || history[0].CFI != "second" || history[1].CFI != "first" {
		t.Fatalf("history = %+v", history)
	}
	restored, err := store.RestoreHistory("reader", "books", "one", first.UpdatedAt)
	if err != nil || restored.CFI != "first" || restored.Revision != second.Revision+1 {
		t.Fatalf("restored = %+v, %v", restored, err)
	}
}
