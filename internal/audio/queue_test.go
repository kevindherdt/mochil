package audio

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func TestQueueStoreConcurrentAddsDoNotLoseEntries(t *testing.T) {
	store, err := OpenQueueStore(filepath.Join(t.TempDir(), "queue.json"))
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			if err := store.Add("reader", QueueEntry{LibraryID: "music", ItemID: fmt.Sprintf("%03d", index)}); err != nil {
				t.Errorf("add: %v", err)
			}
		}(index)
	}
	wait.Wait()
	if got := len(store.Get("reader")); got != 100 {
		t.Fatalf("entries = %d", got)
	}
}

func TestQueueStoreFailedPersistenceDoesNotChangeMemory(t *testing.T) {
	store := &QueueStore{path: "/dev/null/queue.json", data: queueData{Version: 1, Accounts: map[string][]QueueEntry{"reader": {{LibraryID: "music", ItemID: "one"}}}}}
	if err := store.Put("reader", []QueueEntry{{LibraryID: "music", ItemID: "two"}}); err == nil {
		t.Fatal("expected persistence error")
	}
	entries := store.Get("reader")
	if len(entries) != 1 || entries[0].ItemID != "one" {
		t.Fatalf("entries changed: %#v", entries)
	}
}
