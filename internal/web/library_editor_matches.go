package web

import (
	"context"
	"errors"
	"math"
	"net/http"
	"regexp"
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
// Bumped to 3 when the bulk recorder started keeping the blurb. Findings written by version 2 are clear and stored,
// so without this every book looked up before would keep its empty description for ever.
const matchFindingVersion = 3

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

// editionVolume reads the volume a title claims: "Volume 06", "Vol. 5", "Tome 01", "Book 3", "Part 2".
//
// A search for "7th Time Loop - Volume 06" comes back with Tome 01 and Vol. 1 scored highest, because the words are
// nearly identical and the number is the smallest part of the string. The number is the whole question here, so it is
// read out and used for ordering.
var editionVolumePattern = regexp.MustCompile(`(?i)\b(?:vol(?:ume)?|tome|book|part)\s*\.?\s*(\d+(?:\.\d+)?)`)

func editionVolume(title string) float64 {
	matches := editionVolumePattern.FindAllStringSubmatch(title, -1)
	if len(matches) == 0 {
		return 0
	}
	// The last marker wins: "7th Time Loop - Side Stories - Volume 01.2" is volume 1.2, not volume 7.
	value, err := strconv.ParseFloat(matches[len(matches)-1][1], 64)
	if err != nil {
		return 0
	}
	return value
}

// candidateVolume is the volume a candidate says it is, from the provider's own series position when it states one and
// from its title otherwise.
func candidateVolume(candidate matchCandidate) float64 {
	if position := candidate.Suggestion.SeriesIndex; position != nil && *position > 0 {
		return *position
	}
	return editionVolume(candidate.Suggestion.Title)
}

// rankForBook puts the candidates that are the volume this book actually is ahead of the rest, then orders each group by
// score. Search ranks by how alike the words are, and every volume of a series has the same words — so the top hit for
// volume 6 was volume 1, and a person had to read the whole list to find the right one.
func rankForBook(candidates []matchCandidate, title string, position float64) {
	want := position
	if want <= 0 {
		want = editionVolume(title)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if want > 0 {
			left := math.Abs(candidateVolume(candidates[i])-want) < 0.01
			right := math.Abs(candidateVolume(candidates[j])-want) < 0.01
			if left != right {
				return left
			}
		}
		return candidates[i].Score > candidates[j].Score
	})
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
		candidates = append(candidates, matchCandidate{Suggestion: result, Overview: result.Overview, Score: matchScore(book.Title, book.Authors, result)})
	}
	position := 0.0
	if book.SeriesIndex != nil {
		position = *book.SeriesIndex
	}
	rankForBook(candidates, book.Title, position)
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
					// The blurb travels with the finding. matchCandidate has its own Overview field that shadows the
					// suggestion's, so leaving it out wrote "overview":"" to disk: the provider had sent the description
					// and every later review offered nothing for it.
					record(book, bulkMatch{How: "isbn", ISBN: wanted[i], Clear: true, Candidates: []matchCandidate{{Suggestion: hit, Overview: hit.Overview, Score: 1}}})
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
		// "Looked up 0 books" for a selection that is entirely cached reads as "found nothing", which is the opposite of
		// what happened: those books already have their match stored. Say so.
		if done == 0 && skipped == len(books) && len(books) > 0 {
			return "Nothing to look up: all " + strconv.Itoa(skipped) + " selected books already have their Hardcover match stored. " +
				"Pick different books, or re-run after changing their title, authors, series or ISBN.", nil
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
