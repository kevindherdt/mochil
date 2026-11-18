package audio

import (
	"errors"
	"sync"
	"testing"
)

func TestOnlyOneConcurrentPlaylistReplacementWins(t *testing.T) {
	store, _ := OpenPlaylistStore("")
	if err := store.Create("alice", "Saved"); err != nil {
		t.Fatal(err)
	}
	original, _ := store.Get("alice", "Saved")
	revision := PlaylistRevision(original)
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, id := range []string{"one", "two"} {
		group.Add(1)
		go func(id string) {
			defer group.Done()
			<-start
			results <- store.PutIfRevision("alice", "Saved", []QueueEntry{{LibraryID: "music", ItemID: id}}, revision)
		}(id)
	}
	close(start)
	group.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrPlaylistChanged) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
	current, _ := store.Get("alice", "Saved")
	if len(current.Entries) != 1 {
		t.Fatal("invalid final playlist")
	}
}

func TestConditionalPlaylistPutPreservesNewerEntries(t *testing.T) {
	store, _ := OpenPlaylistStore("")
	old := []QueueEntry{{LibraryID: "music", ItemID: "one"}}
	if err := store.Put("alice", "Saved", old); err != nil {
		t.Fatal(err)
	}
	original, _ := store.Get("alice", "Saved")
	revision := PlaylistRevision(original)
	newer := append(append([]QueueEntry(nil), old...), QueueEntry{LibraryID: "music", ItemID: "two"})
	if err := store.PutIfRevision("alice", "Saved", newer, revision); err != nil {
		t.Fatal(err)
	}
	if err := store.PutIfRevision("alice", "Saved", nil, revision); !errors.Is(err, ErrPlaylistChanged) {
		t.Fatalf("stale write: %v", err)
	}
	current, _ := store.Get("alice", "Saved")
	if len(current.Entries) != 2 {
		t.Fatal("new entries lost")
	}
	if err := store.PutIfRevision("bob", "Saved", nil, PlaylistRevision(current)); !errors.Is(err, ErrPlaylistChanged) {
		t.Fatal("foreign/missing playlist replaced")
	}
	if PlaylistRevision(Playlist{Name: "Empty"}) != PlaylistRevision(Playlist{Name: "Empty", Entries: []QueueEntry{}}) {
		t.Fatal("nil/empty revision mismatch")
	}
}
