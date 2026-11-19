package audio

import (
	"errors"
	"os"
	"testing"
)

func TestPlaylistRetriesSkipWritesButPreserveCaseChanges(t *testing.T) {
	store, err := OpenPlaylistStore("")
	if err != nil {
		t.Fatal(err)
	}
	entries := []QueueEntry{{"music", "one"}, {"music", "two"}}
	if err := store.Put("alice", "My favorites", entries); err != nil {
		t.Fatal(err)
	}
	store.path = t.TempDir()
	if err := store.Put("alice", " My   favorites ", entries); err != nil {
		t.Fatal(err)
	}
	if err := store.Rename("alice", "my favorites", "My favorites"); err != nil {
		t.Fatal(err)
	}
	if err := store.Put("alice", "MY FAVORITES", entries); err == nil {
		t.Fatal("case-only title update must persist")
	}
	if err := store.Rename("alice", "My favorites", "MY FAVORITES"); err == nil {
		t.Fatal("case-only rename must persist")
	}
	if err := store.Put("alice", "My favorites", []QueueEntry{entries[1], entries[0]}); err == nil {
		t.Fatal("changed order must persist")
	}
	if err := store.Rename("alice", "Missing", "Missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing rename must not be acknowledged: %v", err)
	}
	if got := store.List("alice"); len(got) != 1 || got[0].Name != "My favorites" || got[0].Entries[0] != entries[0] {
		t.Fatalf("failed write changed memory: %#v", got)
	}
}

func TestEmptyPlaylistRetryDoesNotDiscardPlaylist(t *testing.T) {
	store, _ := OpenPlaylistStore("")
	if err := store.Put("alice", "Empty", nil); err != nil {
		t.Fatal(err)
	}
	store.path = t.TempDir()
	if err := store.Put("alice", "Empty", []QueueEntry{}); err != nil {
		t.Fatal(err)
	}
	if got := store.List("alice"); len(got) != 1 || got[0].Name != "Empty" {
		t.Fatalf("empty playlist lost: %#v", got)
	}
}
