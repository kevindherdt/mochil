package audio

import (
	"reflect"
	"testing"
)

func TestVisiblePlaylistEditPreservesUnseenEntries(t *testing.T) {
	a := QueueEntry{LibraryID: "music", ItemID: "a"}
	b := QueueEntry{LibraryID: "music", ItemID: "b"}
	x := QueueEntry{LibraryID: "offline-library", ItemID: "x"}
	for _, test := range []struct {
		name              string
		replacement, want []QueueEntry
	}{
		{"reorder", []QueueEntry{b, a}, []QueueEntry{b, x, a}},
		{"remove", []QueueEntry{b}, []QueueEntry{b, x}},
		{"clear visible", []QueueEntry{}, []QueueEntry{x}},
		{"repeat track", []QueueEntry{a, a}, []QueueEntry{a, x, a}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := MergeVisiblePlaylistEdit([]QueueEntry{a, x, b}, []QueueEntry{a, b}, test.replacement)
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got=%v err=%v", got, err)
			}
		})
	}
	for _, test := range []struct{ visible, replacement []QueueEntry }{
		{[]QueueEntry{a, a}, nil}, {[]QueueEntry{b}, nil}, {[]QueueEntry{a}, []QueueEntry{x}},
	} {
		if _, err := MergeVisiblePlaylistEdit([]QueueEntry{a, x}, test.visible, test.replacement); err == nil {
			t.Fatal("invalid edit accepted")
		}
	}
	got, err := MergeVisiblePlaylistEdit([]QueueEntry{a, x, a}, []QueueEntry{a, a}, []QueueEntry{a})
	if err != nil || !reflect.DeepEqual(got, []QueueEntry{a, x}) {
		t.Fatalf("repeated visible entries: %v %v", got, err)
	}
	got, err = MergeVisiblePlaylistEdit([]QueueEntry{a, x, a}, []QueueEntry{a}, nil)
	if err != nil || !reflect.DeepEqual(got, []QueueEntry{x, a}) {
		t.Fatalf("unseen repeated entry: %v %v", got, err)
	}
}
