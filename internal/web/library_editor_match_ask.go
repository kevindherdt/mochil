package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/jobs"
	"mochil/internal/matchask"
)

func matchAskKey(library string) string { return "match-ask:" + library }

func askBook(book catalog.Book) matchask.Book {
	return matchask.Book{Title: book.Title, Authors: strings.Join(book.Authors, ", "), Series: book.Series, Volume: book.SeriesIndex, Published: book.Published, Description: book.Description}
}

func askCandidates(candidates []matchCandidate) []matchask.Candidate {
	out := make([]matchask.Candidate, len(candidates))
	for i, c := range candidates {
		out[i] = matchask.Candidate{Title: c.Title, Authors: strings.Join(c.Authors, ", "), Year: c.Year, Overview: c.Overview, Score: c.Score}
	}
	return out
}

// matchAskJob asks the helper model about the books the Hardcover title search could not settle. The model may only
// choose one of the candidates Hardcover already returned (and only a confident choice that resembles the title);
// the result is a finding, applied only when the person uses it. It reads what the lookup stored and never calls Hardcover.
var matchAskJob = editorJob{
	key: matchAskKey,
	available: func(m *multiServer) bool {
		settings := m.hermesSettings()
		return m.matchFound != nil && settings.AskConfigured() && settings.Feature("match")
	},
	unavailable: "Asking needs a helper model and the Hardcover match ability switched on in Settings, Hermes. Run Find Hardcover matches first.",
	busy:        "Hermes is already looking at Hardcover matches in this library.",
	run: func(ctx context.Context, m *multiServer, lib Library, books []catalog.Book, _ editorJobOptions) (string, error) {
		store, settings := m.matchFound, m.hermesSettings()
		var asked, picked, unsure, waiting int
		for index, book := range books {
			if ctx.Err() != nil {
				break
			}
			jobs.Report(ctx, int64(index), int64(len(books)), "Asking Hermes about Hardcover matches")
			if m.hardcoverKnown(lib, book) {
				continue
			}
			match, looked := store.Get(lib.ID, book.ID, book.Size, book.Modified)
			if !looked || match.How != "title" || match.Clear || match.Asked || len(match.Candidates) == 0 {
				continue
			}
			if !m.waitHermesTurn(ctx) {
				break
			}
			asked++
			reportAsking(ctx, index, len(books), asked, book)
			askCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
			reply, err := hermesAsk(askCtx, settings, matchask.Instructions, matchask.Message(askBook(book), askCandidates(match.Candidates)))
			cancel()
			m.endHermesTurn()
			if err != nil {
				if ctx.Err() != nil {
					break
				}
				_ = store.Save()
				if errors.Is(err, errHermesUnreachable) {
					return "", errors.New("The helper model did not answer. What was settled so far is kept.")
				}
				waiting++ // a bad reply is not recorded as asked: the next run may try again
				continue
			}
			match.Asked = true
			answer, ok := matchask.ParseAnswer(reply)
			if choice, verified := matchask.Verify(answer, askCandidates(match.Candidates)); ok && verified {
				chosen := match.Candidates[choice]
				rest := append(append([]matchCandidate{}, match.Candidates[:choice]...), match.Candidates[choice+1:]...)
				match.Candidates = append([]matchCandidate{chosen}, rest...)
				match.Clear = true
				match.Picked = answer.Confidence + " confidence"
				if answer.Reason != "" {
					match.Picked += ": " + answer.Reason
				}
				picked++
			} else {
				unsure++
			}
			store.Put(lib.ID, book.ID, book.Size, book.Modified, match)
			if asked%10 == 0 {
				_ = store.Save()
			}
		}
		if err := store.Save(); err != nil {
			return "", errFindingsNotSaved
		}
		if asked == 0 {
			// The old sentence claimed the match run had not happened, which is false whenever it has: a book whose
			// match is certain, or that is already linked to Hardcover, is not something to ask about.
			if len(books) > 0 {
				return fmt.Sprintf("Nothing to ask: none of the %d selected books has an unsure match waiting. They already have a certain match stored, or are linked to Hardcover already.", len(books)), nil
			}
			return "Nothing to ask: no book has an unsure Hardcover match waiting. Run Find Hardcover matches first.", nil
		}
		message := fmt.Sprintf("Asked Hermes about %d books: %d picked, %d still unsure", asked, picked, unsure)
		if waiting > 0 {
			message += fmt.Sprintf(", %d got no usable answer and stay for the next run", waiting)
		}
		return message, nil
	},
}

// editorMatchAsk asks the helper model about unsure Hardcover matches. Findings only; see above.
func (m *multiServer) editorMatchAsk(w http.ResponseWriter, r *http.Request) {
	m.serveEditorJob(w, r, matchAskJob)
}
