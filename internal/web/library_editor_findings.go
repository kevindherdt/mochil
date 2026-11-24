package web

import "mochil/internal/catalog"

// findingIssues are the labels the editor's background tools (quality check, ISBN search, Hardcover lookups) put on a
// book. They only describe what was found: nothing here changes a book.
func (m *multiServer) findingIssues(lib Library, book catalog.Book, facts hardcoverFacts) []string {
	found := quickQualityIssues(book)
	switch state, _ := facts.state(m, lib, book); state {
	case "doubt", "nomatch", "ignored":
		found = append(found, "hardcover_"+state)
	}
	if m.isbnFound != nil {
		if result, searched := m.isbnFound.Get(lib.ID, book.ID, book.Size, book.Modified); searched {
			switch {
			case result.Own != "" && hasISBN(book):
				// The book has an ISBN and the file has one too: only worth a badge when they are not the same
				// number, which is what a duplicated or wrong ISBN looks like.
				if !sameISBN(result.Own, bookISBN(book)) {
					found = append(found, "isbn_differs")
				}
			case result.Own != "":
				found = append(found, "isbn_found")
			case result.Ambiguous:
				found = append(found, "isbn_unsure")
			}
		}
	}
	if m.matchFound != nil && !m.hardcoverKnown(lib, book) {
		if match, looked := m.matchFound.Get(lib.ID, book.ID, book.Size, book.Modified); looked {
			switch {
			case match.Clear:
				found = append(found, "match_found")
			case len(match.Candidates) > 0:
				found = append(found, "match_unsure")
			default:
				found = append(found, "match_none")
			}
		}
	}
	if m.bookQuality != nil {
		problems, _ := m.bookQuality.Get(lib.ID, book.ID, book.Size, book.Modified)
		found = append(found, problems...)
	}
	return found
}

// addFindings fills the editor row's read-only fields from what the background tools found for the book.
func (m *multiServer) addFindings(row *editorRow, lib Library, book catalog.Book, facts hardcoverFacts) {
	row.Hardcover, row.HardcoverNote = facts.state(m, lib, book)
	// Always shown, not only for books without an ISBN: the ISBN printed in the file is what a wrong or duplicated
	// metadata ISBN has to be compared against, and hiding it for exactly those books hid the comparison.
	if m.isbnFound != nil {
		if result, searched := m.isbnFound.Get(lib.ID, book.ID, book.Size, book.Modified); searched {
			row.IsbnFound, row.IsbnNote = result.Own, isbnNote(result)
		}
	}
	// Shown even for a book already tied to a Hardcover book: applying a match adds that link, and hiding the finding
	// afterwards would take the series away the moment it was used — "Replace from the match" would work once, on the
	// first pass, and never again. The badge in findingIssues stays gated, because that one is about work to do.
	if m.matchFound != nil {
		if match, looked := m.matchFound.Get(lib.ID, book.ID, book.Size, book.Modified); looked {
			row.MatchClear, row.MatchNote = match.Clear, matchNote(match)
			if len(match.Candidates) > 0 {
				best := match.Candidates[0]
				row.Match = &best
			}
		}
	}
}
