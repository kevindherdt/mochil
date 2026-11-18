package audio

import (
	"slices"
	"testing"
)

func TestQueuePutValidatesBeforeChangingState(t *testing.T) {
	store, err := OpenQueueStore("")
	if err != nil {
		t.Fatal(err)
	}
	original := []QueueEntry{{LibraryID: "music", ItemID: "one"}}
	if err := store.Put("reader", original); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, account string
		entries       []QueueEntry
	}{
		{"empty account", "", original},
		{"empty library", "reader", []QueueEntry{{ItemID: "one"}}},
		{"empty item", "reader", []QueueEntry{{LibraryID: "music"}}},
		{"over capacity", "reader", make([]QueueEntry, 10001)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if store.Put(tc.account, tc.entries) == nil {
				t.Fatal("invalid queue accepted")
			}
			if !slices.Equal(store.Get("reader"), original) || len(store.Get("")) != 0 {
				t.Fatal("invalid request changed saved state")
			}
		})
	}
}
