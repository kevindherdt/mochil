package audio

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// Forced persistence is a controlled comparison, not an old checkout: it uses
// today's writer and explicitly bypasses the unchanged-queue fast path.
func BenchmarkQueueUnchangedSave(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		entries := make([]QueueEntry, size)
		for i := range entries {
			entries[i] = QueueEntry{"music", fmt.Sprintf("track-%06d", i)}
		}
		for _, force := range []bool{false, true} {
			mode := "unchanged"
			if force {
				mode = "forced-persistence"
			}
			b.Run(fmt.Sprintf("%d/%s", size, mode), func(b *testing.B) {
				store, err := OpenQueueStore(filepath.Join(b.TempDir(), "queue.json"))
				if err != nil {
					b.Fatal(err)
				}
				if err := store.Put("reader", entries); err != nil {
					b.Fatal(err)
				}
				// Model separately decoded request strings, not pointer-equal IDs.
				request := make([]QueueEntry, len(entries))
				for i, entry := range entries {
					request[i] = QueueEntry{strings.Clone(entry.LibraryID), strings.Clone(entry.ItemID)}
				}
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if force {
						store.mu.Lock()
						err = store.save(store.data)
						store.mu.Unlock()
					} else {
						err = store.Put("reader", request)
					}
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
