package audio

import "testing"

func TestInspectQualityFindsDeterministicMusicProblems(t *testing.T) {
	items := []Item{
		{ID: "1", Path: "Artist/Album/01.flac", Title: "First", Artist: "Unknown Artist", AlbumArtist: "Artist", Album: "Album", Genre: "Rock", Track: 1, Size: 100},
		{ID: "2", Path: "Artist/Album/01-copy.flac", Title: "First", Artist: "Artist", AlbumArtist: "Artist", Album: "Album", Genre: "Pop", Track: 1, Size: 100},
		{ID: "3", Path: "Artist/Album/02.flac", Title: "Second", Artist: "Artist", AlbumArtist: "Different", Album: "Album", Genre: "Rock", Track: 2},
	}
	report := InspectQuality(items, func(Item) bool { return false })
	if report.Tracks != 3 || report.Albums != 1 || report.Counts["unknown_artist"] != 1 || report.Counts["duplicate_file_candidate"] != 1 || report.Counts["inconsistent_album_tags"] != 1 || report.Counts["missing_artwork"] != 1 {
		t.Fatalf("report=%+v", report)
	}
}

func TestInspectQualityUsesKnownEmbeddedArtworkWithoutSidecarLookup(t *testing.T) {
	calls := 0
	items := []Item{
		{Path: "Artist/Album/01.flac", Title: "First", Artist: "Artist", Album: "Album", HasArtwork: true},
		{Path: "Artist/Album/02.flac", Title: "Second", Artist: "Artist", Album: "Album"},
	}
	report := InspectQuality(items, func(Item) bool { calls++; return false })
	if calls != 0 || report.Counts["missing_artwork"] != 0 {
		t.Fatalf("artwork checks=%d report=%+v", calls, report)
	}
}

func TestInspectQualityDoesNotCallCrossFolderTagCollisionsDuplicateFiles(t *testing.T) {
	items := []Item{
		{Path: "Tracks/Shin Yong Jae - Feel You.flac", Title: "Feel You", Artist: "Shin Yong Jae", AlbumArtist: "Various Artists", Album: "Flower of Evil OST", Disc: 1, Track: 3, Size: 100},
		{Path: "Albums/Flower of Evil OST/03 Feel You.flac", Title: "Feel You", Artist: "Shin Yong Jae", AlbumArtist: "Various Artists", Album: "Flower of Evil OST", Disc: 1, Track: 3, Size: 100},
	}
	report := InspectQuality(items, nil)
	if report.Counts["duplicate_file_candidate"] != 0 || report.Counts["track_identity_conflict"] != 0 {
		t.Fatalf("report=%+v", report)
	}
}

func TestInspectQualityTreatsDiscFoldersAsOnePhysicalAlbum(t *testing.T) {
	items := []Item{
		{Path: "Artist/Album/CD 1/01.flac", Title: "First", Artist: "Artist", Album: "Album"},
		{Path: "Artist/Album/Disc_2/01.flac", Title: "Second", Artist: "Artist", Album: "Album"},
	}
	report := InspectQuality(items, func(Item) bool { return true })
	if report.Albums != 1 {
		t.Fatalf("albums=%d report=%+v", report.Albums, report)
	}
}

func TestInspectQualityDoesNotCallArtworkForEveryTrack(t *testing.T) {
	calls := 0
	items := []Item{
		{Path: "Artist/Album/01.flac", Title: "First", Artist: "Artist", Album: "Album"},
		{Path: "Artist/Album/02.flac", Title: "Second", Artist: "Artist", Album: "Album"},
	}
	InspectQuality(items, func(Item) bool { calls++; return true })
	if calls != 1 {
		t.Fatalf("artwork checks=%d want 1", calls)
	}
}
