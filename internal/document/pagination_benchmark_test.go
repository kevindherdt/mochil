package document

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkDocumentPagination100K(b *testing.B) {
	items := make([]Item, 100000)
	for index := range items {
		items[index] = Item{ID: fmt.Sprintf("%06d", index), Title: fmt.Sprintf("Document %06d", index), Path: fmt.Sprintf("%06d.txt", index), Modified: time.Unix(int64(index), 0)}
	}
	store := NewStore("notes", "", "")
	store.publish(items)
	b.Run("keyset-page", func(b *testing.B) {
		for range b.N {
			page, _, _, _, _ := store.Page("title", "050000", 0, 120)
			if len(page) != 120 {
				b.Fatal(len(page))
			}
		}
	})
	b.Run("copy-full-snapshot", func(b *testing.B) {
		for range b.N {
			all := store.Items()
			page := all[50001:50121]
			if len(page) != 120 {
				b.Fatal(len(page))
			}
		}
	})
}
