package web

import (
	"net/http/httptest"
	"testing"

	"mochil/internal/audio"
)

func TestAndroidAudioGroupIdentityUsesAlbumArtistFallbacks(t *testing.T) {
	item := audio.Item{Album: "After Laughter", Artist: "Paramore", AlbumArtist: "Paramore"}
	if name, subtitle := androidAudioGroupIdentity(item, "albums"); name != "After Laughter" || subtitle != "Paramore" {
		t.Fatalf("album identity=%q %q", name, subtitle)
	}
	item.AlbumArtist = ""
	if name, subtitle := androidAudioGroupIdentity(item, "albums"); name != "After Laughter" || subtitle != "Paramore" {
		t.Fatalf("album artist fallback=%q %q", name, subtitle)
	}
	if name, _ := androidAudioGroupIdentity(item, "artists"); name != "Paramore" {
		t.Fatalf("artist fallback=%q", name)
	}
}

func TestAudioGroupIDIsNormalizedAndAlbumFoldersDefineEditions(t *testing.T) {
	first := audioGroupID("Music", "albums", " Shared Album ", "Artist A")
	if first != audioGroupID("music", "ALBUMS", "shared album", " artist a ") || len(first) != 32 {
		t.Fatalf("unstable group id %q", first)
	}
	artistA := audio.Item{Album: "Shared Album", Artist: "Artist A", Path: "Albums/Shared/01.flac"}
	artistB := audio.Item{Album: "Shared Album", Artist: "Artist B", Path: "Albums/Shared/02.flac"}
	otherEdition := audio.Item{Album: "Shared Album", Artist: "Artist A", Path: "Archive/Shared/01.flac"}
	if audioGroupItemID("music", "albums", artistA) != audioGroupItemID("music", "albums", artistB) {
		t.Fatal("multi-artist tracks in one album folder were split")
	}
	if audioGroupItemID("music", "albums", artistA) == audioGroupItemID("music", "albums", otherEdition) {
		t.Fatal("same-titled albums in different folders shared an identity")
	}
	if audioGroupItemID("music", "albums", audio.Item{Album: "Set", Path: "Albums/Set/CD 1/01.flac"}) != audioGroupItemID("music", "albums", audio.Item{Album: "Set", Path: "Albums/Set/Disc_2/01.flac"}) {
		t.Fatal("disc subfolders split one album")
	}
}

func TestAndroidAudioGroupIdentitySupportsAudiobooks(t *testing.T) {
	item := audio.Item{Title: "Volume 2", Artist: "An Author", Album: "Fallback Saga", Series: "Main Saga"}
	if name, subtitle := androidAudioGroupIdentity(item, "series"); name != "Main Saga" || subtitle != "An Author" {
		t.Fatalf("series identity=%q %q", name, subtitle)
	}
	if name, subtitle := androidAudioGroupIdentity(item, "authors"); name != "An Author" || subtitle != "Author" {
		t.Fatalf("author identity=%q %q", name, subtitle)
	}
	item.Series = ""
	if name, _ := androidAudioGroupIdentity(item, "series"); name != "Fallback Saga" {
		t.Fatalf("series fallback=%q", name)
	}
	if !androidAudioGroupingAllowed("audiobooks", "series") || androidAudioGroupingAllowed("music", "series") {
		t.Fatal("audiobook grouping leaked into music")
	}
}

func TestAndroidAudioPageBoundsInvalidInput(t *testing.T) {
	request := httptest.NewRequest("GET", "/?offset=999&limit=999", nil)
	offset, limit := boundedAndroidAudioPage(request, 12)
	if offset != 12 || limit != 60 {
		t.Fatalf("page=%d,%d", offset, limit)
	}
}

func TestAndroidArtistAlbumGroupsAreScopedAndNaturallySorted(t *testing.T) {
	items := []audio.Item{
		{ID: "p2", Album: "Album 10", Artist: "Paramore"},
		{ID: "other", Album: "Album 1", Artist: "Other"},
		{ID: "p1", Album: "Album 2", Artist: "Paramore"},
		{ID: "p1b", Album: "Album 2", Artist: "Paramore"},
	}
	groups := androidArtistAlbumGroups(items, "music", "paramore")
	if len(groups) != 2 || groups[0].Name != "Album 2" || groups[0].Count != 2 || groups[1].Name != "Album 10" {
		t.Fatalf("groups=%#v", groups)
	}
	if groups[0].ArtworkLibraryID != "music" || groups[0].Subtitle != "Paramore" || groups[0].ID == "" {
		t.Fatalf("group metadata=%#v", groups[0])
	}
}

func TestArtistAlbumEditionOrderMatchesRootGroups(t *testing.T) {
	items := []audio.Item{
		{ID: "one", Album: "Album 2", Artist: "Artist", Path: "First/one.flac"},
		{ID: "two", Album: "Album 2", Artist: "Artist", Path: "Second/two.flac"},
		{ID: "three", Album: "Album 10", Artist: "Artist", Path: "Third/three.flac"},
	}
	want := buildAudioGroups(items, "music", "albums", "")
	for attempt := 0; attempt < 50; attempt++ {
		items[0], items[1] = items[1], items[0]
		got := androidArtistAlbumGroups(items, "music", "artist")
		if len(got) != len(want) {
			t.Fatal("editions merged")
		}
		for i := range got {
			if got[i].ID != want[i].ID {
				t.Fatalf("edition order changed: %+v", got)
			}
		}
	}
}
