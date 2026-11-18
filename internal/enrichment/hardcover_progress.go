package enrichment

import (
	"context"
	"errors"
	"math"
)

// ProgressResult says what a progress update did, so the caller can log the truth.
type ProgressResult string

const (
	ProgressStarted   ProgressResult = "reading started with a page count"
	ProgressUpdated   ProgressResult = "page count updated"
	ProgressUnchanged ProgressResult = "page count already right, left alone"
)

// ErrNotOnShelf: the book is not on the reader's Hardcover shelf (the shelf sync adds it), so there is nothing to update.
var ErrNotOnShelf = errors.New("the book is not on the Hardcover shelf")

// ErrNoPageCount: Hardcover does not know how many pages the book has, so a fraction cannot become a page.
var ErrNoPageCount = errors.New("Hardcover has no page count for this book")

// SetProgress records how far through a book the reader is, as a page count, on the connected Hardcover account.
// Only a book that is on the shelf and not yet read is touched: a finished book keeps its history, and a book the
// shelf sync has not added yet is left for it. The latest unfinished read is updated; if there is none, one is started.
func (h *Hardcover) SetProgress(ctx context.Context, bookID string, fraction float64, today string) (ProgressResult, error) {
	if atoi(bookID) <= 0 {
		return "", ErrHardcoverNoMatch
	}
	fraction = math.Max(0, math.Min(1, fraction))
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
	var shelf struct {
		UserBooks []struct {
			ID       int64 `json:"id"`
			StatusID int   `json:"status_id"`
			Book     struct {
				Pages *int `json:"pages"`
			} `json:"book"`
			Reads []struct {
				ID            int64   `json:"id"`
				FinishedAt    *string `json:"finished_at"`
				ProgressPages *int    `json:"progress_pages"`
			} `json:"user_book_reads"`
		} `json:"user_books"`
	}
	if err := h.graphql(ctx, `query ShelfProgress($book: Int!, $user: Int!) { user_books(where: {book_id: {_eq: $book}, user_id: {_eq: $user}}, limit: 1) { id status_id book { pages } user_book_reads(order_by: {id: desc}, limit: 1) { id finished_at progress_pages } } }`, map[string]any{"book": atoi(bookID), "user": me.Me[0].ID}, &shelf); err != nil {
		return "", err
	}
	if len(shelf.UserBooks) == 0 {
		return "", ErrNotOnShelf
	}
	entry := shelf.UserBooks[0]
	if entry.StatusID == HardcoverRead {
		return ProgressUnchanged, nil
	}
	if entry.Book.Pages == nil || *entry.Book.Pages <= 0 {
		return "", ErrNoPageCount
	}
	pages := int(math.Round(fraction * float64(*entry.Book.Pages)))
	var result struct {
		Insert *struct {
			Error string `json:"error"`
		} `json:"insert_user_book_read"`
		Update *struct {
			Error string `json:"error"`
		} `json:"update_user_book_read"`
	}
	if len(entry.Reads) == 0 || entry.Reads[0].FinishedAt != nil {
		if err := h.graphql(ctx, `mutation StartRead($id: Int!, $read: DatesReadInput!) { insert_user_book_read(user_book_id: $id, user_book_read: $read) { error } }`, map[string]any{"id": entry.ID, "read": map[string]any{"progress_pages": pages, "started_at": today}}, &result); err != nil {
			return "", err
		}
		return ProgressStarted, mutationError(result.Insert)
	}
	read := entry.Reads[0]
	if read.ProgressPages != nil && *read.ProgressPages == pages {
		return ProgressUnchanged, nil
	}
	if err := h.graphql(ctx, `mutation UpdateRead($id: Int!, $read: DatesReadInput!) { update_user_book_read(id: $id, object: $read) { error } }`, map[string]any{"id": read.ID, "read": map[string]any{"progress_pages": pages}}, &result); err != nil {
		return "", err
	}
	return ProgressUpdated, mutationError(result.Update)
}
