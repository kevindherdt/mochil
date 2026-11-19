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
	if m.isbnFound != nil && !hasISBN(book) {
		if result, searched := m.isbnFound.Get(lib.ID, book.ID, book.Size, book.Modified); searched {
			switch {
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
	if m.isbnFound != nil && !hasISBN(book) {
		if result, searched := m.isbnFound.Get(lib.ID, book.ID, book.Size, book.Modified); searched {
			row.IsbnFound, row.IsbnNote = result.Own, isbnNote(result)
		}
	}
	if m.matchFound != nil && !m.hardcoverKnown(lib, book) {
		if match, looked := m.matchFound.Get(lib.ID, book.ID, book.Size, book.Modified); looked {
			row.MatchClear, row.MatchNote = match.Clear, matchNote(match)
			if len(match.Candidates) > 0 {
				best := match.Candidates[0]
				row.Match = &best
			}
		}
	}
}
