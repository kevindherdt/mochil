package enrichment

import (
	"context"
	"errors"
	"net/url"
	"strconv"
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
			ISBN13 string `json:"isbn_13"`
			Book   struct {
				ID            int64  `json:"id"`
				Title         string `json:"title"`
				Description   string `json:"description"`
				ReleaseYear   int    `json:"release_year"`
				Slug          string `json:"slug"`
				CachedImage   any    `json:"cached_image"`
				Contributions []struct {
					Author struct {
						Name string `json:"name"`
					} `json:"author"`
				} `json:"contributions"`
			} `json:"book"`
		} `json:"editions"`
	}
	query := `query EditionsByISBN($isbns: [String!]!) { editions(where: {isbn_13: {_in: $isbns}}, limit: 100) { isbn_13 book { id title description release_year slug cached_image contributions { author { name } } } } }`
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
		authors := make([]string, 0, len(book.Contributions))
		for _, contribution := range book.Contributions {
			if name := strings.TrimSpace(contribution.Author.Name); name != "" {
				authors = append(authors, name)
			}
		}
		slug := safeSlug(book.Slug)
		source := "https://hardcover.app/books/" + slug
		if slug == "" {
			source = "https://hardcover.app/search?query=" + url.QueryEscape(book.Title)
		}
		found[edition.ISBN13] = Suggestion{Title: book.Title, Authors: authors, Year: book.ReleaseYear, Overview: boundedText(book.Description, 2000), CoverURL: hardcoverImageURL(book.CachedImage), Provider: "Hardcover", ProviderID: strconv.FormatInt(book.ID, 10), SourceURL: source}
	}
	return found, nil
}
