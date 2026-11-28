package web

import (
	"encoding/json"
	"io"
	"mochil/internal/accounts"
	"mochil/internal/catalog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"mochil/internal/enrichment"
	"mochil/internal/reading"
)

type shelfRecorder struct {
	mu       sync.Mutex
	changes  []string
	searches int
}

func (r *shelfRecorder) add(change string) {
	r.mu.Lock()
	r.changes = append(r.changes, change)
	r.mu.Unlock()
}
func (r *shelfRecorder) got() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.changes...)
}

// fakeHardcover answers like Hardcover for one book, "Dune" by Frank Herbert, and records what is written.
func fakeHardcover(recorder *shelfRecorder) *enrichment.Hardcover {
	return enrichment.NewHardcover("token", &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(request.Body)
		body := string(raw)
		response := `{"data":{}}`
		switch {
		case strings.Contains(body, "SearchBooks"):
			recorder.mu.Lock()
			recorder.searches++
			recorder.mu.Unlock()
			response = `{"data":{"search":{"ids":[5]}}}`
		case strings.Contains(body, "BookDetails"):
			response = `{"data":{"books":[{"id":5,"title":"Dune","slug":"dune","contributions":[{"author":{"name":"Frank Herbert"}}]}]}}`
		case strings.Contains(body, "BookByID"):
			response = `{"data":{"books":[{"id":1,"title":"The Hardcover Title","slug":"the-hardcover-title","contributions":[{"author":{"name":"An Author"}}]}]}}`
		case strings.Contains(body, "CurrentUser"):
			response = `{"data":{"me":[{"id":7}]}}`
		case strings.Contains(body, "ExistingShelfEntry"):
			response = `{"data":{"user_books":[]}}`
		case strings.Contains(body, "AddToShelf"):
			recorder.add(body)
			response = `{"data":{"insert_user_book":{"error":null}}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func waitForShelf(t *testing.T, recorder *shelfRecorder, count int) []string {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if got := recorder.got(); len(got) >= count {
			return got
		}
	}
	return recorder.got()
}

func TestReadingABookReachesTheHardcoverShelfInTheBackground(t *testing.T) {
	server, _ := browseTestServer(t) // "Dune" by Frank Herbert is book a1 in library "books"
	recorder := &shelfRecorder{}
	server.hardcover = fakeHardcover(recorder)
	if err := startSync(t, server, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := server.reading.Put("reader", reading.Position{LibraryID: "books", BookID: "a1", Fraction: .1}, 0); err != nil {
		t.Fatal(err)
	}
	got := waitForShelf(t, recorder, 1)
	if len(got) != 1 || !strings.Contains(got[0], `"status":2`) || !strings.Contains(got[0], `"book":5`) {
		t.Fatalf("shelf writes = %v, want one 'currently reading' for Hardcover book 5", got)
	}
}

func TestOnlyAdministratorsReadingReachesHardcover(t *testing.T) {
	server, _ := browseTestServer(t)
	if !server.hardcoverAccount("anyone") {
		t.Fatal("a one-user server (no accounts) must send its reading")
	}
	store, err := accounts.Open(filepath.Join(t.TempDir(), "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := store.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := store.IdentityByUsername("owner")
	server.accountsStoreField.Store(store)
	recorder := &shelfRecorder{}
	server.hardcover = fakeHardcover(recorder)
	if sent, _ := server.sendToHardcover(hardcoverJob{account: reader.ID, event: reading.Reading, library: "books", book: "a1"}); sent {
		t.Fatal("a regular user's reading went to the administrator's Hardcover account")
	}
	if sent, err := server.sendToHardcover(hardcoverJob{account: owner.ID, event: reading.Reading, library: "books", book: "a1"}); !sent || err != nil {
		t.Fatal("the administrator's reading was not sent")
	}
	if got := recorder.got(); len(got) != 1 {
		t.Fatalf("shelf writes = %v, want exactly the administrator's", got)
	}
	if sent, _ := server.sendToHardcover(hardcoverJob{account: owner.ID, event: reading.Reading, library: "books", book: "missing"}); sent {
		t.Fatal("a book that is not in any library was sent")
	}
}

func TestABookAlreadyInProgressIsAnnouncedOncePerRun(t *testing.T) {
	server, _ := browseTestServer(t)
	if _, err := server.reading.Put("reader", reading.Position{LibraryID: "books", BookID: "a1", Fraction: .3}, 0); err != nil {
		t.Fatal(err) // progress saved before the module was switched on
	}
	recorder := &shelfRecorder{}
	server.hardcover = fakeHardcover(recorder)
	if err := startSync(t, server, ""); err != nil {
		t.Fatal(err)
	}
	for base := int64(1); base <= 3; base++ {
		if _, err := server.reading.Put("reader", reading.Position{LibraryID: "books", BookID: "a1", Fraction: .3 + float64(base)/10}, base); err != nil {
			t.Fatal(err)
		}
	}
	got := waitForShelf(t, recorder, 1)
	time.Sleep(1500 * time.Millisecond)
	if got = recorder.got(); len(got) != 1 {
		t.Fatalf("shelf writes = %d, want one even after three page turns", len(got))
	}
}

func TestAHardcoverBookChosenInMetadataIsUsedWithoutSearching(t *testing.T) {
	server, _ := browseTestServer(t)
	overrides, err := catalog.OpenOverrideStore(filepath.Join(t.TempDir(), "overrides.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := overrides.SetArtwork("books", "a1", "https://assets.hardcover.app/dune.jpg", "Hardcover", "777"); err != nil {
		t.Fatal(err)
	}
	server.metadataOverrides = overrides
	recorder := &shelfRecorder{}
	server.hardcover = fakeHardcover(recorder)
	if sent, err := server.sendToHardcover(hardcoverJob{account: "anyone", event: reading.Reading, library: "books", book: "a1"}); !sent || err != nil {
		t.Fatalf("sent=%v err=%v", sent, err)
	}
	if got := recorder.got(); len(got) != 1 || !strings.Contains(got[0], `"book":777`) || recorder.searches != 0 {
		t.Fatalf("writes=%v searches=%d, want the linked book 777 and no search", got, recorder.searches)
	}
}

func hardcoverRequest(server *multiServer, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	response := httptest.NewRecorder()
	server.apiHardcover(response, request)
	return response
}

// startSync starts the worker the way a test needs it: retired again when the test ends.
//
// The worker writes links, skips and matches into the directory it was given. A test that leaves it running gets
// those writes after it returns, and then t.TempDir's cleanup fails with "directory not empty" — which is how this
// suite failed in CI while passing locally, and why every call site goes through here.
func startSync(t *testing.T, server *multiServer, path string) error {
	t.Helper()
	err := server.startHardcoverSync(path)
	if err == nil {
		t.Cleanup(server.stopHardcoverSync)
	}
	return err
}

func TestASkippedBookCanBeLinkedByHandAndThenSyncs(t *testing.T) {
	server, _ := browseTestServer(t) // "Neuromancer" (n1) is not what the fake Hardcover knows ("Dune")
	if err := startSync(t, server, filepath.Join(t.TempDir(), "links.json")); err != nil {
		t.Fatal(err)
	}
	recorder := &shelfRecorder{}
	server.hardcover = fakeHardcover(recorder)
	account := server.readingAccount(httptest.NewRequest(http.MethodGet, "/", nil))
	if _, err := server.reading.Put(account, reading.Position{LibraryID: "books", BookID: "n1", Fraction: .3}, 0); err != nil {
		t.Fatal(err)
	}
	var skipped struct {
		Enabled bool          `json:"enabled"`
		Items   []skippedBook `json:"items"`
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		_ = json.Unmarshal(hardcoverRequest(server, http.MethodGet, "/api/metadata/hardcover/skipped", "").Body.Bytes(), &skipped)
		if len(skipped.Items) > 0 {
			break
		}
	}
	if !skipped.Enabled || len(skipped.Items) != 1 || skipped.Items[0].Title != "Neuromancer" || skipped.Items[0].Book != "n1" {
		t.Fatalf("skipped = %+v, want Neuromancer listed", skipped)
	}
	if got := recorder.got(); len(got) != 0 {
		t.Fatalf("an unmatched book was written to Hardcover: %v", got)
	}
	recorder.mu.Lock()
	searchesBeforeLinking := recorder.searches
	recorder.mu.Unlock()
	response := hardcoverRequest(server, http.MethodPost, "/api/metadata/hardcover/link", `{"library":"books","book":"n1","provider_id":"313448"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("link = %d %s", response.Code, response.Body.String())
	}
	got := waitForShelf(t, recorder, 1)
	if len(got) != 1 || !strings.Contains(got[0], `"book":313448`) || !strings.Contains(got[0], `"status":2`) || recorder.searches != searchesBeforeLinking {
		t.Fatalf("shelf writes = %v searches=%d, want Neuromancer's linked book 313448 as currently reading", got, recorder.searches)
	}
	_ = json.Unmarshal(hardcoverRequest(server, http.MethodGet, "/api/metadata/hardcover/skipped", "").Body.Bytes(), &skipped)
	if len(skipped.Items) != 0 {
		t.Fatalf("a linked book is still listed as skipped: %+v", skipped.Items)
	}
	if id, ok := server.hardcoverLink("books", "n1"); !ok || id != "313448" {
		t.Fatalf("link = %q %v", id, ok)
	}
}

func TestHardcoverLinksSurviveARestartAndRefuseBadInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "links.json")
	server, _ := browseTestServer(t)
	server.hardcover = fakeHardcover(&shelfRecorder{})
	if err := startSync(t, server, path); err != nil {
		t.Fatal(err)
	}
	if response := hardcoverRequest(server, http.MethodPost, "/api/metadata/hardcover/link", `{"library":"books","book":"a1","provider_id":"42"}`); response.Code != http.StatusOK {
		t.Fatalf("link = %d", response.Code)
	}
	reopened, err := openHardcoverLinks(path)
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := reopened.get("books", "a1"); !ok || id != "42" {
		t.Fatalf("after a restart: %q %v", id, ok)
	}
	for _, body := range []string{
		`{"library":"books","book":"a1","provider_id":"4x2"}`,
		`{"library":"books","book":"a1","provider_id":""}`,
		`{"library":"books","book":"missing","provider_id":"42"}`,
		`{"library":"nope","book":"a1","provider_id":"42"}`,
	} {
		if response := hardcoverRequest(server, http.MethodPost, "/api/metadata/hardcover/link", body); response.Code != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", body, response.Code)
		}
	}
	if response := hardcoverRequest(server, http.MethodGet, "/api/metadata/hardcover/link", ""); response.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET link = %d", response.Code)
	}
}

func TestHardcoverLinkingNeedsTheModule(t *testing.T) {
	server, _ := browseTestServer(t) // the module was never started: no enqueue function, no link store
	if response := hardcoverRequest(server, http.MethodPost, "/api/metadata/hardcover/link", `{"library":"books","book":"a1","provider_id":"42"}`); response.Code != http.StatusBadRequest {
		t.Fatalf("link with the module off = %d", response.Code)
	}
	var skipped struct{ Enabled bool }
	_ = json.Unmarshal(hardcoverRequest(server, http.MethodGet, "/api/metadata/hardcover/skipped", "").Body.Bytes(), &skipped)
	if skipped.Enabled {
		t.Fatal("the skipped list claims the module is on")
	}
}

func TestTheSkippedListSurvivesARestart(t *testing.T) {
	dir := t.TempDir()
	first, _ := browseTestServer(t)
	if err := startSync(t, first, filepath.Join(dir, "hardcover-links.json")); err != nil {
		t.Fatal(err)
	}
	first.hardcoverSkipped.add(skippedBook{Library: "books", Book: "n1", Title: "Neuromancer", Authors: []string{"William Gibson"}, At: 1})
	first.hardcoverSkipped.add(skippedBook{Library: "books", Book: "a1", Title: "Dune", At: 2})
	first.hardcoverSkipped.remove("books", "a1")
	// saving a setting rebuilds the server: a new one starts from the same files
	second, _ := browseTestServer(t)
	if err := startSync(t, second, filepath.Join(dir, "hardcover-links.json")); err != nil {
		t.Fatal(err)
	}
	if items := second.hardcoverSkipped.list(); len(items) != 1 || items[0].Title != "Neuromancer" {
		t.Fatalf("after a restart the list is %+v, want just Neuromancer", items)
	}
}

func TestEachSyncedBookKeepsARecordOfWhatItMatchedTo(t *testing.T) {
	dir := t.TempDir()
	server, _ := browseTestServer(t)
	if err := startSync(t, server, filepath.Join(dir, "hardcover-links.json")); err != nil {
		t.Fatal(err)
	}
	recorder := &shelfRecorder{}
	server.hardcover = fakeHardcover(recorder)
	if sent, err := server.sendToHardcover(hardcoverJob{account: "anyone", event: reading.Reading, library: "books", book: "a1"}); !sent || err != nil {
		t.Fatalf("sent=%v err=%v", sent, err)
	}
	match, ok := server.hardcoverMatches.get("books", "a1")
	if !ok || match.How != "matched" || match.HardcoverID != "5" || match.HardcoverTitle != "Dune" || match.Title != "Dune" || match.URL != "https://hardcover.app/books/dune" || match.Result != "added to the shelf" {
		t.Fatalf("match = %+v, want Dune matched by name to Hardcover book 5", match)
	}
	var reply struct {
		Enabled bool            `json:"enabled"`
		Match   *hardcoverMatch `json:"match"`
		Skipped bool            `json:"skipped"`
	}
	response := hardcoverRequest(server, http.MethodGet, "/api/metadata/hardcover/match?library=books&book=a1", "")
	if err := json.Unmarshal(response.Body.Bytes(), &reply); err != nil || !reply.Enabled || reply.Match == nil || reply.Match.HardcoverTitle != "Dune" || reply.Skipped {
		t.Fatalf("match endpoint = %d %s", response.Code, response.Body.String())
	}
	if response := hardcoverRequest(server, http.MethodGet, "/api/metadata/hardcover/match?library=books&book=missing", ""); response.Code != http.StatusNotFound {
		t.Fatalf("an unknown book = %d", response.Code)
	}
	// a restart keeps the records
	second, _ := browseTestServer(t)
	if err := startSync(t, second, filepath.Join(dir, "hardcover-links.json")); err != nil {
		t.Fatal(err)
	}
	var list struct {
		Items []hardcoverMatch `json:"items"`
	}
	_ = json.Unmarshal(hardcoverRequest(second, http.MethodGet, "/api/metadata/hardcover/matches", "").Body.Bytes(), &list)
	if len(list.Items) != 1 || list.Items[0].HardcoverTitle != "Dune" {
		t.Fatalf("after a restart: %+v", list.Items)
	}
}

func TestLinkingByHandShowsWhatWasChosenAndChangesTheMatch(t *testing.T) {
	server, _ := browseTestServer(t)
	if err := startSync(t, server, ""); err != nil {
		t.Fatal(err)
	}
	recorder := &shelfRecorder{}
	server.hardcover = fakeHardcover(recorder)
	if sent, err := server.sendToHardcover(hardcoverJob{account: "anyone", event: reading.Reading, library: "books", book: "a1"}); !sent || err != nil {
		t.Fatalf("sent=%v err=%v", sent, err)
	}
	if response := hardcoverRequest(server, http.MethodPost, "/api/metadata/hardcover/link", `{"library":"books","book":"a1","provider_id":"999"}`); response.Code != http.StatusOK {
		t.Fatalf("link = %d %s", response.Code, response.Body.String())
	}
	match, _ := server.hardcoverMatches.get("books", "a1")
	if match.How != "linked" || match.HardcoverID != "999" || match.HardcoverTitle != "The Hardcover Title" {
		t.Fatalf("after linking: %+v, want the chosen book shown, not the old guess", match)
	}
	// the next send uses the link, and keeps the title that was saved with it
	if _, err := server.sendToHardcover(hardcoverJob{account: "anyone", event: reading.Finished, library: "books", book: "a1"}); err != nil {
		t.Fatal(err)
	}
	match, _ = server.hardcoverMatches.get("books", "a1")
	if match.How != "linked" || match.HardcoverID != "999" || match.HardcoverTitle != "The Hardcover Title" || len(server.hardcoverMatches.list()) != 1 {
		t.Fatalf("after syncing: %+v (one record per book)", match)
	}
}

func TestALinkMadeBeforeRecordsWereKeptStillShowsItsMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hardcover-links.json")
	if err := os.WriteFile(path, []byte(`{"links":{"books\u0000a1":"1643933"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	server, _ := browseTestServer(t)
	server.hardcover = fakeHardcover(&shelfRecorder{})
	if err := startSync(t, server, path); err != nil {
		t.Fatal(err)
	}
	var reply struct {
		Match *hardcoverMatch `json:"match"`
	}
	response := hardcoverRequest(server, http.MethodGet, "/api/metadata/hardcover/match?library=books&book=a1", "")
	if err := json.Unmarshal(response.Body.Bytes(), &reply); err != nil || reply.Match == nil || reply.Match.How != "linked" || reply.Match.HardcoverID != "1643933" || reply.Match.HardcoverTitle != "The Hardcover Title" {
		t.Fatalf("match = %s", response.Body.String())
	}
}

func TestDoubtIsWordedOnlyForAmbiguousMatches(t *testing.T) {
	if doubtOf(enrichment.ShelfOutcome{}) != "" {
		t.Fatal("a clear match was flagged")
	}
	if doubtOf(enrichment.ShelfOutcome{Ambiguous: true}) == "" {
		t.Fatal("an ambiguous match was not flagged")
	}
}

func TestPreviewShowsWhatABookWouldMatchWithoutSendingAnything(t *testing.T) {
	server, _ := browseTestServer(t)
	recorder := &shelfRecorder{}
	server.hardcover = fakeHardcover(recorder)
	var reply struct {
		Match *struct {
			Title      string `json:"title"`
			ProviderID string `json:"provider_id"`
		} `json:"match"`
		Linked    bool `json:"linked"`
		Ambiguous bool `json:"ambiguous"`
	}
	response := hardcoverRequest(server, http.MethodGet, "/api/metadata/hardcover/preview?library=books&book=a1", "")
	if err := json.Unmarshal(response.Body.Bytes(), &reply); err != nil || reply.Match == nil || reply.Match.Title != "Dune" || reply.Match.ProviderID != "5" || reply.Linked {
		t.Fatalf("preview = %d %s", response.Code, response.Body.String())
	}
	// a book Hardcover does not know: no match, but the closest results are offered
	response = hardcoverRequest(server, http.MethodGet, "/api/metadata/hardcover/preview?library=books&book=n1", "")
	var none struct {
		Match      any   `json:"match"`
		Candidates []any `json:"candidates"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &none); err != nil || none.Match != nil || len(none.Candidates) == 0 {
		t.Fatalf("no-match preview = %d %s", response.Code, response.Body.String())
	}
	if got := recorder.got(); len(got) != 0 {
		t.Fatalf("previewing wrote to Hardcover: %v", got)
	}
	if response := hardcoverRequest(server, http.MethodGet, "/api/metadata/hardcover/preview?library=books&book=missing", ""); response.Code != http.StatusNotFound {
		t.Fatalf("unknown book = %d", response.Code)
	}
}
