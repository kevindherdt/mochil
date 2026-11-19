package enrichment

import (
	"context"
	"math"
)

// RatingResult says what a rating update did, so the caller can log the truth.
type RatingResult string

const (
	RatingSet       RatingResult = "rating set"
	RatingCleared   RatingResult = "rating cleared"
	RatingUnchanged RatingResult = "rating already right, left alone"
)

// SetRating puts the reader's rating, in stars (0.5 to 5; 0 takes the rating off), on a book of the connected Hardcover
// account. Only a book that is already on the shelf is touched: rating never adds a book. Hardcover keeps halves, so
// anything finer is rounded to the nearest half star.
func (h *Hardcover) SetRating(ctx context.Context, bookID string, stars float64) (RatingResult, error) {
	if atoi(bookID) <= 0 {
		return "", ErrHardcoverNoMatch
	}
	stars = math.Round(math.Max(0, math.Min(5, stars))*2) / 2
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
			ID     int64    `json:"id"`
			Rating *float64 `json:"rating"`
		} `json:"user_books"`
	}
	if err := h.graphql(ctx, `query ShelfRating($book: Int!, $user: Int!) { user_books(where: {book_id: {_eq: $book}, user_id: {_eq: $user}}, limit: 1) { id rating } }`, map[string]any{"book": atoi(bookID), "user": me.Me[0].ID}, &shelf); err != nil {
		return "", err
	}
	if len(shelf.UserBooks) == 0 {
		return "", ErrNotOnShelf
	}
	entry := shelf.UserBooks[0]
	had := entry.Rating != nil && *entry.Rating > 0
	if (stars == 0 && !had) || (had && stars > 0 && *entry.Rating == stars) {
		return RatingUnchanged, nil
	}
	var value any // nil is how Hardcover is told to take a rating off
	if stars > 0 {
		value = stars
	}
	var result struct {
		Update *struct {
			Error string `json:"error"`
		} `json:"update_user_book"`
	}
	if err := h.graphql(ctx, `mutation ChangeRating($id: Int!, $object: UserBookUpdateInput!) { update_user_book(id: $id, object: $object) { error } }`, map[string]any{"id": entry.ID, "object": map[string]any{"rating": value}}, &result); err != nil {
		return "", err
	}
	if stars == 0 {
		return RatingCleared, mutationError(result.Update)
	}
	return RatingSet, mutationError(result.Update)
}
