package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/reading"
)

// The device's own statistics database only sees what was read on that reader, so books finished on the phone and
// reading spread over several devices have to come from the server.
func TestKoReaderStatsCountsReadingFromEveryDevice(t *testing.T) {
	server, token := browseTestServer(t)
	server.libraries[0].Store = staticBackend{books: []catalog.Book{
		{ID: "a1", Path: "Dune.epub", Title: "Dune", Authors: []string{"Frank Herbert"}, PageCount: 100, Size: 10},
		{ID: "a2", Path: "Messiah.epub", Title: "Dune Messiah", Authors: []string{"Frank Herbert"}, PageCount: 200, Size: 20},
		{ID: "n1", Path: "Neuromancer.epub", Title: "Neuromancer", Authors: []string{"William Gibson"}, PageCount: 300, Size: 30},
	}}
	now := time.Now().UTC()
	thisMonth := time.Date(now.Year(), now.Month(), 2, 12, 0, 0, 0, time.UTC).UnixMilli()
	longAgo := time.Date(now.Year()-1, time.March, 3, 12, 0, 0, 0, time.UTC).UnixMilli()
	positions := map[string]reading.Position{
		"books\x00a1": {LibraryID: "books", BookID: "a1", UpdatedAt: 100, Fraction: .5},
		"books\x00a2": {LibraryID: "books", BookID: "a2", UpdatedAt: 200, Fraction: 1, Finished: true, FinishedAt: thisMonth},
		"books\x00n1": {LibraryID: "books", BookID: "n1", UpdatedAt: 300, Fraction: 1, Finished: true, FinishedAt: longAgo},
	}
	state, err := json.Marshal(map[string]any{"version": 1, "accounts": map[string]any{"reader": positions}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "reading.json")
	if err := os.WriteFile(path, state, 0600); err != nil {
		t.Fatal(err)
	}
	if server.reading, err = reading.Open(path); err != nil {
		t.Fatal(err)
	}

	if got := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/stats", ""); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: %d", got.Code)
	}
	if got := pluginUpdateRequest(server, http.MethodPost, "/api/koreader/v1/stats", token); got.Code != http.StatusNotFound {
		t.Fatalf("POST: %d", got.Code)
	}
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/stats", token)
	if response.Code != http.StatusOK {
		t.Fatalf("stats: %d %s", response.Code, response.Body.String())
	}
	var reply koReaderStatsReply
	if err := json.Unmarshal(response.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	// Half of Dune (50) plus the two finished books (200 + 300).
	if reply.InProgress != 1 || reply.Finished != 2 || reply.FinishedThisMonth != 1 || reply.PagesRead != 550 {
		t.Fatalf("stats = %+v", reply)
	}
	if len(reply.RecentFinished) != 2 || reply.RecentFinished[0].Title == "" {
		t.Fatalf("recent = %+v", reply.RecentFinished)
	}
}

// A server with no reading store refuses the whole device API with 501 rather than answering an empty summary.
func TestKoReaderStatsWithoutReadingStore(t *testing.T) {
	server, token := browseTestServer(t)
	server.reading = nil
	if got := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/stats", token); got.Code != http.StatusNotImplemented {
		t.Fatalf("stats without a store: %d", got.Code)
	}
}

// The statistics panel sits next to the continue shelf, so its "in progress" count has to mean the
// same thing: a book only opened at page one was never read, and a book parked as on hold or
// dropped is not in progress either. Both are still counted for pages read.
func TestKoReaderStatsInProgressAgreesWithTheContinueShelf(t *testing.T) {
	server, token := browseTestServer(t)
	server.libraries[0].Store = staticBackend{books: []catalog.Book{
		{ID: "a1", Path: "Reading.epub", Title: "Reading", PageCount: 100, Size: 10},
		{ID: "a2", Path: "Opened.epub", Title: "Opened", PageCount: 100, Size: 20},
		{ID: "a3", Path: "Dropped.epub", Title: "Dropped", PageCount: 100, Size: 30},
		{ID: "a4", Path: "OnHold.epub", Title: "On hold", PageCount: 100, Size: 40},
	}}
	positions := map[string]reading.Position{
		"books\x00a1": {LibraryID: "books", BookID: "a1", UpdatedAt: 400, Fraction: .5},
		"books\x00a2": {LibraryID: "books", BookID: "a2", UpdatedAt: 300}, // opened on page one, never read
		"books\x00a3": {LibraryID: "books", BookID: "a3", UpdatedAt: 200, Fraction: .5, Status: reading.StatusDropped},
		"books\x00a4": {LibraryID: "books", BookID: "a4", UpdatedAt: 100, Fraction: .3, Status: reading.StatusOnHold},
	}
	state, err := json.Marshal(map[string]any{"version": 1, "accounts": map[string]any{"reader": positions}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "reading.json")
	if err := os.WriteFile(path, state, 0600); err != nil {
		t.Fatal(err)
	}
	if server.reading, err = reading.Open(path); err != nil {
		t.Fatal(err)
	}

	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/stats", token)
	if response.Code != http.StatusOK {
		t.Fatalf("stats: %d %s", response.Code, response.Body.String())
	}
	var reply koReaderStatsReply
	if err := json.Unmarshal(response.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.InProgress != 1 {
		t.Fatalf("in progress = %d, want 1: only the book actually being read is in progress", reply.InProgress)
	}
	// Pages read keeps counting what was read in parked books (50 + 30) and the reading one (50);
	// the book opened at page one contributes nothing.
	if reply.PagesRead != 130 {
		t.Fatalf("pages read = %d, want 130", reply.PagesRead)
	}
	if shelf := len(server.reading.ListContinue("reader", "", 0)); shelf != reply.InProgress {
		t.Fatalf("continue shelf shows %d books but stats says %d in progress", shelf, reply.InProgress)
	}
}
