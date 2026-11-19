package bookstats

import (
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestACountIsKeptAcrossRunsButOnlyForTheSameVersionOfTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book-stats.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	modified := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	store.Put("books", "a1", 1000, modified, Stats{Words: 80000, Pages: 320})
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := again.Get("books", "a1", 1000, modified); !ok || got.Words != 80000 || got.Pages != 320 {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
	if _, ok := again.Get("books", "a1", 1001, modified); ok {
		t.Fatal("a file that changed size must be counted again")
	}
	if _, ok := again.Get("books", "a1", 1000, modified.Add(time.Second)); ok {
		t.Fatal("a file that was modified must be counted again")
	}
	if _, ok := again.Get("books", "other", 1000, modified); ok {
		t.Fatal("a book that was never counted has no numbers")
	}
}

// TestSaveHoldsTheLockForTheWholeWriteNotJustTheMarshal proves Save()'s disk write (temp-create, write,
// rename) is covered by the same lock as the in-memory marshal, not released early. Without that, two
// concurrent Save() calls can finish their disk writes out of order relative to when they captured their
// snapshot: an older snapshot's write landing after a newer one's silently reverts the file on disk even
// though the newer data is still correct in memory (a lost update discovered only after a restart).
func TestSaveHoldsTheLockForTheWholeWriteNotJustTheMarshal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book-stats.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	modified := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	store.Put("books", "a1", 1000, modified, Stats{Words: 1})

	entered := make(chan struct{})
	release := make(chan struct{})
	var hookFired atomic.Bool
	saveTestHook = func() {
		if hookFired.CompareAndSwap(false, true) {
			close(entered)
			<-release
		}
	}
	defer func() { saveTestHook = nil }()

	firstDone := make(chan error, 1)
	go func() { firstDone <- store.Save() }()
	<-entered // the first Save() is now paused mid-write, still holding the lock

	secondDone := make(chan struct{})
	go func() {
		store.Put("books", "a2", 2000, modified, Stats{Words: 2})
		store.Save()
		close(secondDone)
	}()

	select {
	case <-secondDone:
		t.Fatal("a second Put+Save completed while the first Save() still held the lock mid-write -- the lock does not cover the whole write, so a lost update is possible")
	case <-time.After(50 * time.Millisecond):
		// still blocked behind the first Save's lock, as required
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	<-secondDone

	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := again.Get("books", "a1", 1000, modified); !ok {
		t.Fatal("the first Save's data must survive on disk")
	}
	if _, ok := again.Get("books", "a2", 2000, modified); !ok {
		t.Fatal("the second Save's data must survive on disk, not be lost to an out-of-order write")
	}
}
