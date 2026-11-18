package ingest

import "testing"

func TestInferPublicationUsesExplicitStructure(t *testing.T) {
	p := InferPath(Publication, "Books/Brandon Sanderson/Mistborn/Mistborn - Vol 01 - The Final Empire [isbn=9780765311788].epub")
	if p.Title != "The Final Empire" || p.Series != "Mistborn" || len(p.Authors) != 1 || p.Authors[0] != "Brandon Sanderson" || p.SeriesIndex == nil || *p.SeriesIndex != 1 || p.Identifiers["isbn"] != "9780765311788" {
		t.Fatalf("unexpected proposal: %#v", p)
	}
	if !hasEvidence(p, "series_index", High) || !hasEvidence(p, "identifier.isbn", High) {
		t.Fatalf("missing strong evidence: %#v", p.Evidence)
	}
}

func TestInferVideoEpisodeAndMovie(t *testing.T) {
	e := InferPath(Video, "The Expanse (2015) [tmdbid=63639]/Season 02/The.Expanse.S02E03.Static.mkv")
	if e.Series != "The Expanse" || e.Year != 2015 || e.Season != 2 || e.Episode != 3 || e.Title != "Static" || e.Identifiers["tmdbid"] != "63639" {
		t.Fatalf("unexpected episode: %#v", e)
	}
	m := InferPath(Video, "Movies/Arrival (2016) [tmdbid=329865].mkv")
	if m.Title != "Arrival" || m.Year != 2016 || m.Identifiers["tmdbid"] != "329865" {
		t.Fatalf("unexpected movie: %#v", m)
	}
}

func TestInferMusicDiscTrack(t *testing.T) {
	p := InferPath(Music, "Music/Daft Punk/Discovery (2001)/01-03 - Digital Love.flac")
	if p.AlbumArtist != "Daft Punk" || p.Album != "Discovery" || p.Year != 2001 || p.Disc != 1 || p.Track != 3 || p.Title != "Digital Love" {
		t.Fatalf("unexpected music proposal: %#v", p)
	}
}

func TestInferenceDoesNotGuessBareNumbers(t *testing.T) {
	p := InferPath(Publication, "Author/Series/1984.epub")
	if p.SeriesIndex != nil || p.Year != 0 {
		t.Fatalf("bare number was guessed: %#v", p)
	}
	v := InferPath(Video, "Holiday 2024 1080p.mkv")
	if v.Season != 0 || v.Episode != 0 || v.Year != 0 {
		t.Fatalf("release numbers were guessed: %#v", v)
	}
}

func hasEvidence(p Proposal, field string, confidence Confidence) bool {
	for _, evidence := range p.Evidence {
		if evidence.Field == field && evidence.Confidence == confidence {
			return true
		}
	}
	return false
}
