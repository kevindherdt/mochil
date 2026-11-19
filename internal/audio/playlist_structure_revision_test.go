package audio

import (
	"errors"
	"sync"
	"testing"
)

func TestPlaylistStructureRejectsStaleRevision(t *testing.T) {
	for _, operation := range []string{"rename", "remove"} {
		t.Run(operation, func(t *testing.T) {
			store, _ := OpenPlaylistStore("")
			if err := store.Create("alice", "Saved"); err != nil {
				t.Fatal(err)
			}
			original, _ := store.Get("alice", "Saved")
			if err := store.Add("alice", "Saved", QueueEntry{LibraryID: "music", ItemID: "new"}); err != nil {
				t.Fatal(err)
			}
			mutate := func(account, revision string) error {
				if operation == "rename" {
					return store.RenameIfRevision(account, "Saved", "Renamed", revision)
				}
				return store.RemoveIfRevision(account, "Saved", revision)
			}
			if err := mutate("alice", PlaylistRevision(original)); !errors.Is(err, ErrPlaylistChanged) {
				t.Fatalf("stale mutation: %v", err)
			}
			current, found := store.Get("alice", "Saved")
			if !found || len(current.Entries) != 1 {
				t.Fatal("concurrent append lost")
			}
			if err := mutate("bob", PlaylistRevision(current)); !errors.Is(err, ErrPlaylistChanged) {
				t.Fatalf("foreign mutation: %v", err)
			}
			if err := mutate("alice", PlaylistRevision(current)); err != nil {
				t.Fatal(err)
			}
			if _, found := store.Get("alice", "Saved"); found {
				t.Fatal("original name retained")
			}
			if operation == "rename" {
				renamed, found := store.Get("alice", "Renamed")
				if !found || len(renamed.Entries) != 1 {
					t.Fatal("rename lost entries")
				}
			}
		})
	}
}

func TestConcurrentPlaylistRenameAndRemoveOnlyOneWins(t *testing.T) {
	store, _ := OpenPlaylistStore("")
	if err := store.Create("alice", "Saved"); err != nil {
		t.Fatal(err)
	}
	list, _ := store.Get("alice", "Saved")
	revision := PlaylistRevision(list)
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, mutate := range []func() error{
		func() error { return store.RenameIfRevision("alice", "Saved", "Renamed", revision) },
		func() error { return store.RemoveIfRevision("alice", "Saved", revision) },
	} {
		group.Add(1)
		go func(mutate func() error) { defer group.Done(); <-start; results <- mutate() }(mutate)
	}
	close(start)
	group.Wait()
	close(results)
	winners, conflicts := 0, 0
	for err := range results {
		if err == nil {
			winners++
		} else if errors.Is(err, ErrPlaylistChanged) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("winners=%d conflicts=%d", winners, conflicts)
	}
}
