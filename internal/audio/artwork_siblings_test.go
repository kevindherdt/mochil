package audio

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func previousArtworkSiblings(s *Store, id string, limit int) []Item {
	item, ok := s.Item(id)
	if !ok || limit <= 0 {
		return nil
	}
	directory := filepath.Dir(filepath.FromSlash(item.Path))
	known := make([]Item, 0, limit)
	unknown := make([]Item, 0, limit)
	for _, candidate := range s.Items() {
		if candidate.ID == id || filepath.Dir(filepath.FromSlash(candidate.Path)) != directory {
			continue
		}
		if candidate.HasArtwork {
			known = append(known, candidate)
		} else if len(unknown) < limit {
			unknown = append(unknown, candidate)
		}
	}
	if len(known) >= limit {
		return known[:limit]
	}
	if len(unknown) > limit-len(known) {
		unknown = unknown[:limit-len(known)]
	}
	return append(known, unknown...)
}

func TestArtworkSiblingsRemainBoundedOrderedAndDetached(t *testing.T) {
	s := &Store{}
	var items []Item
	for i := 0; i < 100; i++ {
		items = append(items, Item{ID: fmt.Sprint(i), Path: fmt.Sprintf("Album/%03d.flac", i), HasArtwork: i%3 == 0, Chapters: []Chapter{{Title: "Original"}}})
	}
	items = append(items, Item{ID: "elsewhere", Path: "Other/01.flac", HasArtwork: true})
	s.publish(items)
	for _, limit := range []int{-1, 0, 1, 4, 30, 200} {
		got, want := s.ArtworkSiblings("1", limit), previousArtworkSiblings(s, "1", limit)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("limit %d: different candidates", limit)
		}
	}
	got := s.ArtworkSiblings("1", 4)
	got[0].Chapters[0].Title = "Changed"
	stored, _ := s.Item(got[0].ID)
	if stored.Chapters[0].Title != "Original" {
		t.Fatal("result aliases stored chapters")
	}
	if s.ArtworkSiblings("missing", 4) != nil {
		t.Fatal("unknown track returned siblings")
	}
}

func TestArtworkSiblingsUseOneSnapshotDuringScanPublication(t *testing.T) {
	s := &Store{}
	snapshot := func(revision string) []Item {
		items := make([]Item, 20)
		for i := range items {
			items[i] = Item{ID: fmt.Sprint(i), Path: fmt.Sprintf("Album-%s/%02d.flac", revision, i), Revision: revision, HasArtwork: true}
		}
		return items
	}
	one, two := snapshot("one"), snapshot("two")
	s.publish(one)
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		for i := 0; i < 200; i++ {
			s.publish(one)
			s.publish(two)
		}
	}()
	defer workers.Wait()
	for i := 0; i < 400; i++ {
		items := s.ArtworkSiblings("0", 4)
		if len(items) != 4 {
			t.Fatalf("mixed snapshot omitted siblings: %d", len(items))
		}
		for _, item := range items {
			if item.ID == "0" || item.Revision != items[0].Revision {
				t.Fatal("inconsistent snapshot")
			}
		}
	}
}

func BenchmarkArtworkSiblingLookup(b *testing.B) {
	s := &Store{}
	items := make([]Item, 10000)
	for i := range items {
		items[i] = Item{ID: fmt.Sprint(i), Path: fmt.Sprintf("Album-%04d/%03d.flac", i/20, i%20), HasArtwork: i%4 == 0}
	}
	s.publish(items)
	for _, id := range []string{"1", "9981"} {
		for _, name := range []string{"previous", "bounded"} {
			b.Run(id+"/"+name, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var result []Item
					if name == "previous" {
						result = previousArtworkSiblings(s, id, 4)
					} else {
						result = s.ArtworkSiblings(id, 4)
					}
					if len(result) != 4 {
						b.Fatal("wrong result count")
					}
				}
			})
		}
	}
}
