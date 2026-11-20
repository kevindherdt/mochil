package audio

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentProgressHasOneDurableWinner(t *testing.T) {
	file := filepath.Join(t.TempDir(), "progress.json")
	store, err := OpenProgressStore(file)
	if err != nil {
		t.Fatal(err)
	}
	initial := Progress{LibraryID: "podcasts", ItemID: "episode", Position: 10, Duration: 1000, DeviceID: "initial"}
	if err := store.Put("reader", initial); err != nil {
		t.Fatal(err)
	}
	initial, _ = store.Get("reader", "podcasts", "episode")
	const devices = 12
	start := make(chan struct{})
	results := make(chan error, devices)
	var workers sync.WaitGroup
	for i := 0; i < devices; i++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			next := initial
			next.Position = 100 + float64(index)*10
			next.DeviceID = fmt.Sprintf("device-%d", index)
			results <- store.PutIfRevision("reader", next, initial.Revision)
		}(i)
	}
	close(start)
	workers.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrProgressConflict) {
			t.Fatal(err)
		}
	}
	if winners != 1 {
		t.Fatalf("got %d accepted writers", winners)
	}
	winner, _ := store.Get("reader", "podcasts", "episode")
	if winner.Revision != initial.Revision+1 {
		t.Fatalf("unexpected revision: %+v", winner)
	}
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	version := store.Version()
	for i := 0; i < devices; i++ {
		stale := winner
		stale.Position -= 0.1
		stale.DeviceID = fmt.Sprintf("passive-%d", i)
		stale.UpdatedAt = 1
		if err := store.PutIfRevision("reader", stale, initial.Revision); err != nil {
			t.Fatal(err)
		}
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || store.Version() != version {
		t.Fatal("equivalent stale acknowledgements rewrote state")
	}
	reopened, err := OpenProgressStore(file)
	if err != nil {
		t.Fatal(err)
	}
	durable, ok := reopened.Get("reader", "podcasts", "episode")
	if !ok || durable != winner {
		t.Fatalf("durable winner changed: %+v want %+v", durable, winner)
	}
}
