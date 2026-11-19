package web

import (
	"testing"

	"mochil/internal/catalog"
	"mochil/internal/ingest"
)

func TestIncomingBooksAreComparedWithTheLibrary(t *testing.T) {
	number := func(value float64) *float64 { return &value }
	library := []catalog.Book{
		{ID: "b1", Title: "Mushoku Tensei - Volume 01", Series: "Mushoku Tensei", SeriesIndex: number(1), Authors: []string{"Rifujin"}},
		{ID: "b2", Title: "Mushoku Tensei - Volume 02", Series: "Mushoku Tensei", SeriesIndex: number(2), Authors: []string{"Rifujin"}},
		{ID: "b3", Title: "Naruto Shinden", Series: "Naruto Shinden (2018)", SeriesIndex: number(1)},
		{ID: "b4", Title: "Neuromancer", Authors: []string{"William Gibson"}},
		{ID: "b5", Title: "Dune"},
	}
	incoming := func(title, series string, index *float64, authors ...string) ingest.DiscoveryItem {
		return ingest.DiscoveryItem{SourcePath: title + ".epub", Proposal: ingest.Proposal{Title: title, Series: series, SeriesIndex: index, Authors: authors}}
	}
	rows := matchImports([]ingest.DiscoveryItem{
		incoming("Mushoku Tensei - Volume 01", "Mushoku Tensei", number(1)),     // same series and number
		incoming("Mushoku Tensei - Volume 09", "mushoku  tensei", number(9)),    // known series, new volume
		incoming("Naruto Shinden", "Naruto Shinden", number(1)),                 // series differs only by a year
		incoming("Brand New Series - Volume 01", "Brand New Series", number(1)), // nothing like it
		incoming("Neuromancer", "", nil, "William Gibson"),                      // title and author
		incoming("Dune", "", nil, "Someone Else"),                               // title only: a hint, not a match
		incoming("Mushoku Tensei - Volume 02", "Mushoku Tensei", nil),           // number unknown but the same title is in the library
	}, library)
	want := []struct{ status, existing string }{
		{importInLibrary, "b1"}, {importNewVolume, ""}, {importInLibrary, "b3"}, {importNew, ""}, {importInLibrary, "b4"}, {importPossible, "b5"}, {importPossible, "b2"},
	}
	for i, expected := range want {
		got := rows[i]
		existing := ""
		if got.Existing != nil {
			existing = got.Existing.ID
		}
		if got.Status != expected.status || existing != expected.existing {
			t.Errorf("row %d (%s): got %s/%s, want %s/%s", i, got.SourcePath, got.Status, existing, expected.status, expected.existing)
		}
	}
}

func TestMatchingAnEmptyLibraryMarksEverythingNew(t *testing.T) {
	rows := matchImports([]ingest.DiscoveryItem{{SourcePath: "a.epub", Proposal: ingest.Proposal{Title: "A", Series: "S"}}}, nil)
	if len(rows) != 1 || rows[0].Status != importNew || rows[0].Existing != nil {
		t.Fatalf("%+v", rows)
	}
}
