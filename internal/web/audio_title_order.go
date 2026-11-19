package web

import (
	"sort"

	"mochil/internal/audio"
	"mochil/internal/catalog"
)

// Prepare title keys only for title-ordered views. Date/artist/series views keep
// their own priorities without paying for keys they may never compare.
func sortAudioItems(items []audio.Item, mode, kind string) {
	if len(items) < 2 {
		return
	}
	if mode != "title" && !(mode == "" && kind != "podcasts") {
		sort.SliceStable(items, func(i, j int) bool { return audioItemLess(items[i], items[j], mode, kind) })
		return
	}
	keys := make([]string, len(items))
	for i := range items {
		keys[i] = catalog.SortKey(items[i].Title)
	}
	sort.Stable(audioTitleSorter{items: items, keys: keys})
}

type audioTitleSorter struct {
	items []audio.Item
	keys  []string
}

func (s audioTitleSorter) Len() int { return len(s.items) }
func (s audioTitleSorter) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}
func (s audioTitleSorter) Less(i, j int) bool {
	if catalog.NaturalKeyLess(s.keys[i], s.keys[j]) {
		return true
	}
	if catalog.NaturalKeyLess(s.keys[j], s.keys[i]) {
		return false
	}
	return s.items[i].ID < s.items[j].ID
}
