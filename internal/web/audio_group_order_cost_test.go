package web

import (
	"fmt"
	"math/rand"
	"mochil/internal/catalog"
	"reflect"
	"sort"
	"testing"
)

func previousAudioGroupSort(groups []androidAudioGroup) {
	sort.Slice(groups, func(i, j int) bool {
		if catalog.NaturalLess(groups[i].Name, groups[j].Name) {
			return true
		}
		if catalog.NaturalLess(groups[j].Name, groups[i].Name) {
			return false
		}
		return groups[i].ID < groups[j].ID
	})
}

func TestAudioGroupSortPreparedKeysPreserveOrder(t *testing.T) {
	names := []string{"The Album 2", "Album 02", "Album 10", "AN Album 2", "Été", "été", "123", "", "音楽", "  A Record 9  "}
	for seed := int64(0); seed < 30; seed++ {
		items := make([]androidAudioGroup, 100)
		for i := range items {
			items[i] = androidAudioGroup{ID: fmt.Sprint(i), Name: names[i%len(names)]}
		}
		rand.New(rand.NewSource(seed)).Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
		want := append([]androidAudioGroup(nil), items...)
		previousAudioGroupSort(want)
		sortAudioGroups(items)
		if !reflect.DeepEqual(items, want) {
			t.Fatalf("order differs for seed %d", seed)
		}
	}
}

func BenchmarkAudioGroupSort(b *testing.B) {
	source := make([]androidAudioGroup, 5000)
	for i := range source {
		source[i] = androidAudioGroup{ID: fmt.Sprint(i), Name: fmt.Sprintf("The Album %d", (i*7919)%5000)}
	}
	for name, sorter := range map[string]func([]androidAudioGroup){"previous": previousAudioGroupSort, "prepared": sortAudioGroups} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				items := append([]androidAudioGroup(nil), source...)
				sorter(items)
			}
		})
	}
}
