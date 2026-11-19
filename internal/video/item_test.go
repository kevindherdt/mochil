package video

import (
	"testing"
	"time"
)

func TestIdentifyDirectAndIndexedVideoFormats(t *testing.T) {
	for _, test := range []struct {
		path, mediaType string
		direct          bool
	}{
		{"Movies/My_Film.2026.MP4", "video/mp4", true},
		{"Movies/My Film.webm", "video/webm", true},
		{"Movies/My Film.mkv", "video/x-matroska", false},
	} {
		item, ok := Identify("movies", test.path, 1234, time.Unix(100, 0))
		if !ok || item.MediaType != test.mediaType || item.DirectPlay != test.direct {
			t.Fatalf("Identify(%q) = %#v, %v", test.path, item, ok)
		}
	}
}

func TestIdentifyRejectsPublicationAndEscapingPaths(t *testing.T) {
	for _, path := range []string{"Book.epub", "../Movies/Film.mp4", "/Movies/Film.mp4"} {
		if _, ok := Identify("movies", path, 1, time.Time{}); ok {
			t.Fatalf("Identify(%q) accepted an invalid video path", path)
		}
	}
	if _, ok := Identify("movies", "Film.mp4", -1, time.Time{}); ok {
		t.Fatal("Identify accepted a negative remote object size")
	}
}

func TestIdentityIsStableAndRevisionTracksReplacement(t *testing.T) {
	first, _ := Identify("movies", "Films/Example.mp4", 100, time.Unix(10, 0))
	second, _ := Identify("movies", "Films/Example.mp4", 200, time.Unix(20, 0))
	if first.ID != second.ID || first.Revision == second.Revision {
		t.Fatalf("identity/revision contract failed: %#v %#v", first, second)
	}
	otherShelf, _ := Identify("archive", "Films/Example.mp4", 100, time.Unix(10, 0))
	if first.ID == otherShelf.ID {
		t.Fatal("different libraries shared an identity")
	}
}

func TestIdentifyCreatesReadableTitle(t *testing.T) {
	item, _ := Identify("movies", "Films/My_Film.2026.mp4", 1, time.Time{})
	if item.Title != "My Film 2026" {
		t.Fatalf("title = %q", item.Title)
	}
}
