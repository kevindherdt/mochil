package web

import (
	"math/rand"
	"sort"
	"testing"
	"time"

	"mochil/internal/audio"
)

func TestAudioItemLessOrdersAudiobookSeriesAndVolumesNaturally(t *testing.T) {
	items := []audio.Item{
		{ID: "10", Series: "The Expanse", Track: 10, Title: "Book 10"},
		{ID: "other", Series: "Witcher", Track: 1, Title: "Book 1"},
		{ID: "2", Series: "The Expanse", Track: 2, Title: "Book 2"},
		{ID: "2.5", Series: "The Expanse", SeriesIndex: 2.5, Title: "Novella"},
		{ID: "1", Album: "The Expanse", Track: 1, Title: "Book 1"},
		{ID: "missing", Series: "The Expanse", Title: "Unknown position"},
	}
	sort.SliceStable(items, func(i, j int) bool {
		return audioItemLess(items[i], items[j], "series", "audiobooks")
	})
	want := []string{"1", "2", "2.5", "10", "missing", "other"}
	for index := range want {
		if items[index].ID != want[index] {
			t.Fatalf("position %d = %q, want %q", index, items[index].ID, want[index])
		}
	}
}

func TestPodcastOrderingSupportsOldestNewestAndStableTitleTies(t *testing.T) {
	stamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	items := []audio.Item{
		{ID: "chapter-10", Title: "Chapter 10", Modified: stamp},
		{ID: "older", Title: "Prologue", Modified: stamp.Add(-time.Hour)},
		{ID: "chapter-2", Title: "Chapter 2", Modified: stamp},
	}
	oldest := append([]audio.Item(nil), items...)
	sort.SliceStable(oldest, func(i, j int) bool { return audioItemLess(oldest[i], oldest[j], "oldest", "podcasts") })
	if got := []string{oldest[0].ID, oldest[1].ID, oldest[2].ID}; !equalStrings(got, []string{"older", "chapter-2", "chapter-10"}) {
		t.Fatalf("oldest order = %v", got)
	}
	newest := append([]audio.Item(nil), items...)
	sort.SliceStable(newest, func(i, j int) bool { return audioItemLess(newest[i], newest[j], "newest", "podcasts") })
	if got := []string{newest[0].ID, newest[1].ID, newest[2].ID}; !equalStrings(got, []string{"chapter-2", "chapter-10", "older"}) {
		t.Fatalf("newest order = %v", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func TestAudioSortNormalizedTiesHaveDeterministicIdentityOrder(t *testing.T) {
	for _, mode := range []string{"title", "recent", "oldest", "album", "series", "artist", ""} {
		t.Run(mode, func(t *testing.T) {
			items := []audio.Item{{ID: "a", Title: "Song"}, {ID: "b", Title: "song"}, {ID: "c", Title: "SONG"}}
			for seed := int64(0); seed < 20; seed++ {
				values := append([]audio.Item(nil), items...)
				rand.New(rand.NewSource(seed)).Shuffle(len(values), func(i, j int) { values[i], values[j] = values[j], values[i] })
				sort.SliceStable(values, func(i, j int) bool { return audioItemLess(values[i], values[j], mode, "music") })
				if got := []string{values[0].ID, values[1].ID, values[2].ID}; !equalStrings(got, []string{"a", "b", "c"}) {
					t.Fatalf("seed %d: normalized titles sorted as %v", seed, got)
				}
			}
		})
	}
}

func TestAudioSeriesArticleAndNumericVariantsStillUseVolumeOrder(t *testing.T) {
	first := audio.Item{ID: "z", Title: "Part 1", Series: "The Saga 02", SeriesIndex: 1}
	second := audio.Item{ID: "a", Title: "Part 2", Series: "Saga 2", SeriesIndex: 2}
	if !audioItemLess(first, second, "series", "audiobooks") || audioItemLess(second, first, "series", "audiobooks") {
		t.Fatal("equivalent series names must fall through to volume ordering")
	}
}
