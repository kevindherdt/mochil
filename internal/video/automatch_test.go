package video

import "testing"

func TestParseMediaName(t *testing.T) {
	for raw, want := range map[string][2]any{
		"The Witch - Part 1 The Subversion (2018) Bluray-1080p": {"The Witch - Part 1 The Subversion", 2018},
		"Baki Hanma (2021) [tvdbid=362999]":                     {"Baki Hanma", 2021},
		"Silo":                                                  {"Silo", 0},
		"The World of Kanako (2014) WEBDL-2160p":                {"The World of Kanako", 2014},
	} {
		title, year := ParseMediaName(raw)
		if title != want[0] || year != want[1] {
			t.Errorf("%q = %q, %d; want %v", raw, title, year, want)
		}
	}
}

func TestConfidentMatchNeedsSameWordsAndYear(t *testing.T) {
	if !ConfidentMatch("Spider-Man: Homecoming", 2017, "Spider Man Homecoming", 2017) {
		t.Fatal("punctuation must not matter")
	}
	if !ConfidentMatch("When They Cry - Higurashi", 2006, "Higurashi: When They Cry", 2006) || ConfidentMatch("When They Cry - Higurashi", 0, "Higurashi: When They Cry", 0) {
		t.Fatal("reordered words match only with an agreeing year")
	}
	if ConfidentMatch("Dune", 1984, "Dune", 2021) {
		t.Fatal("a different year is a different film")
	}
	if ConfidentMatch("Mouse", 2021, "Mouse Hunt", 1997) {
		t.Fatal("a longer title is not a match")
	}
	if !ConfidentMatch("Silo", 0, "Silo", 2023) {
		t.Fatal("no year in the name accepts the found year")
	}
}
