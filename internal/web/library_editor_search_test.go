package web

import (
	"strings"
	"testing"
)

// editorSearchReference is the shape the row search had before it was folded once per request: build the row's
// haystack, then check every word the query splits into. The tests compare the folded search against it, so a change
// to how a row is folded or matched has to agree with what the editor sent before.
func editorSearchReference(row editorRow, query string) bool {
	if query == "" {
		return true
	}
	haystack := strings.ToLower(row.Title + " " + strings.Join(row.Authors, " ") + " " + row.Series + " " + row.Path)
	for _, word := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(haystack, word) {
			return false
		}
	}
	return true
}

func editorSearchRows() []editorRow {
	return []editorRow{
		{Title: "Dune", Authors: []string{"Frank Herbert"}, Series: "Dune", Path: "Dune/Dune 1.epub"},
		{Title: "Dune Messiah", Authors: []string{"Frank Herbert", "Brian Herbert"}, Series: "Dune", Path: "Dune/Dune 2.epub"},
		{Title: "", Authors: nil, Path: ""},
		{Title: "Été", Authors: []string{"Ōe Kenzaburō"}, Series: "日本", Path: "ete.epub"},
		{Title: "İstanbul: a city", Authors: []string{"Orhan Pamuk"}, Path: "istanbul.epub"},
		{Title: "weird \xff\xfe bytes", Authors: []string{"So\xffmeone"}, Path: "weird.epub"},
		{Title: "A  Double  Space", Authors: []string{" ", ""}, Series: "  ", Path: "spaces.epub"},
		{Title: "Side Story", Authors: []string{"Yuzuka Morita"}, Series: "Hack GU", Path: "side.epub"},
		{Title: "g.u.", Authors: []string{"Tatsuya Hamazaki"}, Series: "", Path: "gu.epub"},
		{Title: "Dragon Ball", Authors: []string{"Akira Toriyama"}, Series: "Dragon Ball", Path: "dragon/1.cbz"},
	}
}

// editorSearchQueries exercise what the haystack used to see: case, several words, a phrase that only matches across
// the separator between two fields, non-ASCII, invalid UTF-8, punctuation and blank space.
var editorSearchQueries = []string{
	"", " ", "   ", "dune", "DUNE", "DuNe", "frank herbert", "dune frank", "herbert dune",
	"the", "side story", "side  story", "g.u.", "1.epub", "dragon", "?",
	"istanbul", "İstanbul", "i̇stanbul", "n pamuk", "été", "ete", "ōe", "oe", "日本",
	"weird", "\xff", "so\xffmeone", "double space", "a  double  space", "zzz",
}

func TestEditorSearchAgreesWithTheHaystackItReplaced(t *testing.T) {
	for _, row := range editorSearchRows() {
		for _, query := range editorSearchQueries {
			trimmed := strings.TrimSpace(query)
			want := editorSearchReference(row, trimmed)
			got := true
			if search := newEditorSearch(trimmed); search != nil {
				got = search.matches(row)
			}
			if got != want {
				t.Fatalf("query %q on %+v: folded search says %v, the haystack said %v", query, row, got, want)
			}
		}
	}
}

func TestEditorSearchMatchesWordsAcrossFieldBoundaries(t *testing.T) {
	// "dune frank" only matches because the haystack joined the title and the authors with a space; folding the
	// fields separately must keep that, or a person searching for an author by a word that ends the title stops
	// finding the book.
	search := newEditorSearch("dune frank")
	row := editorSearchRows()[0]
	if search == nil || !search.matches(row) {
		t.Fatal("a query spanning the title and the first author no longer matches")
	}
	if search := newEditorSearch("frank dune"); search == nil || !search.matches(editorSearchRows()[1]) {
		t.Fatal("a query in any word order should match")
	}
	if search := newEditorSearch("dune brian"); search == nil || search.matches(editorSearchRows()[0]) {
		t.Fatal("a word from another book matched")
	}
}

func TestEditorSearchDoesNotAllocatePerRow(t *testing.T) {
	// The whole point of the folded search: a request matches every book in the library, and the old shape
	// allocated three times per book (the haystack, the lowered query, the split query). Plain ASCII rows are the
	// common case and must cost nothing; a field with a byte above ASCII goes through strings.ToLower, which is one
	// string per such field rather than three per row.
	rows := make([]editorRow, 0, 500)
	for i := 0; i < 500; i++ {
		rows = append(rows, editorRow{
			Title: searchTitle(i), Authors: []string{"Frank Herbert", "Brian Herbert"}, Series: "Dune",
			Path: "Dune/Dune " + strings.Repeat("x", i%40) + ".epub",
		})
	}
	search := newEditorSearch("dune herbert")
	if search == nil {
		t.Fatal("expected a search")
	}
	if !search.matches(rows[0]) {
		t.Fatal("expected the first row to match, so the buffer is warm")
	}
	if allocs := testing.AllocsPerRun(5, func() {
		for _, row := range rows {
			search.matches(row)
		}
	}); allocs != 0 {
		t.Fatalf("matching %d ASCII rows allocates %v times per pass", len(rows), allocs)
	}
	// Non-ASCII text still folds exactly (the equivalence test above pins that), but through strings.ToLower.
	mixed := make([]editorRow, 0, 500)
	for i := 0; i < 100; i++ {
		mixed = append(mixed, editorSearchRows()[4]) // İstanbul: one non-ASCII field
	}
	mixedSearch := newEditorSearch("istanbul")
	mixedSearch.matches(mixed[0])
	if allocs := testing.AllocsPerRun(5, func() {
		for _, row := range mixed {
			mixedSearch.matches(row)
		}
	}); allocs > float64(len(mixed)) {
		t.Fatalf("a row with one non-ASCII field costs more than one allocation: %v for %d rows", allocs, len(mixed))
	}
}

// searchTitle answers an ASCII title unique to its number, so a run of rows exercises a buffer that stays the same size.
func searchTitle(i int) string {
	return "Dune Book " + strings.Repeat("n", i%7)
}

func TestEditorMatchesAppliesTheFilterBeforeTheSearch(t *testing.T) {
	row := editorRow{Title: "Dune", Authors: []string{"Frank Herbert"}, Path: "Dune.epub", Issues: []string{"duplicate"}}
	if !editorMatches(row, "duplicate", newEditorSearch("dune")) {
		t.Fatal("a book with the issue and the word should match")
	}
	if editorMatches(row, "duplicate", newEditorSearch("neuromancer")) {
		t.Fatal("the search must still be applied")
	}
	if !editorMatches(row, "issues", newEditorSearch("dune")) {
		t.Fatal("a duplicate is an issue the filter means")
	}
	// A quality finding alone is not what the issues filter means: that filter is for problems someone has to
	// decide about, not for books simply missing a description.
	quality := editorRow{Title: "Dune", Issues: []string{"no_isbn"}}
	if editorMatches(quality, "issues", nil) {
		t.Fatal("a quality finding alone should not pass the issues filter")
	}
	if !editorMatches(row, "", nil) {
		t.Fatal("no search and no filter matches every row")
	}
}
