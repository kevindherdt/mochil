package web

import (
	"sort"

	"mochil/internal/catalog"
)

// A map's iteration order must not reorder equal-named editions between requests.
func sortAudioGroups(groups []androidAudioGroup) {
	if len(groups) < 2 {
		return
	}
	keys := make([]string, len(groups))
	for i := range groups {
		keys[i] = catalog.SortKey(groups[i].Name)
	}
	sort.Sort(audioGroupSorter{groups, keys})
}

type audioGroupSorter struct {
	groups []androidAudioGroup
	keys   []string
}

func (s audioGroupSorter) Len() int { return len(s.groups) }
func (s audioGroupSorter) Swap(i, j int) {
	s.groups[i], s.groups[j] = s.groups[j], s.groups[i]
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}
func (s audioGroupSorter) Less(i, j int) bool {
	if catalog.NaturalKeyLess(s.keys[i], s.keys[j]) {
		return true
	}
	if catalog.NaturalKeyLess(s.keys[j], s.keys[i]) {
		return false
	}
	return s.groups[i].ID < s.groups[j].ID
}
