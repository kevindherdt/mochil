package enrichment

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// Hardcover's reading statuses. Only these two are ever sent: want-to-read, paused and the rest stay the reader's own.
const (
	HardcoverReading = 2
	HardcoverRead    = 3
)

// ShelfResult says what a status change did, so the caller can log the truth.
type ShelfResult string

const (
	ShelfAdded     ShelfResult = "added to the shelf"
	ShelfChanged   ShelfResult = "status changed"
	ShelfUnchanged ShelfResult = "already right, left alone"
	// ShelfAbsent: the book is not on the shelf and adding it was not allowed, because the reader removed it there.
	ShelfAbsent ShelfResult = "not on the shelf, not added"
)

// ShelfOutcome is what SetStatus did and which Hardcover book it did it to, so the match can be shown and checked.
type ShelfOutcome struct {
	Result ShelfResult
	Book   Suggestion
	// Ambiguous: more than one Hardcover book fits the title and author equally well, and the top-ranked one was taken.
	Ambiguous bool
}

// ErrHardcoverNoMatch means Hardcover has no book that is clearly this one. Nothing is written in that case:
// marking the wrong book as read is worse than skipping one.
var ErrHardcoverNoMatch = errors.New("Hardcover has no matching book")

// ErrHardcoverAmbiguous means several Hardcover books fit the title and author equally well. Writing the top-ranked one
// would put a stranger's book on the reader's shelf -- "It" resolved to a cookbook once -- so the caller is told to let a
// person pick instead. Guessing here edits state outside this server.
var ErrHardcoverAmbiguous = errors.New("several Hardcover books match this title and author")

// SetStatus marks a book as currently reading or read on the connected Hardcover account.
// It never moves a finished book back to reading, and does nothing when the status is already right.
func (h *Hardcover) SetStatus(ctx context.Context, title string, authors []string, status int) (ShelfOutcome, error) {
	return h.SetStatusAllowing(ctx, title, authors, status, true)
}

// SetStatusAllowing is SetStatus where adding a book that is not on the shelf can be refused (ShelfAbsent), for
// books that were on it before and have since been removed.
func (h *Hardcover) SetStatusAllowing(ctx context.Context, title string, authors []string, status int, allowAdd bool) (ShelfOutcome, error) {
	if status != HardcoverReading && status != HardcoverRead {
		return ShelfOutcome{}, fmt.Errorf("unsupported Hardcover status %d", status)
	}
	book, ambiguous, err := h.findBook(ctx, title, authors)
	if err != nil {
		return ShelfOutcome{}, err
	}
	if ambiguous {
		return ShelfOutcome{Book: book, Ambiguous: true}, ErrHardcoverAmbiguous
	}
	result, err := h.SetStatusByIDAllowing(ctx, book.ProviderID, status, allowAdd)
	return ShelfOutcome{Result: result, Book: book}, err
}

// findBookID looks for the one Hardcover book that is clearly this one. The title alone is searched first: adding
// the author to the search text lets bundles and study guides crowd the real book out of the first results.
func (h *Hardcover) findBook(ctx context.Context, title string, authors []string) (Suggestion, bool, error) {
	author := ""
	if len(authors) > 0 {
		author = authors[0]
	}
	queries := []Query{{Title: title}}
	if author != "" {
		queries = append(queries, Query{Title: title, Author: author})
	}
	for _, query := range queries {
		suggestions, err := h.Search(ctx, query)
		if err != nil {
			return Suggestion{}, false, err
		}
		var fits []Suggestion
		for _, suggestion := range suggestions {
			if sameTitle(suggestion.Title, title) && sharesAuthor(suggestion.Authors, authors) {
				fits = append(fits, suggestion)
			}
		}
		if len(fits) > 0 {
			return fits[0], len(fits) > 1, nil
		}
	}
	return Suggestion{}, false, ErrHardcoverNoMatch
}

// Match finds the Hardcover book a title and authors would be matched to, without changing anything on Hardcover.
func (h *Hardcover) Match(ctx context.Context, title string, authors []string) (book Suggestion, ambiguous bool, err error) {
	return h.findBook(ctx, title, authors)
}

// BookByID describes one Hardcover book, for showing the reader what a link points to.
func (h *Hardcover) BookByID(ctx context.Context, id string) (Suggestion, error) {
	if h.token == "" {
		return Suggestion{}, ErrProviderNotConnected
	}
	if atoi(id) <= 0 {
		return Suggestion{}, ErrHardcoverNoMatch
	}
	var details struct {
		Books []struct {
			ID            int64  `json:"id"`
			Title         string `json:"title"`
			ReleaseYear   int    `json:"release_year"`
			Slug          string `json:"slug"`
			Contributions []struct {
				Author struct {
					Name string `json:"name"`
				} `json:"author"`
			} `json:"contributions"`
		} `json:"books"`
	}
	if err := h.graphql(ctx, `query BookByID($id: Int!) { books(where: {id: {_eq: $id}}, limit: 1) { id title release_year slug contributions { author { name } } } }`, map[string]any{"id": atoi(id)}, &details); err != nil {
		return Suggestion{}, err
	}
	if len(details.Books) != 1 {
		return Suggestion{}, ErrHardcoverNoMatch
	}
	book := details.Books[0]
	authors := make([]string, 0, len(book.Contributions))
	for _, contribution := range book.Contributions {
		if name := strings.TrimSpace(contribution.Author.Name); name != "" {
			authors = append(authors, name)
		}
	}
	sourceURL := "https://hardcover.app/books/" + safeSlug(book.Slug)
	if safeSlug(book.Slug) == "" {
		sourceURL = "https://hardcover.app/search?query=" + url.QueryEscape(book.Title)
	}
	return Suggestion{Title: book.Title, Authors: authors, Year: book.ReleaseYear, Provider: "Hardcover", ProviderID: id, SourceURL: sourceURL}, nil
}

// SetStatusByID is SetStatus for a book whose Hardcover id is already known, for instance one the reader chose
// in the Metadata suggestions. It never moves a finished book back to reading.
func (h *Hardcover) SetStatusByID(ctx context.Context, bookID string, status int) (ShelfResult, error) {
	return h.SetStatusByIDAllowing(ctx, bookID, status, true)
}

// SetStatusByIDAllowing is SetStatusByID where adding a missing shelf entry can be refused (ShelfAbsent).
func (h *Hardcover) SetStatusByIDAllowing(ctx context.Context, bookID string, status int, allowAdd bool) (ShelfResult, error) {
	if status != HardcoverReading && status != HardcoverRead {
		return "", fmt.Errorf("unsupported Hardcover status %d", status)
	}
	if atoi(bookID) <= 0 {
		return "", ErrHardcoverNoMatch
	}
	var me struct {
		Me []struct {
			ID int64 `json:"id"`
		} `json:"me"`
	}
	if err := h.graphql(ctx, `query CurrentUser { me(limit: 1) { id } }`, nil, &me); err != nil {
		return "", err
	}
	if len(me.Me) != 1 || me.Me[0].ID <= 0 {
		return "", ErrProviderRejected
	}
	var existing struct {
		UserBooks []struct {
			ID       int64 `json:"id"`
			StatusID int   `json:"status_id"`
		} `json:"user_books"`
	}
	if err := h.graphql(ctx, `query ExistingShelfEntry($book: Int!, $user: Int!) { user_books(where: {book_id: {_eq: $book}, user_id: {_eq: $user}}, limit: 1) { id status_id } }`, map[string]any{"book": atoi(bookID), "user": me.Me[0].ID}, &existing); err != nil {
		return "", err
	}
	var result struct {
		Insert *struct {
			Error string `json:"error"`
		} `json:"insert_user_book"`
		Update *struct {
			Error string `json:"error"`
		} `json:"update_user_book"`
	}
	if len(existing.UserBooks) == 0 && !allowAdd {
		return ShelfAbsent, nil
	}
	if len(existing.UserBooks) == 0 {
		if err := h.graphql(ctx, `mutation AddToShelf($book: Int!, $status: Int!) { insert_user_book(object: {book_id: $book, status_id: $status}) { error } }`, map[string]any{"book": atoi(bookID), "status": status}, &result); err != nil {
			return "", err
		}
		return ShelfAdded, mutationError(result.Insert)
	}
	current := existing.UserBooks[0]
	if current.StatusID == status || (status == HardcoverReading && current.StatusID == HardcoverRead) {
		return ShelfUnchanged, nil
	}
	if err := h.graphql(ctx, `mutation ChangeShelfEntry($id: Int!, $status: Int!) { update_user_book(id: $id, object: {status_id: $status}) { error } }`, map[string]any{"id": current.ID, "status": status}, &result); err != nil {
		return "", err
	}
	return ShelfChanged, mutationError(result.Update)
}

func mutationError(result *struct {
	Error string `json:"error"`
}) error {
	if result != nil && strings.TrimSpace(result.Error) != "" {
		return fmt.Errorf("Hardcover refused the change: %s", result.Error)
	}
	return nil
}

func atoi(value string) int64 {
	var n int64
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0
		}
		n = n*10 + int64(digit-'0')
	}
	return n
}

var (
	volumePart = regexp.MustCompile(`(?i)\bvol(?:ume|\.)?\s*0*(\d+)\.(\d+)\b`)
	volumeOnly = regexp.MustCompile(`(?i)\bvol(?:ume|\.)?\s*0*(\d+)\b`)
)

// words lower-cases and drops punctuation so "Dune: Messiah" and "dune messiah" compare equal.
func words(text string) string {
	// Light-novel files say "Volume 01.2" where Hardcover says "Volume 1 Part 2": compare them alike.
	text = volumePart.ReplaceAllString(text, "volume $1 part $2")
	text = volumeOnly.ReplaceAllString(text, "volume $1")
	var out strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			out.WriteRune(r)
		} else if unicode.IsSpace(r) || r == ':' || r == '-' {
			out.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

func sameTitle(a, b string) bool { return words(a) != "" && words(a) == words(b) }

// sharesAuthor accepts a book with no known author, otherwise needs one author in common.
func sharesAuthor(found, wanted []string) bool {
	if len(wanted) == 0 {
		return true
	}
	for _, f := range found {
		for _, w := range wanted {
			if words(f) != "" && words(f) == words(w) {
				return true
			}
		}
	}
	return false
}
