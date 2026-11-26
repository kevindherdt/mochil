package web

import (
	"strings"
	"testing"
	"time"

	"mochil/internal/bookcheck"
	"mochil/internal/catalog"
	"mochil/internal/enrichment"
	"mochil/internal/filefacts"
	"mochil/internal/isbn"
)

// A row's badges and its own fields are read from one finding. Three lookups per book used to answer the same
// questions — the badge, the row, and the tooltip — and merging them is only safe if every label still lands where it
// did, in the order the editor lists it: what the catalogue alone shows, then what the background tools found, then
// what the quality check saw in the file.
func TestFindingsBadgesAndRowComeFromOneLookup(t *testing.T) {
	server := editorTestServer(t)
	var err error
	if server.isbnFound, err = filefacts.Open[isbn.Result](""); err != nil {
		t.Fatal(err)
	}
	if server.matchFound, err = filefacts.Open[bulkMatch](""); err != nil {
		t.Fatal(err)
	}
	if server.bookQuality, err = bookcheck.Open(""); err != nil {
		t.Fatal(err)
	}
	lib := server.libraries[0]
	modified := time.Unix(1700000000, 0)
	book := catalog.Book{ID: "a1", Title: "Dune", Identifiers: map[string]string{"isbn": "9780593098271"}, Size: 42, Modified: modified}
	server.isbnFound.Put(lib.ID, book.ID, book.Size, book.Modified, isbn.Result{Own: "9780306406157"})
	server.matchFound.Put(lib.ID, book.ID, book.Size, book.Modified, bulkMatch{Version: matchFindingVersion,
		Candidates: []matchCandidate{{Suggestion: enrichment.Suggestion{Title: "Dune Messiah", ProviderID: "6"}}}})
	server.bookQuality.Put(lib.ID, book.ID, book.Size, book.Modified, []string{"no_cover"})
	facts := hardcoverFacts{on: true, matches: map[string]hardcoverMatch{}, skipped: map[string]bool{},
		ignored: map[string]ignoredBook{linkKey(lib.ID, book.ID): {Reason: "taken off the shelf"}}}

	found := server.findings(lib, book, facts)
	var row editorRow
	found.apply(&row)
	row.Issues = found.badges(nil, book)

	if got := strings.Join(row.Issues, ","); got != "no_description,hardcover_ignored,isbn_differs,match_unsure,no_cover" {
		t.Fatalf("badges = %q", got)
	}
	if row.IsbnFound != "9780306406157" || row.IsbnNote != "" {
		t.Fatalf("the row carries the finding without wording its note: %+v", row)
	}
	if row.Hardcover != "ignored" || !strings.Contains(row.HardcoverNote, "taken off the shelf") {
		t.Fatalf("hardcover = %q %q", row.Hardcover, row.HardcoverNote)
	}
	if row.Match == nil || row.MatchClear || !strings.Contains(row.MatchNote, "Dune Messiah") {
		t.Fatalf("match = %+v %q", row.Match, row.MatchNote)
	}
}

// A book already tied to a Hardcover book is not work to do, so its badge stops — but its finding stays on the row,
// because applying it is what adds the link.
func TestALinkedBookKeepsItsRowFindingWithoutTheBadge(t *testing.T) {
	server := editorTestServer(t)
	var err error
	if server.matchFound, err = filefacts.Open[bulkMatch](""); err != nil {
		t.Fatal(err)
	}
	lib := server.libraries[0]
	book := catalog.Book{ID: "a1", Title: "Dune", Identifiers: map[string]string{"hardcover": "6"}}
	server.matchFound.Put(lib.ID, book.ID, book.Size, book.Modified, bulkMatch{Version: matchFindingVersion, Clear: true,
		Candidates: []matchCandidate{{Suggestion: enrichment.Suggestion{Title: "Dune", ProviderID: "6"}}}})

	found := server.findings(lib, book, hardcoverFacts{})
	var row editorRow
	found.apply(&row)
	issues := found.badges(nil, book)

	for _, issue := range issues {
		if issue == "match_found" {
			t.Fatalf("a linked book is not work to do: %v", issues)
		}
	}
	if row.Match == nil || !row.MatchClear {
		t.Fatalf("a linked book lost the finding it could still apply: %+v", row.Match)
	}
}
