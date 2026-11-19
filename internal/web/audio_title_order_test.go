package web

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"

	"mochil/internal/audio"
)

func TestPreparedAudioTitlesPreserveAllSortModes(t *testing.T) {
	names := []string{"The Song 2", "Song 02", "SONG 2", "Song 10", "Été", "", "音楽", "An Album 9"}
	for _, kind := range []string{"music", "audiobooks", "podcasts"} {
		for _, mode := range []string{"", "title", "artist", "album", "series", "recent", "oldest", "newest"} {
			for seed := int64(0); seed < 10; seed++ {
				items := make([]audio.Item, 80)
				for i := range items {
					items[i] = audio.Item{ID: fmt.Sprint(i), Title: names[i%len(names)], Artist: names[(i+1)%len(names)],
						Album: names[(i+2)%len(names)], Track: i % 5, Disc: i % 2, Modified: time.Unix(int64(i%3), 0)}
				}
				rand.New(rand.NewSource(seed)).Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
				want := append([]audio.Item(nil), items...)
				sort.SliceStable(want, func(i, j int) bool { return audioItemLess(want[i], want[j], mode, kind) })
				sortAudioItems(items, mode, kind)
				if !reflect.DeepEqual(items, want) {
					t.Fatalf("order changed for %s/%s seed %d", kind, mode, seed)
				}
			}
		}
	}
}

func BenchmarkAudioTitleOrdering(b *testing.B) {
	source := make([]audio.Item, 5000)
	for i := range source {
		source[i] = audio.Item{ID: fmt.Sprint(i), Title: fmt.Sprintf("The Song %d", (i*7919)%len(source))}
	}
	for _, prepared := range []bool{false, true} {
		b.Run(fmt.Sprintf("prepared=%v", prepared), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				items := append([]audio.Item(nil), source...)
				if prepared {
					sortAudioItems(items, "title", "music")
				} else {
					sort.SliceStable(items, func(i, j int) bool { return audioItemLess(items[i], items[j], "title", "music") })
				}
			}
		})
	}
}
