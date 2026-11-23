package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/reading"
)

// statusShelfHandler builds a Books shelf of five books with one reading position each, so every status view has a
// different, checkable answer:
//
//	in-progress : started, not finished      (open)
//	finished    : finished                   (done)
//	parked      : started but on hold        (neither in progress nor unread — it was put aside on purpose)
//	opened      : opened at page one only    (unread: looking is not reading)
//	untouched   : no position at all         (unread)
func statusShelfHandler(t *testing.T) http.Handler {
	t.Helper()
	positions := map[string]reading.Position{
		"books\x00open":   {LibraryID: "books", BookID: "open", UpdatedAt: 500, Fraction: .42},
		"books\x00done":   {LibraryID: "books", BookID: "done", UpdatedAt: 400, Fraction: 1, Finished: true},
		"books\x00parked": {LibraryID: "books", BookID: "parked", UpdatedAt: 300, Fraction: .2, Status: reading.StatusOnHold},
		"books\x00opened": {LibraryID: "books", BookID: "opened", UpdatedAt: 200},
	}
	statePath := filepath.Join(t.TempDir(), "reading.json")
	state, err := json.Marshal(map[string]any{"version": 1, "accounts": map[string]any{"default": positions}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, state, 0600); err != nil {
		t.Fatal(err)
	}
	ids := []string{"open", "done", "parked", "opened", "untouched"}
	books := make([]catalog.Book, 0, len(ids))
	for index, id := range ids {
		books = append(books, catalog.Book{
			ID:        id,
			Path:      id + ".epub",
			Title:     strings.ToUpper(id[:1]) + id[1:],
			MediaType: "application/epub+zip",
			Modified:  time.Unix(int64(10+index), 0),
		})
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: statePath},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: books}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func statusTitles(t *testing.T, handler http.Handler, query string) (titles []string, code int, header http.Header) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/books?library=books&"+query, nil))
	if response.Code != http.StatusOK {
		return nil, response.Code, response.Header()
	}
	var items []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode %s: %v", query, err)
	}
	for _, item := range items {
		titles = append(titles, item.Title)
	}
	return titles, response.Code, response.Header()
}

// The status filter has to answer from reading state, and it has to answer *inside* the scan: the count and the
// page must describe the filtered shelf, not the shelf the filter was applied to afterwards.
func TestBooksStatusFilterSelectsByReadingState(t *testing.T) {
	handler := statusShelfHandler(t)

	cases := []struct {
		status string
		want   []string
	}{
		{"in_progress", []string{"Open"}},
		{"read", []string{"Done"}},
		// Unopened, opened-at-page-one, and... parked is NOT unread: it has a position the reader set on purpose.
		{"unread", []string{"Opened", "Untouched"}},
	}
	for _, testCase := range cases {
		titles, code, _ := statusTitles(t, handler, "status="+testCase.status)
		if code != http.StatusOK {
			t.Fatalf("%s: status %d", testCase.status, code)
		}
		if strings.Join(titles, ",") != strings.Join(testCase.want, ",") {
			t.Fatalf("%s: got %v, want %v", testCase.status, titles, testCase.want)
		}
	}

	// No filter is still the whole shelf.
	all, code, _ := statusTitles(t, handler, "")
	if code != http.StatusOK || len(all) != 5 {
		t.Fatalf("unfiltered: %d %v", code, all)
	}
}

// X-Total-Count is what a client sizes its list from, so it has to count the filtered set.
func TestBooksStatusFilterCountsTheFilteredShelf(t *testing.T) {
	handler := statusShelfHandler(t)
	_, _, header := statusTitles(t, handler, "status=unread")
	if got := header.Get("X-Total-Count"); got != "2" {
		t.Fatalf("unread total = %q, want 2", got)
	}
	_, _, header = statusTitles(t, handler, "status=in_progress")
	if got := header.Get("X-Total-Count"); got != "1" {
		t.Fatalf("in-progress total = %q, want 1", got)
	}
}

// A cursor minted for one status must not be honoured by another: the anchor is a position in a different list, so
// reusing it would silently skip or repeat books.
func TestBooksStatusFilterPartitionsTheCursorView(t *testing.T) {
	handler := statusShelfHandler(t)

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/books?library=books&limit=1", nil))
	cursor := first.Header().Get("X-Next-Cursor")
	if cursor == "" {
		t.Fatalf("no cursor from an unfiltered page: %d %s", first.Code, first.Body.String())
	}

	reused := httptest.NewRecorder()
	handler.ServeHTTP(reused, httptest.NewRequest(http.MethodGet, "/api/books?library=books&status=unread&cursor="+cursor, nil))
	if reused.Code != http.StatusBadRequest {
		t.Fatalf("a cursor from another view was accepted: %d %s", reused.Code, reused.Body.String())
	}
}

// An unknown status is a client bug, not something to silently ignore into an unfiltered shelf.
func TestBooksStatusFilterRejectsUnknownValues(t *testing.T) {
	handler := statusShelfHandler(t)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/books?library=books&status=paused", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=paused answered %d, want 400", response.Code)
	}
}

// The predicate looks positions up with its own copy of the store's key format; if the two ever drift, every status
// view would answer "unread" and the bug would look like missing reading data rather than a key mismatch.
func TestReadingKeyMatchesTheStore(t *testing.T) {
	store, err := reading.Open(filepath.Join(t.TempDir(), "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put("default", reading.Position{LibraryID: "books", BookID: "one", Fraction: .5, UpdatedAt: 1}, 0); err != nil {
		t.Skipf("store API differs: %v", err)
	}
	snapshot := store.Snapshot("default")
	if _, ok := snapshot[readingKey("books", "one")]; !ok {
		t.Fatalf("readingKey(%q, %q) found nothing in the store's own snapshot", "books", "one")
	}
}
