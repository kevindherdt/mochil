package web

import (
	"io"
	"net/http"
	"strings"

	"mochil/internal/enrichment"
	"mochil/internal/reading"
	"testing"
	"time"
)

func TestProgressIsSentFirstThenOnlyAfterTimeAndMovement(t *testing.T) {
	start := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	last := progressSent{at: start, fraction: .30}
	for _, tc := range []struct {
		name     string
		seen     bool
		fraction float64
		after    time.Duration
		want     bool
	}{
		{"the first time", false, .01, 0, true},
		{"too soon", true, .60, 90 * time.Second, false},
		{"hardly moved", true, .305, time.Hour, false},
		{"enough of both", true, .33, 3 * time.Minute, true},
		{"read back a bit", true, .20, time.Hour, true},
	} {
		if got := progressDue(last, tc.seen, tc.fraction, start.Add(tc.after)); got != tc.want {
			t.Errorf("%s: got %v", tc.name, got)
		}
	}
}

// progressHardcover plays Hardcover for "Dune" (book 5, 400 pages) already on the shelf as currently reading.
func progressHardcover(recorder *shelfRecorder) *enrichment.Hardcover {
	return enrichment.NewHardcover("token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		response := `{"data":{}}`
		switch {
		case strings.Contains(body, "SearchBooks"):
			response = `{"data":{"search":{"ids":[5]}}}`
		case strings.Contains(body, "BookDetails"):
			response = `{"data":{"books":[{"id":5,"title":"Dune","slug":"dune","contributions":[{"author":{"name":"Frank Herbert"}}]}]}}`
		case strings.Contains(body, "CurrentUser"):
			response = `{"data":{"me":[{"id":7}]}}`
		case strings.Contains(body, "ExistingShelfEntry"):
			response = `{"data":{"user_books":[{"id":11,"status_id":2}]}}`
		case strings.Contains(body, "ShelfProgress"):
			response = `{"data":{"user_books":[{"id":11,"status_id":2,"book":{"pages":400},"user_book_reads":[]}]}}`
		case strings.Contains(body, "StartRead"):
			recorder.add(body)
			response = `{"data":{"insert_user_book_read":{"error":null}}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
}

func TestReadingProgressReachesHardcoverOnceAndNotWithEveryPageTurn(t *testing.T) {
	server, _ := browseTestServer(t)
	recorder := &shelfRecorder{}
	server.hardcover = progressHardcover(recorder)
	if err := server.startHardcoverSync(""); err != nil {
		t.Fatal(err)
	}
	var revision int64
	for _, fraction := range []float64{.25, .26, .27} {
		saved, err := server.reading.Put("reader", reading.Position{LibraryID: "books", BookID: "a1", Fraction: fraction}, revision)
		if err != nil {
			t.Fatal(err)
		}
		revision = saved.Revision
	}
	got := waitForShelf(t, recorder, 1)
	time.Sleep(300 * time.Millisecond)
	got = recorder.got()
	if len(got) != 1 || !strings.Contains(got[0], `"progress_pages":100`) || !strings.Contains(got[0], `"id":11`) {
		t.Fatalf("progress writes = %v, want one read started at page 100 of 400", got)
	}
}

func TestProgressNeverSearchesHardcoverForABookItDoesNotKnow(t *testing.T) {
	server, _ := browseTestServer(t)
	recorder := &shelfRecorder{}
	server.hardcover = progressHardcover(recorder)
	if err := server.startHardcoverSync(""); err != nil {
		t.Fatal(err)
	}
	if sent, err := server.sendProgressToHardcover(hardcoverJob{account: "reader", library: "books", book: "a1", progress: .5, isProgress: true}); sent || err != nil {
		t.Fatalf("a book with no stored match must not cost a request: sent=%v err=%v", sent, err)
	}
	if got := recorder.got(); len(got) != 0 {
		t.Fatalf("wrote %v", got)
	}
}
