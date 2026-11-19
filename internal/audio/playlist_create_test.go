package audio

import (
	"errors"
	"sync"
	"testing"
)

func TestPlaylistCreateIsAtomicAndAccountScoped(t *testing.T) {
	store, _ := OpenPlaylistStore("")
	var group sync.WaitGroup
	results := make(chan error, 16)
	for i := 0; i < 16; i++ {
		group.Add(1)
		go func() { defer group.Done(); results <- store.Create("alice", "Evening") }()
	}
	group.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrPlaylistExists) {
			t.Fatal(err)
		}
	}
	if successes != 1 || len(store.List("alice")) != 1 {
		t.Fatalf("successful creates: %d", successes)
	}
	if err := store.Create("bob", "Evening"); err != nil {
		t.Fatal(err)
	}
}

func TestPlaylistCreateFailedPersistencePreservesState(t *testing.T) {
	store, _ := OpenPlaylistStore("")
	store.path = t.TempDir()
	if err := store.Create("alice", "Evening"); err == nil {
		t.Fatal("expected write failure")
	}
	if len(store.List("alice")) != 0 {
		t.Fatal("failed create published state")
	}
}
