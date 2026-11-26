package web

import (
	"mochil/internal/catalog"
	"mochil/internal/isbn"
)

// bookFindings is what the editor's background tools (quality check, ISBN search, Hardcover lookups) know about one
// book, read once. They only describe what was found: nothing here changes a book.
// A chunk used to ask the same stores two or three
// times per book — once for the badge, once when the row was built, and the ISBN once more for the tooltip of the rows
// being sent — each lookup a locked map read compared against the file's size and date. On 5 430 books that was
// ~16 000 lookups for one request; now it is one of each, and the row carries what it read.
type bookFindings struct {
	hardcover     string // how the book stands with the Hardcover sync, see hardcoverFacts.state
	hardcoverNote string
	// linked is set when the book is already tied to a Hardcover book, so a found match is not news to act on.
	linked     bool
	isbn       isbn.Result
	isbnKnown  bool
	match      bulkMatch
	matchKnown bool
	// quality is what the quality check found in the file itself, appended after the badges.
	quality []string
}

// findings reads what the background tools found for one book.
func (m *multiServer) findings(lib Library, book catalog.Book, facts hardcoverFacts) bookFindings {
	var found bookFindings
	found.hardcover, found.hardcoverNote = facts.state(m, lib, book)
	if m.isbnFound != nil {
		found.isbn, found.isbnKnown = m.isbnFound.Get(lib.ID, book.ID, book.Size, book.Modified)
	}
	if m.matchFound != nil {
		found.match, found.matchKnown = m.matchFound.Get(lib.ID, book.ID, book.Size, book.Modified)
		found.linked = m.hardcoverKnown(lib, book)
	}
	if m.bookQuality != nil {
		found.quality, _ = m.bookQuality.Get(lib.ID, book.ID, book.Size, book.Modified)
	}
	return found
}

// badges adds the found labels to the issues the editor already worked out for this book, in the order the editor
// lists them: the Hardcover state, the ISBN, the match, then the quality check's own findings.
func (f bookFindings) badges(into []string, book catalog.Book) []string {
	into = quickQualityIssues(into, book)
	switch f.hardcover {
	case "doubt", "nomatch", "ignored":
		into = append(into, "hardcover_"+f.hardcover)
	}
	if f.isbnKnown {
		switch {
		case f.isbn.Own != "" && hasISBN(book):
			// The book has an ISBN and the file has one too: only worth a badge when they are not the same
			// number, which is what a duplicated or wrong ISBN looks like.
			if !sameISBN(f.isbn.Own, bookISBN(book)) {
				into = append(into, "isbn_differs")
			}
		case f.isbn.Own != "":
			into = append(into, "isbn_found")
		case f.isbn.Ambiguous:
			into = append(into, "isbn_unsure")
		}
	}
	// The match badge stays gated on the book not being linked yet: it is about work to do, and applying a match is
	// the work. The row's own match fields are not gated — see apply.
	if f.matchKnown && !f.linked {
		switch {
		case f.match.Clear:
			into = append(into, "match_found")
		case len(f.match.Candidates) > 0:
			into = append(into, "match_unsure")
		default:
			into = append(into, "match_none")
		}
	}
	return append(into, f.quality...)
}

// apply fills the editor row's read-only fields from what the background tools found. It only reads.
func (f bookFindings) apply(row *editorRow) {
	row.Hardcover, row.HardcoverNote = f.hardcover, f.hardcoverNote
	// Always shown, not only for books without an ISBN: the ISBN printed in the file is what a wrong or duplicated
	// metadata ISBN has to be compared against, and hiding it for exactly those books hid the comparison.
	if f.isbnKnown {
		row.IsbnFound = f.isbn.Own
	}
	// Shown even for a book already tied to a Hardcover book: applying a match adds that link, and hiding the finding
	// afterwards would take the series away the moment it was used — "Replace from the match" would work once, on the
	// first pass, and never again. The badge in badges stays gated, because that one is about work to do.
	if f.matchKnown {
		row.MatchClear, row.MatchNote = f.match.Clear, matchNote(f.match)
		if len(f.match.Candidates) > 0 {
			best := f.match.Candidates[0]
			row.Match = &best
		}
	}
}

// addFindingNote is the sentence about a found ISBN, which a person reads in a tooltip rather than sorts on. It is
// filled for the rows a response actually carries, once the page is known: wording one for every book in the library
// costs a lookup and a built string per book, and a chunk request builds every row only to sort them.
func (m *multiServer) addFindingNote(row *editorRow, lib Library) {
	if m.isbnFound == nil || row.book == nil {
		return
	}
	if result, searched := m.isbnFound.Get(lib.ID, row.book.ID, row.book.Size, row.book.Modified); searched {
		row.IsbnNote = isbnNote(result)
	}
}
