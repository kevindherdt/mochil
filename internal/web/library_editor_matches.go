package web

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/enrichment"
	"mochil/internal/isbn"
	"mochil/internal/jobs"
)

func hardcoverMatchKey(library string) string { return "hardcover-match:" + library }

// matchFindingVersion is the shape of a stored finding. A finding written by an older version is missing fields the
// editor now uses — series and its position arrived in version 2 — and there is no way to tell "Hardcover does not know
// the series" from "this was looked up before we asked". Such a finding is refreshed on the next run rather than shown
// as if it were complete.
const matchFindingVersion = 2

// bulkMatch is what looking a book up on Hardcover found. It is only a finding: nothing is applied to the book until
// the person takes it, and the lookup is kept for as long as the book's file stays the same.
type bulkMatch struct {
	Version    int              `json:"v,omitempty"`
	How        string           `json:"how"` // "isbn" (exact) or "title"
	ISBN       string           `json:"isbn,omitempty"`
	Clear      bool             `json:"clear"`
	Candidates []matchCandidate `json:"candidates,omitempty"`
	// Asked is set once the helper model has judged this close call, so a later run does not ask again; Picked is its
	// reason when it chose one of the candidates (which is then first in the list and Clear).
	Asked  bool   `json:"asked,omitempty"`
	Picked string `json:"picked,omitempty"`
}

const maximumStoredCandidates = 5

// matchIsClear is the rule the single-book dialog uses: one candidate well ahead of the rest.
func matchIsClear(candidates []matchCandidate) bool {
	return len(candidates) > 0 && candidates[0].Score >= 0.9 && (len(candidates) == 1 || candidates[0].Score-candidates[1].Score >= 0.2)
}

// matchISBN is the ISBN to look a book up by: the one it carries, or else the one read out of the book itself.
func (m *multiServer) matchISBN(lib Library, book catalog.Book) string {
	for scheme, value := range book.Identifiers {
		if strings.EqualFold(scheme, "isbn") {
			if normal, ok := isbn.Normalize(value); ok {
				return normal
			}
		}
	}
	if m.isbnFound != nil {
		if result, ok := m.isbnFound.Get(lib.ID, book.ID, book.Size, book.Modified); ok && result.Own != "" {
			return result.Own
		}
	}
	return ""
}

// hardcoverKnown says whether a book is already tied to a Hardcover book, which needs no finding.
func (m *multiServer) hardcoverKnown(lib Library, book catalog.Book) bool {
	if _, linked := m.hardcoverLink(lib.ID, book.ID); linked {
		return true
	}
	return strings.TrimSpace(book.Identifiers["hardcover"]) != ""
}

// pause waits between Hardcover requests (it allows about sixty a minute), or returns false if the job was stopped.
func pause(ctx context.Context, requests int) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Duration(requests) * hardcoverTimings().requestGap):
		return true
	}
}

func titleMatch(ctx context.Context, h *enrichment.Hardcover, book catalog.Book) ([]matchCandidate, error) {
	search := enrichment.Query{Title: book.Title}
	if len(book.Authors) > 0 {
		search.Author = book.Authors[0]
	}
	results, err := h.Search(ctx, search)
	if err == nil && len(results) == 0 && search.Author != "" {
		results, err = h.Search(ctx, enrichment.Query{Title: book.Title})
	}
	if err != nil {
		return nil, err
	}
	candidates := make([]matchCandidate, 0, len(results))
	for _, result := range results {
		candidates = append(candidates, matchCandidate{Suggestion: result, Score: matchScore(book.Title, book.Authors, result)})
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	if len(candidates) > maximumStoredCandidates {
		candidates = candidates[:maximumStoredCandidates]
	}
	return candidates, nil
}

var matchJob = editorJob{
	key:         hardcoverMatchKey,
	available:   func(m *multiServer) bool { return m.matchFound != nil && m.hardcover != nil && m.moduleOn("hardcover") },
	unavailable: "Turn on the Hardcover module in Settings to find matches.",
	busy:        "Finding matches is already running for this library.",
	run: func(ctx context.Context, m *multiServer, lib Library, books []catalog.Book, opts editorJobOptions) (string, error) {
		store := m.matchFound
		total := int64(len(books))
		var done, byISBN, byTitle, unsure, none, failed int
		var stoppedBy error
		var withISBN, byName []catalog.Book
		for _, book := range books {
			if m.hardcoverKnown(lib, book) {
				continue
			}
			if have, ok := store.Get(lib.ID, book.ID, book.Size, book.Modified); ok && have.Version >= matchFindingVersion && !opts.Recount {
				continue
			}
			if m.matchISBN(lib, book) != "" {
				withISBN = append(withISBN, book)
			} else {
				byName = append(byName, book)
			}
		}
		skipped := len(books) - len(withISBN) - len(byName) // already linked or already looked up
		progress := func() {
			jobs.Report(ctx, int64(min(skipped+done+failed, len(books))), total, "Finding matches on Hardcover")
		}
		record := func(book catalog.Book, match bulkMatch) {
			match.Version = matchFindingVersion
			store.Put(lib.ID, book.ID, book.Size, book.Modified, match)
			done++
			if done%25 == 0 {
				_ = store.Save()
			}
		}
		stop := func(err error) bool {
			if errors.Is(err, enrichment.ErrProviderRateLimited) || errors.Is(err, enrichment.ErrProviderRejected) || errors.Is(err, enrichment.ErrProviderNotConnected) {
				stoppedBy = err
				return true
			}
			return false
		}
		// An ISBN names one edition, so these are exact and come twenty-five to a request.
		for from := 0; from < len(withISBN) && ctx.Err() == nil && stoppedBy == nil; from += enrichment.MaximumISBNLookup {
			batch := withISBN[from:min(from+enrichment.MaximumISBNLookup, len(withISBN))]
			wanted := make([]string, len(batch))
			for i, book := range batch {
				wanted[i] = m.matchISBN(lib, book)
			}
			progress()
			found, err := m.hardcover.LookupISBNs(ctx, wanted)
			if err != nil {
				if ctx.Err() != nil || stop(err) {
					break
				}
				failed += len(batch)
				continue
			}
			for i, book := range batch {
				if hit, ok := found[wanted[i]]; ok {
					record(book, bulkMatch{How: "isbn", ISBN: wanted[i], Clear: true, Candidates: []matchCandidate{{Suggestion: hit, Score: 1}}})
					byISBN++
				} else {
					byName = append(byName, book) // Hardcover does not know that edition: fall back to the title
				}
			}
			if !pause(ctx, 1) {
				break
			}
		}
		for _, book := range byName {
			if ctx.Err() != nil || stoppedBy != nil {
				break
			}
			progress()
			candidates, err := titleMatch(ctx, m.hardcover, book)
			if err != nil {
				if ctx.Err() != nil || stop(err) {
					break
				}
				failed++
				continue
			}
			match := bulkMatch{How: "title", Candidates: candidates, Clear: matchIsClear(candidates)}
			record(book, match)
			switch {
			case match.Clear:
				byTitle++
			case len(candidates) > 0:
				unsure++
			default:
				none++
			}
			if !pause(ctx, 2) {
				break
			}
		}
		if err := store.Save(); err != nil {
			return "", errFindingsNotSaved
		}
		message := "Looked up " + strconv.Itoa(done) + " books: " + strconv.Itoa(byISBN) + " by ISBN, " + strconv.Itoa(byTitle) + " clear by title, " + strconv.Itoa(unsure) + " unsure, " + strconv.Itoa(none) + " with no match"
		if failed > 0 {
			message += "; " + strconv.Itoa(failed) + " could not be looked up"
		}
		switch {
		case errors.Is(stoppedBy, enrichment.ErrProviderRateLimited):
			message += ". Hardcover asked us to slow down, so this stopped early; what was found is kept. Start it again later."
		case stoppedBy != nil:
			message += ". Hardcover refused the request, so this stopped early. Check the Hardcover token in Settings."
		}
		return message, nil
	},
}

// editorMatches looks books up on Hardcover in the background and keeps what it finds; nothing is applied to a book.
func (m *multiServer) editorMatches(w http.ResponseWriter, r *http.Request) {
	m.serveEditorJob(w, r, matchJob)
}

// matchNote words a stored finding for the editor.
func matchNote(match bulkMatch) string {
	if len(match.Candidates) == 0 {
		return "Hardcover has nothing for this book."
	}
	best := match.Candidates[0]
	how := "Found by its title and author"
	if match.How == "isbn" {
		how = "Found by its ISBN " + match.ISBN + ", so this is exact"
	}
	note := how + ": " + best.Title + " by " + strings.Join(best.Authors, ", ") + "."
	if match.Picked != "" {
		return "Picked by Hermes (" + match.Picked + "): " + best.Title + " by " + strings.Join(best.Authors, ", ") + "."
	}
	if !match.Clear {
		note = "Not certain. Closest by title and author: " + best.Title + " by " + strings.Join(best.Authors, ", ") + " (" + strconv.Itoa(int(best.Score*100)) + "%)."
		for _, other := range match.Candidates[1:] {
			note += " Also: " + other.Title + " (" + strconv.Itoa(int(other.Score*100)) + "%)."
		}
	}
	return note
}
