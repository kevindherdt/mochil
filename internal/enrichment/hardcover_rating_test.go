package enrichment

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// hardcoverRating plays Hardcover for the rating tests: the reader's shelf entry for book 5 is whatever is given.
func hardcoverRating(t *testing.T, shelf string, calls *[]string) *Hardcover {
	t.Helper()
	return NewHardcover("private-token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		response := `{"data":{}}`
		switch {
		case strings.Contains(body, "CurrentUser"):
			response = `{"data":{"me":[{"id":7}]}}`
		case strings.Contains(body, "ShelfRating"):
			response = `{"data":{"user_books":` + shelf + `}}`
		case strings.Contains(body, "ChangeRating"):
			*calls = append(*calls, body)
			response = `{"data":{"update_user_book":{"error":null}}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
}

func TestHardcoverRatingIsSetChangedClearedAndNeverRepeated(t *testing.T) {
	var calls []string
	unrated := `[{"id":11,"rating":null}]`
	if result, err := hardcoverRating(t, unrated, &calls).SetRating(context.Background(), "5", 4.5); err != nil || result != RatingSet || len(calls) != 1 || !strings.Contains(calls[0], `"rating":4.5`) || !strings.Contains(calls[0], `"id":11`) {
		t.Fatalf("result=%q err=%v calls=%v", result, err, calls)
	}
	calls = nil
	rated := `[{"id":11,"rating":4.5}]`
	if result, err := hardcoverRating(t, rated, &calls).SetRating(context.Background(), "5", 4.5); err != nil || result != RatingUnchanged || len(calls) != 0 {
		t.Fatalf("the same rating was written again: result=%q err=%v calls=%v", result, err, calls)
	}
	if result, err := hardcoverRating(t, rated, &calls).SetRating(context.Background(), "5", 3.8); err != nil || result != RatingSet || !strings.Contains(calls[0], `"rating":4`) {
		t.Fatalf("3.8 stars should go as the nearest half (4): result=%q err=%v calls=%v", result, err, calls)
	}
	calls = nil
	if result, err := hardcoverRating(t, rated, &calls).SetRating(context.Background(), "5", 0); err != nil || result != RatingCleared || !strings.Contains(calls[0], `"rating":null`) {
		t.Fatalf("clearing must send null: result=%q err=%v calls=%v", result, err, calls)
	}
	calls = nil
	if result, err := hardcoverRating(t, unrated, &calls).SetRating(context.Background(), "5", 0); err != nil || result != RatingUnchanged || len(calls) != 0 {
		t.Fatalf("clearing nothing wrote: result=%q err=%v calls=%v", result, err, calls)
	}
}

func TestHardcoverRatingNeverAddsABookToTheShelf(t *testing.T) {
	var calls []string
	if _, err := hardcoverRating(t, `[]`, &calls).SetRating(context.Background(), "5", 5); err != ErrNotOnShelf || len(calls) != 0 {
		t.Fatalf("err=%v calls=%v, want ErrNotOnShelf and no writes", err, calls)
	}
	if _, err := hardcoverRating(t, `[]`, &calls).SetRating(context.Background(), "", 5); err != ErrHardcoverNoMatch {
		t.Fatalf("err=%v, want ErrHardcoverNoMatch for an unknown book", err)
	}
}
