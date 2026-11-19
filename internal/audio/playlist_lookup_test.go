package audio

import (
	"fmt"
	"testing"
)

func TestPlaylistGetIsAccountScopedAndDetached(t *testing.T) {
	s, err := OpenPlaylistStore("")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put("reader", "Road / trip", []QueueEntry{{LibraryID: "music", ItemID: "one"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Put("reader", "Empty", nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Road / trip", "ROAD / TRIP"} {
		got, ok := s.Get("reader", name)
		if !ok || got.Name != "Road / trip" || len(got.Entries) != 1 {
			t.Fatalf("lookup: %#v, %v", got, ok)
		}
		got.Entries[0].ItemID = "changed"
	}
	got, _ := s.Get("reader", "Road / trip")
	if got.Entries[0].ItemID != "one" {
		t.Fatal("lookup exposed mutable store entries")
	}
	if _, ok := s.Get("someone-else", "Road / trip"); ok {
		t.Fatal("cross-account lookup succeeded")
	}
	if _, ok := s.Get("reader", "Missing"); ok {
		t.Fatal("missing playlist found")
	}
	if got, ok := s.Get("reader", "Empty"); !ok || len(got.Entries) != 0 {
		t.Fatal("empty playlist confused with missing playlist")
	}
}

func BenchmarkPlaylistLookup(b *testing.B) {
	for _, count := range []int{10, 100, 1000} {
		lists := make([]Playlist, count)
		for i := range lists {
			lists[i] = Playlist{Name: fmt.Sprintf("Playlist %04d", i), Entries: make([]QueueEntry, 100)}
		}
		s := &PlaylistStore{data: playlistData{Version: 1, Accounts: map[string][]Playlist{"reader": lists}}}
		name := lists[count-1].Name
		b.Run(fmt.Sprintf("%d/Get", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, ok := s.Get("reader", name); !ok {
					b.Fatal("missing")
				}
			}
		})
		b.Run(fmt.Sprintf("%d/ListThenFind", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				for _, list := range s.List("reader") {
					if list.Name == name {
						break
					}
				}
			}
		})
	}
}
