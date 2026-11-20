package web

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"mochil/internal/enrichment"
	"mochil/internal/reading"
)

func TestARetryComesAroundLaterEachTimeAndThenStops(t *testing.T) {
	retries := newHardcoverRetries()
	finished := hardcoverJob{account: "reader", library: "books", book: "a1", event: reading.Finished}
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	for attempt, delay := range hardcoverTimings().retryDelays {
		retries.failed(finished, now)
		if got := retries.due(now.Add(delay - time.Second)); len(got) != 0 {
			t.Fatalf("attempt %d came %v early", attempt+1, time.Second)
		}
		if got := retries.due(now.Add(delay)); len(got) != 1 {
			t.Fatalf("attempt %d not due after %v", attempt+1, delay)
		}
		now = now.Add(delay)
	}
	retries.failed(finished, now)
	if len(retries.waiting) != 0 {
		t.Fatal("after the last delay the job must be given up on")
	}
}

func TestOnlyWhatWouldBeLostIsRetried(t *testing.T) {
	retries := newHardcoverRetries()
	now := time.Now()
	retries.failed(hardcoverJob{account: "reader", library: "books", book: "a1", event: reading.Reading}, now)
	if len(retries.waiting) != 0 {
		t.Fatal("the repeating reading status comes again with the next page turn")
	}
	retries.failed(hardcoverJob{account: "reader", library: "books", book: "a1", event: reading.Reading, progress: .4, isProgress: true}, now)
	retries.failed(hardcoverJob{account: "reader", library: "books", book: "a1", event: reading.Finished}, now)
	if len(retries.waiting) != 2 {
		t.Fatalf("progress and finished are kept, got %d", len(retries.waiting))
	}
}

func TestAJobThatWentThroughOrWasReplacedIsForgotten(t *testing.T) {
	retries := newHardcoverRetries()
	job := hardcoverJob{account: "reader", library: "books", book: "a1", event: reading.Reading, progress: .4, isProgress: true}
	retries.failed(job, time.Now())
	retries.settled(job)
	if len(retries.waiting) != 0 || len(retries.due(time.Now().Add(24*time.Hour))) != 0 {
		t.Fatal("a settled job must not come back")
	}
}

func TestADueJobIsNotHandedOutAgainBeforeItsResultIsKnown(t *testing.T) {
	retries := newHardcoverRetries()
	job := hardcoverJob{account: "reader", library: "books", book: "a1", event: reading.Finished}
	now := time.Now()
	retries.failed(job, now)
	if len(retries.due(now.Add(time.Hour))) != 1 || len(retries.due(now.Add(time.Hour))) != 0 {
		t.Fatal("a job being sent must not be handed out twice")
	}
}

// failingProgressHardcover is progressHardcover whose first n page-count writes fail with a server error.
func failingProgressHardcover(recorder *shelfRecorder, failures *atomic.Int32) *enrichment.Hardcover {
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
			if failures.Add(-1) >= 0 {
				return &http.Response{StatusCode: http.StatusBadGateway, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("down"))}, nil
			}
			recorder.add(body)
			response = `{"data":{"insert_user_book_read":{"error":null}}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
}

func quickHardcoverTimings(t *testing.T) {
	t.Helper()
	hardcoverTimingMu.Lock()
	saved := hardcoverTimingIs
	hardcoverTimingIs = hardcoverTiming{progressEvery: 80 * time.Millisecond, maintenanceEvery: 20 * time.Millisecond, requestGap: time.Millisecond,
		retryDelays: []time.Duration{30 * time.Millisecond, 30 * time.Millisecond, 30 * time.Millisecond}}
	hardcoverTimingMu.Unlock()
	t.Cleanup(func() {
		hardcoverTimingMu.Lock()
		hardcoverTimingIs = saved
		hardcoverTimingMu.Unlock()
	})
}

func TestTheLastPagesOfASessionReachHardcoverOnceTheThrottleAllows(t *testing.T) {
	quickHardcoverTimings(t)
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
	got := waitForShelf(t, recorder, 2)
	if !strings.Contains(got[0], `"progress_pages":100`) || !strings.Contains(got[1], `"progress_pages":108`) {
		t.Fatalf("writes = %v, want page 100 first and the last position (page 108) after the throttle", got)
	}
}

func TestAProgressUpdateThatFailedIsSentAgain(t *testing.T) {
	quickHardcoverTimings(t)
	server, _ := browseTestServer(t)
	recorder := &shelfRecorder{}
	var failures atomic.Int32
	failures.Store(2)
	server.hardcover = failingProgressHardcover(recorder, &failures)
	if err := server.startHardcoverSync(""); err != nil {
		t.Fatal(err)
	}
	if _, err := server.reading.Put("reader", reading.Position{LibraryID: "books", BookID: "a1", Fraction: .25}, 0); err != nil {
		t.Fatal(err)
	}
	got := waitForShelf(t, recorder, 1)
	if !strings.Contains(got[0], `"progress_pages":100`) {
		t.Fatalf("writes = %v", got)
	}
}
