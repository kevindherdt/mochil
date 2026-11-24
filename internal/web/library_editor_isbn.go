package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"mochil/internal/catalog"
	"mochil/internal/isbn"
	"mochil/internal/jobs"
)

var (
	errNoFile           = errors.New("this book's file is not available to read")
	errFindingsNotSaved = errors.New("the findings could not be saved")
)

func isbnFindKey(library string) string { return "isbn-find:" + library }

// hasISBN says whether a book already carries an ISBN in its identifiers.
func hasISBN(book catalog.Book) bool {
	for scheme, value := range book.Identifiers {
		if strings.EqualFold(scheme, "isbn") && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return strings.HasPrefix(strings.ToLower(book.Identifier), "isbn")
}

// bookISBN is the ISBN a book already carries in its metadata, normalised to digits (and a trailing X).
func bookISBN(book catalog.Book) string {
	for scheme, value := range book.Identifiers {
		if strings.EqualFold(scheme, "isbn") {
			return normaliseISBN(value)
		}
	}
	if strings.HasPrefix(strings.ToLower(book.Identifier), "isbn") {
		return normaliseISBN(book.Identifier)
	}
	return ""
}

func normaliseISBN(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if (r >= '0' && r <= '9') || r == 'x' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sameISBN says whether two ISBNs are the same number, ignoring hyphens and spacing — "978-1-945054-48-8" and
// "9781945054488" are one ISBN, and treating them as a mismatch would report a difference on every book.
func sameISBN(a, b string) bool {
	left, right := normaliseISBN(a), normaliseISBN(b)
	return left != "" && left == right
}

// findBookISBN reads one book's file for its ISBN. It only reads.
func findBookISBN(ctx context.Context, lib Library, book catalog.Book) (isbn.Result, error) {
	acquired, err := lib.Store.Acquire(ctx, book)
	if err != nil {
		return isbn.Result{}, err
	}
	if acquired.Reader == nil {
		return isbn.Result{}, errNoFile
	}
	defer acquired.Reader.Close()
	return isbn.Extract(acquired.Reader, book.Format, isbn.Hints{Title: book.Title, Volume: book.SeriesIndex})
}

var isbnJob = editorJob{
	key:         isbnFindKey,
	available:   func(m *multiServer) bool { return m.isbnFound != nil },
	unavailable: "Looking for ISBNs is not available right now.",
	busy:        "The ISBN search is already running for this library.",
	run: func(ctx context.Context, m *multiServer, lib Library, books []catalog.Book, opts editorJobOptions) (string, error) {
		store := m.isbnFound
		total := int64(len(books))
		var searched, clear, unsure, none, unreadable, unsupported, hadISBN, before int
		for index, book := range books {
			if ctx.Err() != nil {
				break
			}
			jobs.Report(ctx, int64(index), total, "Looking for ISBNs")
			// A book that already carries an ISBN is usually "nothing to find" and is skipped, which keeps the scan
			// cheap. With Recheck the person has asked the opposite question — is the ISBN in the metadata the one
			// printed in the file? — so those books are read as well.
			if hasISBN(book) && !opts.Recheck {
				hadISBN++
				continue
			}
			if _, have := store.Get(lib.ID, book.ID, book.Size, book.Modified); have && !opts.Recount {
				before++
				continue
			}
			result, err := findBookISBN(ctx, lib, book)
			if errors.Is(err, isbn.ErrUnsupported) {
				// A format this search cannot open is not "no ISBN printed": storing a
				// zero result would report that falsehood forever.
				unsupported++
				continue
			}
			if err != nil {
				unreadable++
				continue
			}
			searched++
			switch {
			case result.Own != "":
				clear++
			case result.Ambiguous:
				unsure++
			default:
				none++
			}
			store.Put(lib.ID, book.ID, book.Size, book.Modified, result)
			if index%50 == 49 {
				_ = store.Save()
			}
		}
		if err := store.Save(); err != nil {
			return "", errFindingsNotSaved
		}
		message := "Searched " + strconv.Itoa(searched) + " books: " + strconv.Itoa(clear) + " with a clear ISBN, " + strconv.Itoa(unsure) + " unsure, " + strconv.Itoa(none) + " with none"
		if unreadable > 0 {
			message += "; " + strconv.Itoa(unreadable) + " could not be read"
		}
		if unsupported > 0 {
			message += "; " + strconv.Itoa(unsupported) + " are in a format this search cannot open"
		}
		if searched == 0 && unreadable == 0 && unsupported == 0 {
			message = "Nothing new to search: " + strconv.Itoa(hadISBN) + " already have an ISBN and " + strconv.Itoa(before) + " were searched before (their findings are kept)."
		}
		return message, nil
	},
}

// editorISBN looks inside books for their ISBN in the background; nothing is written to the books or the catalogue.
// What it found is shown beside each book, and taken only when the person chooses.
func (m *multiServer) editorISBN(w http.ResponseWriter, r *http.Request) {
	m.serveEditorJob(w, r, isbnJob)
}

// How much of a book's ISBN pile the note spells out. A book can print a dozen numbers — a colophon, a spine, a footer
// on every page — and the note is a tooltip on one cell, so listing all of them with all their reasons said the useful
// thing once and then kept going.
const (
	isbnNoteExtras  = 2
	isbnNoteMaximum = 320
)

// isbnNote words what was found for a book in a few sentences, with the reasons for the choice.
//
// The first two after the best are named and the rest are counted: that is the part somebody reads ("which number, why,
// and how many others were in there") and it keeps the field small. Listed in full it was the largest field on an editor
// row — 233 kB of a 935 kB page measured on a 5 544-book library, and up to 2 224 characters for one book whose EPUB
// prints an ISBN on every chapter footer.
func isbnNote(result isbn.Result) string {
	if len(result.Candidates) == 0 {
		return "No ISBN is printed in this book."
	}
	top := result.Candidates[0]
	why := func(c isbn.Candidate) string { return c.ISBN + " (" + strings.Join(c.Reasons, ", ") + ")" }
	var note string
	rest := result.Candidates[1:]
	if result.Own != "" {
		note = "Best match: " + strings.Join(top.Reasons, ", ") + "."
	} else {
		note = "Not certain which is this book's own. Closest: " + why(top) + "."
	}
	if len(rest) > 0 {
		shown := rest
		if len(shown) > isbnNoteExtras {
			shown = shown[:isbnNoteExtras]
		}
		others := make([]string, len(shown))
		for i, other := range shown {
			others[i] = why(other)
		}
		note += " Also in the book: " + strings.Join(others, "; ")
		if hidden := len(rest) - len(shown); hidden > 0 {
			note += fmt.Sprintf(" and %d more", hidden)
		}
		note += "."
	}
	return bounded(isbnNoteMaximum, note)
}

// bounded cuts s to at most limit runes, ending with an ellipsis, without splitting a multi-byte character in half.
func bounded(limit int, s string) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit-1]) + "…"
}
