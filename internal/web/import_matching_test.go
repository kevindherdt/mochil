package web

import (
	"fmt"
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

// The index is built once per request now; matching must behave exactly as the old per-page rebuild did. This is the
// equivalence pin, and it is also the cheap one: one library, two calls, same rows.
func TestMatchingFromAPrebuiltIndexMatchesTheOldPerPageCall(t *testing.T) {
	index := 2.0
	books := []catalog.Book{
		{ID: "a", Title: "Dune", Authors: []string{"Frank Herbert"}, Series: "Dune", SeriesIndex: &index},
		{ID: "b", Title: "Neuromancer", Authors: []string{"William Gibson"}},
	}
	items := []ingest.DiscoveryItem{
		{SourcePath: "incoming/dune-2.epub", Proposal: ingest.Proposal{Kind: ingest.Publication, Title: "Dune", Authors: []string{"Frank Herbert"}, Series: "Dune", SeriesIndex: &index}},
		{SourcePath: "incoming/neuromancer.epub", Proposal: ingest.Proposal{Kind: ingest.Publication, Title: "Neuromancer", Authors: []string{"William Gibson"}}},
	}
	built := buildImportIndex(books)
	fromIndex := built.match(items)
	fromCall := matchImports(items, books)
	if len(fromIndex) != len(fromCall) {
		t.Fatalf("index matching returned %d rows, the wrapper %d", len(fromIndex), len(fromCall))
	}
	for i := range fromIndex {
		if fromIndex[i].Status != fromCall[i].Status || fromIndex[i].SourcePath != fromCall[i].SourcePath {
			t.Errorf("row %d differs: %+v vs %+v", i, fromIndex[i], fromCall[i])
		}
	}
	if fromIndex[0].Status != importInLibrary {
		t.Errorf("a duplicate should still be recognised, got %q", fromIndex[0].Status)
	}
}

// The cost this refactor removed, as a number rather than an adjective: sixty discovery pages over a 5,000-book library,
// rebuilding the whole index per page versus building it once.
func benchmarkImportMatching(b *testing.B, perPageRebuild bool) {
	books := make([]catalog.Book, 5000)
	for i := range books {
		index := float64(i%40 + 1)
		books[i] = catalog.Book{ID: fmt.Sprintf("%05d", i), Title: fmt.Sprintf("Book %05d", i), Authors: []string{"A. Author"}, Series: "Series", SeriesIndex: &index}
	}
	items := make([]ingest.DiscoveryItem, 100)
	for i := range items {
		items[i] = ingest.DiscoveryItem{SourcePath: fmt.Sprintf("incoming/%03d.epub", i), Proposal: ingest.Proposal{Kind: ingest.Publication, Title: fmt.Sprintf("Book %05d", i), Authors: []string{"A. Author"}, Series: "Series"}}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if perPageRebuild {
			for page := 0; page < 60; page++ {
				_ = matchImports(items, books)
			}
		} else {
			index := buildImportIndex(books)
			for page := 0; page < 60; page++ {
				_ = index.match(items)
			}
		}
	}
}

func BenchmarkImportMatchingRebuiltPerPage(b *testing.B) { benchmarkImportMatching(b, true) }
func BenchmarkImportMatchingFromOneIndex(b *testing.B)   { benchmarkImportMatching(b, false) }
