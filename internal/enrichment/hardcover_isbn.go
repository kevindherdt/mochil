package enrichment

import (
	"context"
	"errors"
	"strings"
)

// MaximumISBNLookup is how many ISBNs LookupISBNs asks about in one request.
const MaximumISBNLookup = 25

// LookupISBNs finds the Hardcover book that each ISBN-13 belongs to, in one request. An ISBN names one edition of one
// book, so a hit is exact, unlike a search by title. ISBNs Hardcover does not know are missing from the result. It
// only reads.
func (h *Hardcover) LookupISBNs(ctx context.Context, isbns []string) (map[string]Suggestion, error) {
	if h.token == "" {
		return nil, ErrProviderNotConnected
	}
	if len(isbns) == 0 {
		return map[string]Suggestion{}, nil
	}
	if len(isbns) > MaximumISBNLookup {
		return nil, errors.New("Hardcover lookup takes a bounded list of ISBNs")
	}
	select {
	case h.gate <- struct{}{}:
		defer func() { <-h.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	var payload struct {
		Editions []struct {
			ISBN13 string        `json:"isbn_13"`
			Book   hardcoverBook `json:"book"`
		} `json:"editions"`
	}
	query := `query EditionsByISBN($isbns: [String!]!) { editions(where: {isbn_13: {_in: $isbns}}, limit: 100) { isbn_13 book { id title description release_year slug cached_image contributions { author { name } } book_series { position featured series { name } } } } }`
	if err := h.graphql(ctx, query, map[string]any{"isbns": isbns}, &payload); err != nil {
		return nil, err
	}
	found := make(map[string]Suggestion, len(payload.Editions))
	for _, edition := range payload.Editions {
		book := edition.Book
		if book.ID <= 0 || strings.TrimSpace(book.Title) == "" || edition.ISBN13 == "" {
			continue
		}
		if _, seen := found[edition.ISBN13]; seen {
			continue
		}
		found[edition.ISBN13] = hardcoverSuggestion(book)
	}
	return found, nil
}
