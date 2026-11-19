package ingest

import "testing"

func TestNamingPresets(t *testing.T) {
	index := func(value float64) *float64 { return &value }
	roxy := Proposal{Kind: Publication, Title: "Roxy (Side Story)", Authors: []string{"Rifujin"}, Series: "Mushoku Tensei"}
	vol3 := Proposal{Kind: Publication, Title: "Jobless Reincarnation", Authors: []string{"Rifujin"}, Series: "Mushoku Tensei", SeriesIndex: index(3)}
	half := Proposal{Kind: Publication, Title: "Interlude", Authors: []string{"Rifujin"}, Series: "Mushoku Tensei", SeriesIndex: index(1.5)}
	alone := Proposal{Kind: Publication, Title: "Neuromancer", Authors: []string{"William Gibson"}}
	tricky := Proposal{Kind: Publication, Title: `What/If: "Why?"`, Authors: []string{"A. Author"}, Series: "Series: One?"}
	cases := []struct {
		name, preset string
		proposal     Proposal
		source, want string
	}{
		{"kavita volume", NamingKavita, vol3, "a.epub", "Mushoku Tensei/Mushoku Tensei - Vol 03.epub"},
		{"kavita decimal volume", NamingKavita, half, "a.epub", "Mushoku Tensei/Mushoku Tensei - Vol 01.5.epub"},
		{"kavita side story is flat in the series", NamingKavita, roxy, "a.EPUB", "Mushoku Tensei/Mushoku Tensei - Side Story - Roxy.epub"},
		{"kavita standalone", NamingKavita, alone, "a.epub", "Neuromancer/Neuromancer.epub"},
		{"kavita unnumbered", NamingKavita, Proposal{Kind: Publication, Title: "Extra", Series: "S"}, "a.pdf", "S/S - Extra.pdf"},
		{"calibre layout", NamingCalibre, alone, "a.epub", "William Gibson/Neuromancer/Neuromancer - William Gibson.epub"},
		{"calibre without author", NamingCalibre, Proposal{Kind: Publication, Title: "Lost"}, "a.epub", "Unknown/Lost/Lost.epub"},
		{"default is unchanged", NamingDefault, vol3, "a.epub", "Rifujin/Mushoku Tensei/Mushoku Tensei - Vol 3 - Jobless Reincarnation.epub"},
		{"empty preset is the default", "", vol3, "a.epub", "Rifujin/Mushoku Tensei/Mushoku Tensei - Vol 3 - Jobless Reincarnation.epub"},
		{"unsafe characters never make a path", NamingKavita, tricky, "a.epub", "Series- One/Series- One - What-If- -Why.epub"},
	}
	for _, test := range cases {
		got, ok := TargetFor(test.preset, test.source, test.proposal)
		if !ok || got != test.want {
			t.Errorf("%s: got %q (%v), want %q", test.name, got, ok, test.want)
		}
	}
}

func TestANamelessBookIsNotInventedAName(t *testing.T) {
	for _, preset := range []string{NamingCalibre, NamingKavita} {
		if got, ok := TargetFor(preset, "a.epub", Proposal{Kind: Publication, Title: "  "}); ok {
			t.Errorf("%s named a book with no title: %q", preset, got)
		}
	}
}

func TestNamingNeverEscapesTheLibraryRoot(t *testing.T) {
	for _, preset := range []string{NamingCalibre, NamingKavita, NamingDefault} {
		got, _ := TargetFor(preset, "a.epub", Proposal{Kind: Publication, Title: "..", Authors: []string{"../.."}, Series: "../../etc"})
		if !safeJournalPath(got) && got != "" {
			t.Errorf("%s produced an unsafe path %q", preset, got)
		}
	}
	if !ValidNamingPreset("kavita") || ValidNamingPreset("../x") {
		t.Fatal("preset validation is wrong")
	}
}
