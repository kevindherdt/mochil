package web

import (
	"fmt"
	"testing"

	"mochil/internal/audio"
)

// Includes identity construction, compilation merging, query filtering and
// sorting. Does not include store snapshot copying, HTTP serialization or I/O.
func BenchmarkAudioGroupProjection(b *testing.B) {
	items := make([]audio.Item, 10000)
	for i := range items {
		album, artist := i/10, i/100
		items[i] = audio.Item{
			ID: fmt.Sprint(i), Title: fmt.Sprintf("Track %d", i%10),
			Album:  fmt.Sprintf("The Album %d", album),
			Artist: fmt.Sprintf("Artist %d", artist),
			Path:   fmt.Sprintf("Albums/Edition %d/%d.flac", album, i%10),
		}
	}
	for _, mode := range []string{"albums", "artists"} {
		for _, query := range []string{"", "artist 9", "missing"} {
			b.Run(mode+"/"+query, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					groups := buildAudioGroups(items, "music", mode, query)
					if query == "missing" && len(groups) != 0 {
						b.Fatal("unexpected match")
					}
				}
			})
		}
	}
}
