package web

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"mochil/internal/catalog"
	"mochil/internal/jobs"
)

func bookCheckKey(library string) string { return "book-check:" + library }

// qualityIssues are the findings of the quality check. They have their own filters and do not count towards "Needs
// review", which is for things that are wrong with a book's catalogue entry.
var qualityIssues = map[string]bool{"no_description": true, "no_isbn": true, "no_cover": true, "unreadable": true, "isbn_found": true, "isbn_differs": true, "isbn_unsure": true, "match_found": true, "match_unsure": true, "match_none": true, "hardcover_doubt": true, "hardcover_nomatch": true, "hardcover_ignored": true}

// quickQualityIssues needs nothing but the catalogue, so it is always on show.
func quickQualityIssues(book catalog.Book) []string {
	var found []string
	if strings.TrimSpace(book.Description) == "" {
		found = append(found, "no_description")
	}
	hasISBN := strings.HasPrefix(strings.ToLower(book.Identifier), "isbn")
	for scheme, value := range book.Identifiers {
		hasISBN = hasISBN || (strings.EqualFold(scheme, "isbn") && strings.TrimSpace(value) != "")
	}
	if !hasISBN {
		found = append(found, "no_isbn")
	}
	return found
}

// checkBook opens one book's file the way a reader would and says what is wrong with it. It only reads.
func checkBook(ctx context.Context, lib Library, book catalog.Book) []string {
	if book.Size == 0 {
		return []string{"unreadable"}
	}
	acquired, err := lib.Store.Acquire(ctx, book)
	if err != nil || acquired.Reader == nil {
		return []string{"unreadable"}
	}
	acquired.Reader.Close()
	// A cover can be looked for in a Calibre folder or an EPUB/CBZ archive; other formats have no place to keep one.
	provider, ranged := catalog.CoverCapability(lib.Store)
	switch format := strings.ToLower(book.Format); {
	case ranged:
		if cover, err := provider.Cover(ctx, book); err != nil || len(cover) == 0 {
			return []string{"no_cover"}
		}
	case format == "epub" || format == "cbz":
		if cover, err := archiveCover(ctx, lib.Store, book); err != nil || len(cover) == 0 {
			return []string{"no_cover"}
		}
	}
	return nil
}

var checkJob = editorJob{
	key:         bookCheckKey,
	available:   func(m *multiServer) bool { return m.bookQuality != nil },
	unavailable: "The quality check is not available right now.",
	busy:        "The check is already running for this library.",
	run: func(ctx context.Context, m *multiServer, lib Library, books []catalog.Book, _ editorJobOptions) (string, error) {
		store := m.bookQuality
		total := int64(len(books))
		problems := 0
		for index, book := range books {
			if ctx.Err() != nil {
				break
			}
			jobs.Report(ctx, int64(index), total, "Checking books")
			found := checkBook(ctx, lib, book)
			if len(found) > 0 {
				problems++
			}
			store.Put(lib.ID, book.ID, book.Size, book.Modified, found)
			if index%50 == 49 {
				_ = store.Save()
			}
		}
		if err := store.Save(); err != nil {
			return "", errors.New("the findings could not be saved")
		}
		message := "Checked " + strconv.Itoa(len(books)) + " books"
		if problems > 0 {
			message += "; " + strconv.Itoa(problems) + " with a missing cover or an unreadable file"
		} else {
			message += "; no missing covers or unreadable files"
		}
		return message, nil
	},
}

// editorCheck runs the quality check on some books (or the whole library) in the background.
func (m *multiServer) editorCheck(w http.ResponseWriter, r *http.Request) {
	m.serveEditorJob(w, r, checkJob)
}
