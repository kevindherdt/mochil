package filefacts

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

type found struct {
	Own string `json:"own"`
}

func TestFactsSurviveARestartButNotAChangedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.json")
	at := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	store, _ := Open[found](path)
	store.Put("lib", "a", 10, at, found{Own: "9780306406157"})
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := Open[found](path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := again.Get("lib", "a", 10, at); !ok || got.Own != "9780306406157" {
		t.Fatalf("%v %v", got, ok)
	}
	if _, ok := again.Get("lib", "a", 11, at); ok {
		t.Fatal("a changed size makes it stale")
	}
	if _, ok := again.Get("lib", "a", 10, at.Add(time.Second)); ok {
		t.Fatal("a changed date makes it stale")
	}
	if _, ok := again.Get("lib", "other", 10, at); ok {
		t.Fatal("an unseen book has no fact")
	}
}

func TestACorruptFileIsReportedNotSilentlyDropped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.json")
	if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open[found](path); err == nil {
		t.Fatal("a corrupt store must be an error")
	}
}

// TestSaveHoldsTheLockForTheWholeWriteNotJustTheMarshal proves Save()'s disk write (temp-create, write,
// rename) is covered by the same lock as the in-memory marshal, not released early. See the identical
// test in internal/bookstats for the full reasoning -- this store had the exact same bug.
func TestSaveHoldsTheLockForTheWholeWriteNotJustTheMarshal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.json")
	at := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	store, _ := Open[found](path)
	store.Put("lib", "a", 10, at, found{Own: "9780306406157"})

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
		store.Put("lib", "b", 20, at, found{Own: "other"})
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

	again, err := Open[found](path)
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

func TestRemoveForgetsOneBookWithoutTouchingTheRest(t *testing.T) {
	store, err := Open[int](filepath.Join(t.TempDir(), "facts.json"))
	if err != nil {
		t.Fatal(err)
	}
	modified := time.Unix(1700000000, 0)
	store.Put("lib", "one", 10, modified, 1)
	store.Put("lib", "two", 20, modified, 2)
	if !store.Remove("lib", "one") {
		t.Fatal("Remove must report the entry it dropped")
	}
	if _, ok := store.Get("lib", "one", 10, modified); ok {
		t.Fatal("a removed entry must not be returned")
	}
	if value, ok := store.Get("lib", "two", 20, modified); !ok || value != 2 {
		t.Fatalf("other entries must survive: %v %v", value, ok)
	}
	if store.Remove("lib", "missing") {
		t.Fatal("removing an unknown book must report false")
	}
}
