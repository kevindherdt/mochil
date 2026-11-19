package musicnames

import (
	"strings"
	"testing"
)

func album() []Track {
	return []Track{
		{ID: "1", Path: "dl/Radiohead - OK Computer - 01 Airbag.MP3", Title: "Airbag", Artist: "Radiohead", Album: "OK Computer", Track: 1},
		{ID: "2", Path: "dl/02.mp3", Title: "Paranoid Android", Artist: "Radiohead", AlbumArtist: "Radiohead", Album: "OK Computer", Track: 2},
	}
}

func TestProposesArtistAlbumTrackTitleAndKeepsTheFileType(t *testing.T) {
	got := propose(album(), nil, false)
	if got[0].To != "Radiohead/OK Computer/01 - Airbag.mp3" || got[1].To != "Radiohead/OK Computer/02 - Paranoid Android.mp3" || got[0].Status != "change" {
		t.Fatalf("%+v", got)
	}
}

func TestAlreadyNamedTracksAreLeftAlone(t *testing.T) {
	tracks := album()[:1]
	tracks[0].Path = "Radiohead/OK Computer/01 - Airbag.mp3"
	if got := propose(tracks, nil, false); got[0].Status != "same" || got[0].To != "" {
		t.Fatalf("%+v", got)
	}
}

func TestUnsafeCharactersNeverReachThePathOrEscapeTheLibrary(t *testing.T) {
	tracks := []Track{{ID: "1", Path: "x.flac", Title: "../../etc/passwd: \"AC/DC\"?", Artist: "AC/DC", Album: "..", Track: 1}}
	got := propose(tracks, nil, false)
	if got[0].Status != "problem" && (strings.Contains(got[0].To, "..") || strings.Count(got[0].To, "/") != 2) {
		t.Fatalf("%+v", got)
	}
	if clean := Clean(`  ..a/b\c:d*e?f"g<h>i|j.  `); clean != "a b c d e f g h i j" {
		t.Fatalf("%q", clean)
	}
	if Clean("..") != "" || Clean("\x00\x01") != "" {
		t.Fatal("nothing left means empty")
	}
}

func TestMissingTagsAreProblemsNotGuesses(t *testing.T) {
	for _, change := range []func(*Track){
		func(t *Track) { t.Artist, t.AlbumArtist = "", "" },
		func(t *Track) { t.Album = "" },
		func(t *Track) { t.Title = "" },
		func(t *Track) { t.Track = 0 },
		func(t *Track) { t.Path = "noextension" },
	} {
		track := album()[0]
		change(&track)
		if got := propose([]Track{track}, nil, false); got[0].Status != "problem" || got[0].To != "" || got[0].Problem == "" {
			t.Fatalf("%+v", got)
		}
	}
}

func TestCollisionsAreCaughtInsideTheAlbumAndWithOtherFilesIgnoringCase(t *testing.T) {
	tracks := album()
	tracks[1].Title, tracks[1].Track = "AIRBAG", 1
	got := propose(tracks, nil, false)
	if got[0].Status != "problem" || got[1].Status != "problem" {
		t.Fatalf("two tracks, one name: %+v", got)
	}
	other := func(rel string) bool { return rel == "Radiohead/OK Computer/01 - Airbag.mp3" }
	if got := propose(album()[:1], other, false); got[0].Status != "problem" {
		t.Fatalf("an existing different file must block: %+v", got)
	}
	// a track already sitting at its own target is not a collision with itself
	own := album()[:1]
	own[0].Path = "radiohead/ok computer/01 - airbag.mp3"
	if got := propose(own, other, false); got[0].Status == "problem" {
		t.Fatalf("%+v", got)
	}
}

func TestMultiDiscAlbumsKeepTheirDiscNumber(t *testing.T) {
	tracks := []Track{
		{ID: "1", Path: "a.mp3", Title: "One", Artist: "A", Album: "B", Track: 1, Disc: 1},
		{ID: "2", Path: "b.mp3", Title: "One again", Artist: "A", Album: "B", Track: 1, Disc: 2},
	}
	got := propose(tracks, nil, false)
	if got[0].To != "A/B/1-01 - One.mp3" || got[1].To != "A/B/2-01 - One again.mp3" {
		t.Fatalf("%+v", got)
	}
}

func TestVeryLongNamesAreCutOrRefused(t *testing.T) {
	long := strings.Repeat("é", 300)
	got := propose([]Track{{ID: "1", Path: "a.mp3", Title: long, Artist: long, Album: long, Track: 1}}, nil, false)
	if got[0].Status != "problem" {
		t.Fatalf("%+v", got[0])
	}
	if len([]rune(Clean(long))) != maximumComponent {
		t.Fatal("one component is bounded")
	}
}

func TestFileNamesStayInsideTheTrackFolder(t *testing.T) {
	got := ProposeFileNames(album(), nil)
	if got[0].To != "dl/01 - Airbag.mp3" || got[1].To != "dl/02 - Paranoid Android.mp3" {
		t.Fatalf("%+v", got)
	}
}

func TestValidateRenameRefusesAnythingButASafeNameInTheSameFolder(t *testing.T) {
	good := [][2]string{{"dl/a.mp3", "dl/01 - A.mp3"}, {"dl/a.MP3", "dl/a.mp3"}, {"a.flac", "01 - Song.flac"}}
	for _, c := range good {
		if err := ValidateRename(c[0], c[1]); err != nil {
			t.Fatalf("%v: %v", c, err)
		}
	}
	bad := [][2]string{
		{"dl/a.mp3", "dl/a.mp3"}, {"dl/a.mp3", "other/a.mp3"}, {"dl/a.mp3", "dl/sub/a.mp3"}, {"dl/a.mp3", "../a.mp3"},
		{"dl/a.mp3", "dl/a.flac"}, {"dl/a.mp3", "dl/a"}, {"dl/a.mp3", "dl/ .mp3"}, {"dl/a.mp3", "dl/a:b.mp3"},
		{"dl/a.mp3", "dl/" + strings.Repeat("x", 300) + ".mp3"}, {"", "dl/a.mp3"},
	}
	for _, c := range bad {
		if ValidateRename(c[0], c[1]) == nil {
			t.Fatalf("%v must be refused", c)
		}
	}
}

func TestTidyMayOnlyRemoveWordsNeverAddOrChangeThem(t *testing.T) {
	cases := []struct{ original, cleaned, want string }{
		{"01 - RADIOHEAD - Airbag (Official Video) [HD]", "Airbag", "Airbag"},
		{"AIRBAG", "Airbag", "Airbag"},
		{"Airbag", "Airbag (Remastered)", "Airbag"},                // added a word: refused
		{"Parnoid Android", "Paranoid Android", "Parnoid Android"}, // "fixed" a word: refused
		{"Airbag", "", "Airbag"},
		{"Airbag", "!!!", "Airbag"},
		{"Song Song", "Song Song Song", "Song Song"},
	}
	for _, c := range cases {
		if got := Reconcile(c.original, c.cleaned); got != c.want {
			t.Fatalf("Reconcile(%q, %q) = %q, want %q", c.original, c.cleaned, got, c.want)
		}
	}
}

func TestTidyReplyIsParsedAndAppliedWithNotes(t *testing.T) {
	tidy, ok := ParseTidy("```json\n{\"tracks\":[{\"n\":1,\"title\":\"Airbag\",\"artist\":\"Radiohead\",\"album\":\"OK Computer\"},{\"n\":2,\"title\":\"Invented Words\",\"artist\":\"\",\"album\":\"\"}]}\n```")
	if !ok {
		t.Fatal("not parsed")
	}
	tracks := []Track{
		{Title: "01 - Radiohead - Airbag (Official Video)", Artist: "RADIOHEAD", Album: "OK COMPUTER", Track: 1},
		{Title: "Lucky", Artist: "Radiohead", Album: "OK Computer", Track: 2},
	}
	got, notes := ApplyTidy(tracks, tidy)
	if got[0].Title != "Airbag" || got[0].Artist != "Radiohead" || got[0].Album != "OK Computer" || len(notes[0]) != 3 {
		t.Fatalf("%+v %v", got[0], notes[0])
	}
	if got[1].Title != "Lucky" || len(notes[1]) != 0 {
		t.Fatalf("%+v %v", got[1], notes[1])
	}
	if tracks[0].Title == "Airbag" {
		t.Fatal("the original tracks must not be changed")
	}
	if _, ok := ParseTidy("I cleaned them all!"); ok {
		t.Fatal("prose is not an answer")
	}
}

func TestPreCleanRemovesJunkAndOnlyJunk(t *testing.T) {
	cases := []struct{ title, artist, album, wantTitle, wantArtist, wantAlbum string }{
		{"01 - RADIOHEAD - Airbag (Official Video) [HD]", "RADIOHEAD", "OK COMPUTER", "Airbag", "Radiohead", "OK Computer"},
		{"www.mp3site.com_paranoid_android", "Radiohead", "OK Computer (320kbps)", "paranoid android", "Radiohead", "OK Computer"},
		{"Exit Music (For A Film)", "Radiohead", "OK Computer", "Exit Music (For A Film)", "Radiohead", "OK Computer"},
		{"Highway to Hell", "AC/DC", "Highway to Hell (Remastered 2009)", "Highway to Hell", "AC/DC", "Highway to Hell (Remastered 2009)"},
		{"07. Lucky", "Radiohead", "", "Lucky", "Radiohead", ""},
		{"1999", "Prince", "1999", "1999", "Prince", "1999"},
		{"(Official Video)", "X", "Y", "(Official Video)", "X", "Y"}, // nothing real would be left: untouched
	}
	for _, c := range cases {
		got, _ := PreClean([]Track{{Title: c.title, Artist: c.artist, Album: c.album}})
		if got[0].Title != c.wantTitle || got[0].Artist != c.wantArtist || got[0].Album != c.wantAlbum {
			t.Fatalf("%q/%q/%q -> %q/%q/%q, want %q/%q/%q", c.title, c.artist, c.album, got[0].Title, got[0].Artist, got[0].Album, c.wantTitle, c.wantArtist, c.wantAlbum)
		}
	}
}
