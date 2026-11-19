package enrichment

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// hardcoverProgress plays Hardcover for the progress tests: the reader's shelf entry for book 5 is whatever is given.
func hardcoverProgress(t *testing.T, shelf string, calls *[]string) *Hardcover {
	t.Helper()
	return NewHardcover("private-token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		response := `{"data":{}}`
		switch {
		case strings.Contains(body, "CurrentUser"):
			response = `{"data":{"me":[{"id":7}]}}`
		case strings.Contains(body, "ShelfProgress"):
			response = `{"data":{"user_books":` + shelf + `}}`
		case strings.Contains(body, "StartRead"):
			*calls = append(*calls, "start "+body)
			response = `{"data":{"insert_user_book_read":{"error":null}}}`
		case strings.Contains(body, "UpdateRead"):
			*calls = append(*calls, "update "+body)
			response = `{"data":{"update_user_book_read":{"error":null}}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
}

func TestHardcoverProgressStartsAReadWithPagesAndThenUpdatesIt(t *testing.T) {
	var calls []string
	started := `[{"id":11,"status_id":2,"book":{"pages":400},"user_book_reads":[]}]`
	result, err := hardcoverProgress(t, started, &calls).SetProgress(context.Background(), "5", .25, "2026-10-29")
	if err != nil || result != ProgressStarted || len(calls) != 1 || !strings.HasPrefix(calls[0], "start") || !strings.Contains(calls[0], `"progress_pages":100`) || !strings.Contains(calls[0], `"started_at":"2026-10-29"`) || !strings.Contains(calls[0], `"id":11`) {
		t.Fatalf("result=%q err=%v calls=%v", result, err, calls)
	}
	calls = nil
	open := `[{"id":11,"status_id":2,"book":{"pages":400},"user_book_reads":[{"id":99,"finished_at":null,"progress_pages":100}]}]`
	if result, err := hardcoverProgress(t, open, &calls).SetProgress(context.Background(), "5", .5, "2026-10-29"); err != nil || result != ProgressUpdated || len(calls) != 1 || !strings.HasPrefix(calls[0], "update") || !strings.Contains(calls[0], `"id":99`) || !strings.Contains(calls[0], `"progress_pages":200`) {
		t.Fatalf("result=%q err=%v calls=%v", result, err, calls)
	}
	calls = nil
	if result, err := hardcoverProgress(t, open, &calls).SetProgress(context.Background(), "5", .25, "2026-10-29"); err != nil || result != ProgressUnchanged || len(calls) != 0 {
		t.Fatalf("the same page was written again: result=%q err=%v calls=%v", result, err, calls)
	}
}

func TestHardcoverProgressLeavesFinishedBooksUnshelvedBooksAndUnknownPageCountsAlone(t *testing.T) {
	var calls []string
	cases := []struct {
		name  string
		shelf string
		want  error
	}{
		{"read", `[{"id":11,"status_id":3,"book":{"pages":400},"user_book_reads":[{"id":99,"finished_at":"2026-01-01","progress_pages":400}]}]`, nil},
		{"not on the shelf", `[]`, ErrNotOnShelf},
		{"no page count", `[{"id":11,"status_id":2,"book":{"pages":null},"user_book_reads":[]}]`, ErrNoPageCount},
	}
	for _, tc := range cases {
		if _, err := hardcoverProgress(t, tc.shelf, &calls).SetProgress(context.Background(), "5", .5, "2026-10-29"); !errors.Is(err, tc.want) {
			t.Fatalf("%s: err=%v, want %v", tc.name, err, tc.want)
		}
	}
	if len(calls) != 0 {
		t.Fatalf("wrote to Hardcover when it should not have: %v", calls)
	}
}

func TestHardcoverProgressStartsAFreshReadAfterAFinishedOne(t *testing.T) {
	var calls []string
	again := `[{"id":11,"status_id":2,"book":{"pages":200},"user_book_reads":[{"id":99,"finished_at":"2025-05-05","progress_pages":200}]}]`
	if result, err := hardcoverProgress(t, again, &calls).SetProgress(context.Background(), "5", 1.5, "2026-10-29"); err != nil || result != ProgressStarted || !strings.Contains(calls[0], `"progress_pages":200`) {
		t.Fatalf("a re-read starts a new read, and a fraction past the end is held at the last page: result=%q err=%v calls=%v", result, err, calls)
	}
}
