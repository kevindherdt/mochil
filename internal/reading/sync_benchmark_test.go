package reading

import (
	"fmt"
	"runtime"
	"testing"
)

// The comparison adds back the whole-store snapshot formerly taken before
// revision checks. Neither case writes to disk: both acknowledge stale input.
func BenchmarkEquivalentCheckpointLargeHistory(b *testing.B) {
	for _, entries := range []int{100, 5000} {
		b.Run(fmt.Sprintf("items_%d", entries), func(b *testing.B) {
			for _, priorSnapshot := range []bool{false, true} {
				name := "current"
				if priorSnapshot {
					name = "with_previous_snapshot_cost"
				}
				b.Run(name, func(b *testing.B) {
					store, _ := Open("")
					positions := make(map[string]Position, entries)
					histories := make(map[string][]Position, entries)
					for index := 0; index < entries; index++ {
						id := fmt.Sprintf("book-%d", index)
						position := Position{LibraryID: "books", BookID: id, Revision: 1, Fraction: .5,
							Locators:  map[string]string{"readium": "chapter-location"},
							WebAnchor: &Anchor{SpineIndex: 1, Exact: "A passage of text.", Prefix: "Previous sentence."}}
						key := readingKey("books", id)
						positions[key] = position
						for history := 0; history < 5; history++ {
							histories[key] = append(histories[key], clonePosition(position))
						}
					}
					store.data.Accounts["reader"] = positions
					store.data.History["reader"] = histories
					checkpoint := Position{LibraryID: "books", BookID: "book-0", Fraction: .5}
					b.ReportAllocs()
					b.ResetTimer()
					for index := 0; index < b.N; index++ {
						if priorSnapshot {
							store.mu.Lock()
							snapshot := cloneStoreData(store.data)
							store.mu.Unlock()
							runtime.KeepAlive(snapshot)
						}
						if _, err := store.Put("reader", checkpoint, 0); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
