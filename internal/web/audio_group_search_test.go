package web

import (
	"reflect"
	"testing"

	"mochil/internal/audio"
)

func TestAudioGroupSearchPreservesCompleteCompilation(t *testing.T) {
	items := []audio.Item{
		{ID: "first", Album: "Compilation", Artist: "Alice", Path: "Compilation/01.flac"},
		{ID: "second", Album: "Compilation", Artist: "Bob", Path: "Compilation/02.flac"},
		{ID: "third", Album: "Other", Artist: "Carol", Path: "Other/01.flac"},
	}
	want := buildAudioGroups(items, "music", "albums", "compilation")
	if len(want) != 1 || want[0].Count != 2 || want[0].Subtitle != "Various artists" || want[0].ArtworkItemID != "first" {
		t.Fatalf("unexpected compilation: %+v", want)
	}
	for _, query := range []string{"alice", "bob", "various artists"} {
		t.Run(query, func(t *testing.T) {
			got := buildAudioGroups(items, "music", "albums", query)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("search changed album identity, count, subtitle or artwork: got=%+v want=%+v", got, want)
			}
		})
	}
	if got := buildAudioGroups(items, "music", "albums", "missing"); len(got) != 0 {
		t.Fatalf("unrelated albums returned: %+v", got)
	}
}
