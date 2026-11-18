package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mochil/internal/catalog"
)

// summaryBackend serves the cheap accessors and counts any full-catalogue copy,
// so a test can prove a feed did not take one.
type summaryBackend struct {
	books  []catalog.Book
	count  int
	latest time.Time
	reads  int
	snaps  int
}

func (b *summaryBackend) Scan() (catalog.Stats, error) { return catalog.Stats{Books: b.count}, nil }
func (b *summaryBackend) Books() []catalog.Book {
	b.reads++
	return append([]catalog.Book(nil), b.books...)
}
func (b *summaryBackend) Snapshot() []catalog.Book {
	b.snaps++
	return b.books
}
func (b *summaryBackend) Summary() (int, time.Time) { return b.count, b.latest }
func (b *summaryBackend) Book(id string) (catalog.Book, bool) {
	for _, book := range b.books {
		if book.ID == id {
			return book, true
		}
	}
	return catalog.Book{}, false
}
func (b *summaryBackend) Acquire(context.Context, catalog.Book) (catalog.Acquisition, error) {
	return catalog.Acquisition{}, nil
}

func TestOPDSRootFeedUsesTheCatalogueSummaryInsteadOfCopyingBooks(t *testing.T) {
	latest := time.Date(2026, 11, 2, 10, 15, 0, 0, time.UTC)
	backend := &summaryBackend{count: 4321, latest: latest}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{
		{ID: "books", Name: "Books", Kind: "ebooks", Store: backend},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/opds", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("root feed status = %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "4321 items") {
		t.Fatalf("root feed lost the library count: %s", body)
	}
	if !strings.Contains(body, latest.Format(time.RFC3339)) {
		t.Fatalf("root feed lost the catalogue timestamp: %s", body)
	}
	if backend.reads != 0 {
		t.Fatalf("the root feed copied the whole catalogue %d times", backend.reads)
	}
}

func TestBrowseFeedReadsThePublishedSnapshotWithoutCopyingBooks(t *testing.T) {
	backend := &summaryBackend{books: []catalog.Book{
		{ID: "one", Path: "One.epub", Title: "One", MediaType: "application/epub+zip"},
	}, count: 1}
	handler, err := New(Config{}, backend, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/opds/titles?bucket=O", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), ">One<") {
		t.Fatalf("titles feed = %d %s", response.Code, response.Body.String())
	}
	if backend.reads != 0 {
		t.Fatalf("the titles feed copied the catalogue %d times", backend.reads)
	}
	if backend.snaps == 0 {
		t.Fatal("the titles feed did not use the published snapshot")
	}
}
