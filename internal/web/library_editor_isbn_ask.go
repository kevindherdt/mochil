package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/isbn"
	"mochil/internal/jobs"
)

// Asking Hermes settles the close calls of "Find ISBNs": books whose ISBN the automatic search could not decide on, or
// found nothing for. It is a tie-breaker, never a source: Hermes may only choose a number the book really contains
// (checked again here, including the check digit), and the choice is stored as a finding beside the book, exactly like
// the automatic ones. Nothing is written to a book until the person uses "Use found ISBN" and saves.

func isbnAskKey(library string) string { return "isbn-ask:" + library }

// findBookISBNEvidence is findBookISBN plus the text around each number.
func findBookISBNEvidence(ctx context.Context, lib Library, book catalog.Book) (isbn.Result, isbn.Evidence, error) {
	acquired, err := lib.Store.Acquire(ctx, book)
	if err != nil {
		return isbn.Result{}, isbn.Evidence{}, err
	}
	if acquired.Reader == nil {
		return isbn.Result{}, isbn.Evidence{}, errNoFile
	}
	defer acquired.Reader.Close()
	return isbn.ExtractEvidence(acquired.Reader, book.Format, isbn.Hints{Title: book.Title, Volume: book.SeriesIndex})
}

var isbnAskJob = editorJob{
	key: isbnAskKey,
	available: func(m *multiServer) bool {
		settings := m.hermesSettings()
		return m.isbnFound != nil && settings.HelperConfigured() && settings.Feature("isbn")
	},
	unavailable: "Asking needs a helper model and the ISBN ability switched on in Settings, Hermes.",
	busy:        "Hermes is already looking at ISBNs in this library.",
	run: func(ctx context.Context, m *multiServer, lib Library, books []catalog.Book, _ bool) (string, error) {
		store, settings := m.isbnFound, m.hermesSettings()
		var asked, picked, unsure, nothing, unreadable, clear, hadISBN, alreadyAsked int
		for index, book := range books {
			if ctx.Err() != nil {
				break
			}
			jobs.Report(ctx, int64(index), int64(len(books)), "Asking Hermes about ISBNs")
			if hasISBN(book) {
				hadISBN++
				continue
			}
			if prior, have := store.Get(lib.ID, book.ID, book.Size, book.Modified); have && prior.Own != "" {
				clear++ // already settled, by the search or by an earlier question
				continue
			} else if have && prior.Asked {
				alreadyAsked++ // Hermes could not settle it last time; run Find ISBNs again to reopen it
				continue
			}
			result, evidence, err := findBookISBNEvidence(ctx, lib, book)
			if err != nil {
				unreadable++
				continue
			}
			if result.Own != "" || (len(result.Candidates) == 0 && evidence.FrontMatter == "") {
				store.Put(lib.ID, book.ID, book.Size, book.Modified, result) // nothing for Hermes to judge
				if result.Own != "" {
					clear++
				} else {
					nothing++
				}
				continue
			}
			if !m.waitHermesTurn(ctx) {
				break
			}
			asked++
			reportAsking(ctx, index, len(books), asked, book)
			facts := isbn.Facts{Title: book.Title, Authors: strings.Join(book.Authors, ", "), Series: book.Series, Volume: book.SeriesIndex, Format: book.Format}
			askCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
			reply, err := hermesAsk(askCtx, settings, isbn.AskInstructions, isbn.AskMessage(facts, result, evidence))
			cancel()
			m.endHermesTurn()
			if err != nil {
				if ctx.Err() != nil {
					break
				}
				_ = store.Save()
				if errors.Is(err, errHermesUnreachable) {
					return "", errors.New("Hermes did not answer. What was settled so far is kept.")
				}
				unsure++ // no usable answer: not marked, so the next run tries this book again
				continue
			}
			answer, ok := isbn.ParseAnswer(reply)
			chosen, verified := isbn.Verify(answer, result, evidence)
			if ok && verified && (answer.Confidence == "high" || answer.Confidence == "medium") {
				store.Put(lib.ID, book.ID, book.Size, book.Modified, isbn.Choose(result, chosen, answer))
				picked++
			} else {
				result.Asked = true
				store.Put(lib.ID, book.ID, book.Size, book.Modified, result)
				unsure++
			}
			if asked%10 == 0 {
				_ = store.Save()
			}
		}
		if err := store.Save(); err != nil {
			return "", errFindingsNotSaved
		}
		message := fmt.Sprintf("Asked Hermes about %d books: %d picked, %d still unsure", asked, picked, unsure)
		if nothing > 0 {
			message += ", " + strconv.Itoa(nothing) + " with no ISBN printed to judge"
		}
		if unreadable > 0 {
			message += ", " + strconv.Itoa(unreadable) + " could not be read"
		}
		if alreadyAsked > 0 {
			message += fmt.Sprintf(". %d were asked before and stay unsure", alreadyAsked)
		}
		if asked == 0 && unreadable == 0 && nothing == 0 {
			message = "Nothing to ask: " + strconv.Itoa(hadISBN) + " already have an ISBN, " + strconv.Itoa(clear) + " have a clear one found and " + strconv.Itoa(alreadyAsked) + " were asked before and stay unsure."
		}
		return message, nil
	},
}

// editorISBNAsk asks Hermes about the books the ISBN search could not settle. Findings only; see above.
func (m *multiServer) editorISBNAsk(w http.ResponseWriter, r *http.Request) {
	m.serveEditorJob(w, r, isbnAskJob)
}
