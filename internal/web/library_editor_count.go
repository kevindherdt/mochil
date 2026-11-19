package web

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"mochil/internal/bookstats"
	"mochil/internal/catalog"
	"mochil/internal/jobs"
)

func bookStatsKey(library string) string { return "book-stats:" + library }

// editorCounts is what the editor shows in the Words and Pages columns: the stored count if the book's file has not
// changed since, or the page count a comic archive carries itself.
func (m *multiServer) editorCounts(library string, book catalog.Book) (bookstats.Stats, bool) {
	if m.bookStats != nil {
		if stats, ok := m.bookStats.Get(library, book.ID, book.Size, book.Modified); ok {
			return stats, true
		}
	}
	if book.PageCount > 0 {
		return bookstats.Stats{Pages: book.PageCount}, true
	}
	return bookstats.Stats{}, false
}

// countBook measures one book's file. A book that cannot be counted (a format with nothing to count, a file that
// will not open) is skipped and reported, never fatal to the rest.
func countBook(ctx context.Context, lib Library, book catalog.Book) (bookstats.Stats, error) {
	acquired, err := lib.Store.Acquire(ctx, book)
	if err != nil {
		return bookstats.Stats{}, err
	}
	if acquired.Reader == nil {
		return bookstats.Stats{}, errors.New("this book's file is not available to read")
	}
	defer acquired.Reader.Close()
	return bookstats.Count(acquired.Reader, book.Format, book.PageCount)
}

var countJob = editorJob{
	key:         bookStatsKey,
	available:   func(m *multiServer) bool { return m.bookStats != nil },
	unavailable: "Counting is not available right now.",
	busy:        "Counting is already running for this library.",
	run: func(ctx context.Context, m *multiServer, lib Library, books []catalog.Book, recount bool) (string, error) {
		store := m.bookStats
		total := int64(len(books))
		skipped := 0
		for index, book := range books {
			if ctx.Err() != nil {
				break
			}
			jobs.Report(ctx, int64(index), total, "Counting words and pages")
			if !recount {
				if _, have := store.Get(lib.ID, book.ID, book.Size, book.Modified); have {
					continue
				}
			}
			stats, err := countBook(ctx, lib, book)
			if err != nil {
				skipped++
				continue
			}
			store.Put(lib.ID, book.ID, book.Size, book.Modified, stats)
			if index%50 == 49 {
				_ = store.Save()
			}
		}
		if err := store.Save(); err != nil {
			return "", errors.New("the counts could not be saved")
		}
		message := "Counted " + strconv.Itoa(len(books)-skipped) + " books"
		if skipped > 0 {
			message += "; " + strconv.Itoa(skipped) + " could not be read or have nothing to count"
		}
		return message, nil
	},
}

// editorCount counts words and pages for some books (or the whole library) in the background. Counting reads every
// file, so it runs once and the numbers are kept.
func (m *multiServer) editorCount(w http.ResponseWriter, r *http.Request) {
	m.serveEditorJob(w, r, countJob)
}
