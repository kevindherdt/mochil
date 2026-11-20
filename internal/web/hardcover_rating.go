package web

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"mochil/internal/enrichment"
)

// ratedBooks lists the books whose rating an edit touched: set, cleared, or sent back to what the files say.
func ratedBooks(apply map[string]map[string]string, resets map[string][]string, resetAll []string) []string {
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for id, fields := range apply {
		if _, ok := fields["rating"]; ok {
			add(id)
		}
	}
	for id, fields := range resets {
		for _, field := range fields {
			if field == "rating" {
				add(id)
			}
		}
	}
	for _, id := range resetAll {
		add(id)
	}
	return ids
}

// sendRatingsToHardcover queues each book's rating as it stands now. Nothing is sent unless the Hardcover shelf sync
// is running and the ratings module is on, and the worker only touches books it already knows on the shelf.
func (m *multiServer) sendRatingsToHardcover(account string, lib Library, ids []string) {
	if len(ids) == 0 || m.hardcoverEnqueue == nil || !m.moduleOn("hardcover_ratings") {
		return
	}
	for _, id := range ids {
		if book, found := lib.Store.Book(id); found {
			m.hardcoverEnqueue(hardcoverJob{account: account, library: lib.ID, book: id, isRating: true, stars: float64(book.Rating) / 2})
		}
	}
}

// sendRatingToHardcover reports whether it made a request, and how that went.
func (m *multiServer) sendRatingToHardcover(job hardcoverJob) (bool, error) {
	if !m.hardcoverAccount(job.account) || m.hardcoverIgnored.has(job.library, job.book) {
		return false, nil
	}
	id, ok := m.hardcoverLink(job.library, job.book)
	if !ok {
		match, found := m.hardcoverMatches.get(job.library, job.book)
		if !found || match.HardcoverID == "" {
			return false, nil // never searches Hardcover for a book it does not know
		}
		id = match.HardcoverID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := m.hardcover.SetRating(ctx, id, job.stars)
	switch {
	case err == nil:
		slog.Info("Hardcover rating", "book", job.book, "stars", job.stars, "result", string(result))
	case errors.Is(err, enrichment.ErrNotOnShelf):
		slog.Info("Hardcover rating skipped", "book", job.book, "reason", err.Error())
		return true, nil
	default:
		slog.Warn("Hardcover rating failed", "book", job.book, "error", err)
	}
	return true, err
}
