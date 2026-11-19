package bookcheck

import (
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestFindingsSurviveARestartButNotAChangedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "q.json")
	at := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	store, _ := Open(path)
	store.Put("lib", "a", 10, at, []string{"no_cover"})
	store.Put("lib", "b", 10, at, nil)
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := again.Get("lib", "a", 10, at); !ok || len(got) != 1 || got[0] != "no_cover" {
		t.Fatalf("a = %v %v", got, ok)
	}
	if got, ok := again.Get("lib", "b", 10, at); !ok || len(got) != 0 {
		t.Fatalf("a clean book is a finding too: %v %v", got, ok)
	}
	if _, ok := again.Get("lib", "a", 11, at); ok {
		t.Fatal("a changed size makes the finding stale")
	}
	if _, ok := again.Get("lib", "a", 10, at.Add(time.Second)); ok {
		t.Fatal("a changed date makes the finding stale")
	}
	if _, ok := again.Get("lib", "never", 10, at); ok {
		t.Fatal("an unchecked book has no finding")
	}
}

// TestSaveHoldsTheLockForTheWholeWriteNotJustTheMarshal proves Save()'s disk write (temp-create, write,
// rename) is covered by the same lock as the in-memory marshal, not released early. See the identical
// test in internal/bookstats for the full reasoning -- this store had the exact same bug.
func TestSaveHoldsTheLockForTheWholeWriteNotJustTheMarshal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "q.json")
	at := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	store, _ := Open(path)
	store.Put("lib", "a", 10, at, []string{"no_cover"})

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
	<-entered

	secondDone := make(chan struct{})
	go func() {
		store.Put("lib", "b", 20, at, nil)
		store.Save()
		close(secondDone)
	}()

	select {
	case <-secondDone:
		t.Fatal("a second Put+Save completed while the first Save() still held the lock mid-write")
	case <-time.After(50 * time.Millisecond):
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
	if _, ok := again.Get("lib", "a", 10, at); !ok {
		t.Fatal("the first Save's data must survive on disk")
	}
	if _, ok := again.Get("lib", "b", 20, at); !ok {
		t.Fatal("the second Save's data must survive on disk, not be lost to an out-of-order write")
	}
}
