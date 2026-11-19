package web

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"mochil/internal/accounts"
	"mochil/internal/catalog"
	"mochil/internal/enrichment"
	"mochil/internal/metadata"
	"mochil/internal/reading"
	"mochil/internal/video"
)

type staticBackend struct{ books []catalog.Book }

type versionedStaticBackend struct {
	staticBackend
	version uint64
	reads   int
}

func (s *versionedStaticBackend) CatalogVersion() uint64 { return s.version }
func (s *versionedStaticBackend) Books() []catalog.Book {
	s.reads++
	return s.staticBackend.Books()
}

func TestRequestIdentityIsolatesReadingStateBetweenLocalAccounts(t *testing.T) {
	directory := t.TempDir()
	readingPath := filepath.Join(directory, "reading.json")
	accountStore, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := accountStore.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	positions, err := reading.Open(readingPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = positions.Put("owner", reading.Position{LibraryID: "books", BookID: "owner-book", CFI: "epubcfi(/6/2)", Fraction: .2}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = positions.Put(reader.ID, reading.Position{LibraryID: "books", BookID: "reader-book", CFI: "epubcfi(/6/4)", Fraction: .7}, 0); err != nil {
		t.Fatal(err)
	}
	if err := positions.Flush(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{Username: "owner", Password: "owner password", ReadingState: readingPath}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{{ID: "owner-book", Path: "Owner.epub", Title: "Owner book"}, {ID: "reader-book", Path: "Reader.epub", Title: "Reader book"}}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()

	request := loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"reader password"}, "next": {"/api/reading"}})
	login := httptest.NewRecorder()
	handler.ServeHTTP(login, request)
	if login.Code != http.StatusSeeOther {
		t.Fatalf("reader login=%d %s", login.Code, login.Body.String())
	}
	readingRequest := httptest.NewRequest(http.MethodGet, "/api/reading", nil)
	readingRequest.AddCookie(login.Result().Cookies()[0])
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, readingRequest)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Reader book") || strings.Contains(response.Body.String(), "Owner book") {
		t.Fatalf("isolated reading response=%d %s", response.Code, response.Body.String())
	}
}

type staticPublication struct{ *bytes.Reader }

func (staticPublication) Close() error { return nil }

type metadataRoundTrip func(*http.Request) (*http.Response, error)

func (f metadataRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func (s staticBackend) Scan() (catalog.Stats, error) { return catalog.Stats{Books: len(s.books)}, nil }
func (s staticBackend) Books() []catalog.Book        { return append([]catalog.Book(nil), s.books...) }
func (s staticBackend) Book(id string) (catalog.Book, bool) {
	for _, book := range s.books {
		if book.ID == id {
			return book, true
		}
	}
	return catalog.Book{}, false
}

type hydratingBackend struct{ staticBackend }

func (hydratingBackend) MetadataHydrating() bool { return true }
func (s staticBackend) Acquire(_ context.Context, book catalog.Book) (catalog.Acquisition, error) {
	return catalog.Acquisition{Reader: staticPublication{bytes.NewReader([]byte(book.Title))}}, nil
}

type versionedTestBackend struct {
	mu      sync.RWMutex
	books   []catalog.Book
	version uint64
}

func (s *versionedTestBackend) Scan() (catalog.Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return catalog.Stats{Books: len(s.books)}, nil
}
func (s *versionedTestBackend) Books() []catalog.Book {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]catalog.Book(nil), s.books...)
}
func (s *versionedTestBackend) Book(id string) (catalog.Book, bool) {
	for _, book := range s.Books() {
		if book.ID == id {
			return book, true
		}
	}
	return catalog.Book{}, false
}
func (s *versionedTestBackend) Acquire(context.Context, catalog.Book) (catalog.Acquisition, error) {
	return catalog.Acquisition{}, nil
}
func (s *versionedTestBackend) CatalogVersion() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}
func (s *versionedTestBackend) replace(books []catalog.Book) {
	s.mu.Lock()
	s.books = append([]catalog.Book(nil), books...)
	s.version++
	s.mu.Unlock()
}

func TestMultiLibraryRootAndRouting(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var libraries []Library
	for _, id := range []string{"books", "manga"} {
		dir := t.TempDir()
		store := catalog.NewStore(dir, filepath.Join(t.TempDir(), id+".json"))
		if _, err := store.Scan(); err != nil {
			t.Fatal(err)
		}
		libraries = append(libraries, Library{ID: id, Name: strings.ToUpper(id), Kind: "books", Store: store})
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, libraries, logger)
	if err != nil {
		t.Fatal(err)
	}

	rootRequest := httptest.NewRequest("GET", "/opds", nil)
	rootResponse := httptest.NewRecorder()
	handler.ServeHTTP(rootResponse, rootRequest)
	if rootResponse.Code != 200 || !strings.Contains(rootResponse.Body.String(), "/opds/libraries/books") {
		t.Fatalf("unexpected root response: %d %s", rootResponse.Code, rootResponse.Body.String())
	}

	libraryRequest := httptest.NewRequest("GET", "/opds/libraries/manga/series", nil)
	libraryResponse := httptest.NewRecorder()
	handler.ServeHTTP(libraryResponse, libraryRequest)
	if libraryResponse.Code != 200 {
		t.Fatalf("library route returned %d", libraryResponse.Code)
	}
}

func TestMetadataIssuesAPI(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{
		ID: "books", Name: "Books", Kind: "books",
		Store: staticBackend{books: []catalog.Book{{ID: "one", Title: "One", Authors: []string{"Unknown"}}}},
	}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/issues?library=books", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"missing_author"`) {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
}

func TestMetadataSuggestionsRequireOptInAndReturnProvenance(t *testing.T) {
	libraries := []Library{{ID: "books", Name: "Books", Kind: "books", MetadataProvider: "openlibrary", Store: staticBackend{books: []catalog.Book{{ID: "one", Title: "A Book", Authors: []string{"Unknown"}}}}}}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, libraries, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	managed := handler.(*managedHandler)
	server := managed.Handler.(*multiServer)
	server.openLibrary = enrichment.NewOpenLibrary(&http.Client{Transport: metadataRoundTrip(func(request *http.Request) (*http.Response, error) {
		body := `{"docs":[{"key":"/works/OL1W","title":"A Book","author_name":["Writer"]}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/suggestions?library=books&book=one", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"provider":"Open Library"`) || !strings.Contains(response.Body.String(), `"shared":{"author":"","title":"A Book"}`) || !strings.Contains(response.Body.String(), `"Writer"`) {
		t.Fatalf("unexpected suggestion response: %d %s", response.Code, response.Body.String())
	}

	disabled, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "private", Name: "Private", Kind: "books", Store: libraries[0].Store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	denied := httptest.NewRecorder()
	disabled.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/api/metadata/suggestions?library=private&book=one", nil))
	if denied.Code != http.StatusConflict {
		t.Fatalf("disabled lookup status=%d", denied.Code)
	}
}

func TestHardcoverSuggestionsUseWriteOnlyServerToken(t *testing.T) {
	library := Library{ID: "manga", Name: "Manga", Kind: "manga", MetadataProvider: "hardcover", Store: staticBackend{books: []catalog.Book{{ID: "one", Title: "Dune", Authors: []string{"Frank Herbert"}}}}}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, HardcoverToken: "private-token"}, []Library{library}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	server := handler.(*managedHandler).Handler.(*multiServer)
	server.hardcover = enrichment.NewHardcover("private-token", &http.Client{Transport: metadataRoundTrip(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer private-token" {
			t.Fatal("Hardcover token was not kept in the server request")
		}
		body, _ := io.ReadAll(request.Body)
		response := `{"data":{"search":{"ids":[1]}}}`
		if strings.Contains(string(body), "BookDetails") {
			response = `{"data":{"books":[{"id":1,"title":"Dune","release_year":1965,"slug":"dune","contributions":[{"author":{"name":"Frank Herbert"}}]}]}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/suggestions?library=manga&book=one", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"provider":"Hardcover"`) || !strings.Contains(response.Body.String(), `"year":1965`) || strings.Contains(response.Body.String(), "private-token") {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}

func TestProviderConnectionReportsUsageWithoutLeakingToken(t *testing.T) {
	libraries := []Library{{ID: "books", Name: "My Books", Kind: "books", MetadataProvider: "hardcover", Store: staticBackend{books: []catalog.Book{{ID: "one", Title: "Dune"}}}}}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, HardcoverToken: "private-token"}, libraries, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	server := handler.(*managedHandler).Handler.(*multiServer)
	server.hardcover = enrichment.NewHardcover("private-token", &http.Client{Transport: metadataRoundTrip(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer private-token" {
			t.Fatal("token was not sent only to Hardcover")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"data":{"me":[{"id":1}]}}`))}, nil
	})})
	request := httptest.NewRequest(http.MethodPost, "/api/metadata/provider-connection", strings.NewReader(`{"provider":"hardcover"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"connected":true`) || !strings.Contains(response.Body.String(), `"name":"My Books"`) || strings.Contains(response.Body.String(), "private-token") {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}

// A LAN install has no public_url. Signing a phone in must still work and must hand back the address
// the request arrived on, because the app stores that address for every later call. This used to
// answer 501, which the app reports as "Mochil is temporarily unavailable. Try again."
func TestAndroidLoginWithoutPublicURLUsesTheRequestAddress(t *testing.T) {
	handler, err := NewMulti(MultiConfig{
		Username:     "reader",
		Password:     "private-password",
		ReadingState: filepath.Join(t.TempDir(), "reading.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/android/v1/login", strings.NewReader(`{"device_name":"Pixel 9"}`))
	request.Host = "192.168.1.20:8099"
	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth("reader", "private-password")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("lan login = %d %s, want 201 with a device token", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"server_url":"http://192.168.1.20:8099"`) {
		t.Fatalf("server_url must be the address the app used, got %s", response.Body.String())
	}
}

func TestAndroidLoginExchangesBasicCredentialsForDeviceToken(t *testing.T) {
	handler, err := NewMulti(MultiConfig{
		Username:     "reader",
		Password:     "private-password",
		PublicURL:    "https://books.example.com",
		ReadingState: filepath.Join(t.TempDir(), "reading.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/android/v1/login", strings.NewReader(`{"device_name":"Pixel 9"}`))
	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth("reader", "private-password")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"device_token"`) || strings.Contains(response.Body.String(), "private-password") {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}

	denied := httptest.NewRecorder()
	badRequest := httptest.NewRequest(http.MethodPost, "/api/android/v1/login", strings.NewReader(`{"device_name":"Pixel 9"}`))
	badRequest.Header.Set("Content-Type", "application/json")
	badRequest.SetBasicAuth("reader", "wrong")
	handler.ServeHTTP(denied, badRequest)
	if denied.Code != http.StatusUnauthorized || strings.Contains(denied.Body.String(), "private-password") {
		t.Fatalf("denied=%d %s", denied.Code, denied.Body.String())
	}
}

func TestAndroidTVPairingRequiresBrowserApprovalAndIssuesOneToken(t *testing.T) {
	handler, err := NewMulti(MultiConfig{
		Username: "reader", Password: "private-password", PublicURL: "https://books.example.com",
		ReadingState: filepath.Join(t.TempDir(), "reading.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	create := httptest.NewRequest(http.MethodPost, "/api/android/v1/pairing", strings.NewReader(`{"device_name":"Living room TV"}`))
	create.Header.Set("Content-Type", "application/json")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	var pairing struct {
		ID   string `json:"pairing_id"`
		Code string `json:"user_code"`
	}
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &pairing) != nil || len(pairing.Code) != 6 || pairing.ID == "" {
		t.Fatalf("create=%d %s", created.Code, created.Body.String())
	}
	pending := httptest.NewRecorder()
	handler.ServeHTTP(pending, httptest.NewRequest(http.MethodGet, "/api/android/v1/pairing/"+pairing.ID, nil))
	if pending.Code != http.StatusAccepted {
		t.Fatalf("pending=%d %s", pending.Code, pending.Body.String())
	}

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/pair?code="+pairing.Code, nil))
	if unauthorized.Code != http.StatusSeeOther || !strings.HasPrefix(unauthorized.Header().Get("Location"), "/login") {
		t.Fatalf("unauthorized=%d %q", unauthorized.Code, unauthorized.Header().Get("Location"))
	}
	login := loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"private-password"}, "next": {"/pair?code=" + pairing.Code}})
	loggedIn := httptest.NewRecorder()
	handler.ServeHTTP(loggedIn, login)
	if loggedIn.Code != http.StatusSeeOther || len(loggedIn.Result().Cookies()) == 0 {
		t.Fatalf("login=%d", loggedIn.Code)
	}
	cookie := loggedIn.Result().Cookies()[0]
	approvalPage := httptest.NewRequest(http.MethodGet, "/pair?code="+pairing.Code, nil)
	approvalPage.AddCookie(cookie)
	approvalResponse := httptest.NewRecorder()
	handler.ServeHTTP(approvalResponse, approvalPage)
	match := regexp.MustCompile(`name="approval" value="([^"]+)"`).FindStringSubmatch(approvalResponse.Body.String())
	if approvalResponse.Code != http.StatusOK || len(match) != 2 {
		t.Fatalf("approval page=%d %s", approvalResponse.Code, approvalResponse.Body.String())
	}
	form := url.Values{"code": {pairing.Code}, "approval": {match[1]}}
	approve := httptest.NewRequest(http.MethodPost, "/pair", strings.NewReader(form.Encode()))
	approve.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approve.AddCookie(cookie)
	approved := httptest.NewRecorder()
	handler.ServeHTTP(approved, approve)
	if approved.Code != http.StatusOK || !strings.Contains(approved.Body.String(), "TV connected") {
		t.Fatalf("approve=%d %s", approved.Code, approved.Body.String())
	}

	exchange := httptest.NewRecorder()
	handler.ServeHTTP(exchange, httptest.NewRequest(http.MethodGet, "/api/android/v1/pairing/"+pairing.ID, nil))
	if exchange.Code != http.StatusCreated || !strings.Contains(exchange.Body.String(), `"device_token"`) {
		t.Fatalf("exchange=%d %s", exchange.Code, exchange.Body.String())
	}
	replay := httptest.NewRecorder()
	handler.ServeHTTP(replay, httptest.NewRequest(http.MethodGet, "/api/android/v1/pairing/"+pairing.ID, nil))
	if replay.Code != http.StatusGone {
		t.Fatalf("replay=%d %s", replay.Code, replay.Body.String())
	}
}

func TestAndroidDeviceTokenProtectsNativeCatalogue(t *testing.T) {
	handler, err := NewMulti(MultiConfig{
		Username: "reader", Password: "private-password", PublicURL: "https://books.example.com",
		ReadingState:        filepath.Join(t.TempDir(), "reading.json"),
		AudioQueueState:     filepath.Join(t.TempDir(), "audio-queue.json"),
		AudioPlaylistsState: filepath.Join(t.TempDir(), "audio-playlists.json"),
		CollectionsState:    filepath.Join(t.TempDir(), "collections.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{{ID: "one", Title: "Dune", Series: "Dune Saga"}}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	login := httptest.NewRequest(http.MethodPost, "/api/android/v1/login", strings.NewReader(`{"device_name":"Pixel 9"}`))
	login.Header.Set("Content-Type", "application/json")
	login.SetBasicAuth("reader", "private-password")
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, login)
	var credential struct {
		Token string `json:"device_token"`
	}
	if loginResponse.Code != http.StatusCreated || json.Unmarshal(loginResponse.Body.Bytes(), &credential) != nil || credential.Token == "" {
		t.Fatalf("login=%d %s", loginResponse.Code, loginResponse.Body.String())
	}

	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/api/android/v1/libraries", nil))
	if denied.Code != http.StatusUnauthorized || denied.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("uncredentialed request=%d headers=%v", denied.Code, denied.Header())
	}

	for path, expected := range map[string]string{
		"/api/android/v1/libraries":                     `"Books"`,
		"/api/android/v1/books?library=books&limit=60":  `"Dune"`,
		"/api/android/v1/series?library=books&limit=60": `"Dune Saga"`,
		"/api/android/v1/publications/books/one":        `Dune`,
		"/api/android/v1/audio-queue":                   `[]`,
		"/api/android/v1/audio-playlists":               `[]`,
		"/api/android/v1/collections?library=books":     `[]`,
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer "+credential.Token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("%s=%d %s", path, response.Code, response.Body.String())
		}
	}
	androidRequest := func(method, path, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+credential.Token)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	createdCollection := androidRequest(http.MethodPost, "/api/android/v1/collections", `{"name":"To read","library_id":"books"}`)
	if createdCollection.Code != http.StatusCreated {
		t.Fatalf("create Android collection=%d %s", createdCollection.Code, createdCollection.Body.String())
	}
	var collection savedCollection
	if err := json.Unmarshal(createdCollection.Body.Bytes(), &collection); err != nil || collection.ID == "" {
		t.Fatalf("decode Android collection: %v %s", err, createdCollection.Body.String())
	}
	member := androidRequest(http.MethodPost, "/api/android/v1/collections/"+collection.ID+"/members", `{"type":"series","library_id":"books","series":"Dune Saga"}`)
	if member.Code != http.StatusNoContent {
		t.Fatalf("add Android series member=%d %s", member.Code, member.Body.String())
	}
	collectionList := androidRequest(http.MethodGet, "/api/android/v1/collections?library=books", "")
	if collectionList.Code != http.StatusOK || !strings.Contains(collectionList.Body.String(), `"count":1`) {
		t.Fatalf("Android collection summary=%d %s", collectionList.Code, collectionList.Body.String())
	}
	collectionBooks := androidRequest(http.MethodGet, "/api/android/v1/collections/"+collection.ID+"/books", "")
	if collectionBooks.Code != http.StatusOK || !strings.Contains(collectionBooks.Body.String(), `"title":"Dune"`) {
		t.Fatalf("Android collection books=%d %s", collectionBooks.Code, collectionBooks.Body.String())
	}
	diagnostics := httptest.NewRequest(http.MethodPost, "/api/android/v1/diagnostics", strings.NewReader(`{"app_version":"0.11.0","android":"16","device":"Test phone","events":[{"at":1,"kind":"network","name":"load_books","duration_ms":42,"outcome":"ok"}]}`))
	diagnostics.Header.Set("Authorization", "Bearer "+credential.Token)
	diagnostics.Header.Set("Content-Type", "application/json")
	diagnosticsResponse := httptest.NewRecorder()
	handler.ServeHTTP(diagnosticsResponse, diagnostics)
	if diagnosticsResponse.Code != http.StatusNoContent {
		t.Fatalf("diagnostics=%d %s", diagnosticsResponse.Code, diagnosticsResponse.Body.String())
	}
	home := androidRequest(http.MethodGet, "/api/android/v1/home", "")
	if home.Code != http.StatusOK || !strings.Contains(home.Body.String(), `"reading":`) || !strings.Contains(home.Body.String(), `"listening":`) || !strings.Contains(home.Body.String(), `"watching":`) {
		t.Fatalf("Android home=%d %s", home.Code, home.Body.String())
	}
	progressPut := httptest.NewRequest(http.MethodPut, "/api/android/v1/reading/books/one", strings.NewReader(`{"fraction":0.25,"base_revision":0,"device_id":"pixel","device_name":"Mochil Android","koreader_locator":"/body/DocFragment[4]/body/div[1]/p[3]","locators":{"readium":"{\"href\":\"chapter1\"}"}}`))
	progressPut.Header.Set("Authorization", "Bearer "+credential.Token)
	progressPut.Header.Set("Content-Type", "application/json")
	progressResponse := httptest.NewRecorder()
	handler.ServeHTTP(progressResponse, progressPut)
	if progressResponse.Code != http.StatusOK || !strings.Contains(progressResponse.Body.String(), `"readium"`) || !strings.Contains(progressResponse.Body.String(), `"koreader_locator":"/body/DocFragment[4]/body/div[1]/p[3]"`) {
		t.Fatalf("progress put=%d %s", progressResponse.Code, progressResponse.Body.String())
	}
	oversizedDeviceName := strings.Repeat("x", reading.MaximumDeviceNameBytes+1)
	invalidProgressPut := httptest.NewRequest(http.MethodPut, "/api/android/v1/reading/books/one", strings.NewReader(`{"fraction":0.25,"base_revision":1,"device_id":"pixel","device_name":"`+oversizedDeviceName+`"}`))
	invalidProgressPut.Header.Set("Authorization", "Bearer "+credential.Token)
	invalidProgressPut.Header.Set("Content-Type", "application/json")
	invalidProgressResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidProgressResponse, invalidProgressPut)
	if invalidProgressResponse.Code != http.StatusBadRequest {
		t.Fatalf("oversized progress device name=%d %s", invalidProgressResponse.Code, invalidProgressResponse.Body.String())
	}
	progressGet := httptest.NewRequest(http.MethodGet, "/api/android/v1/reading/books/one", nil)
	progressGet.Header.Set("Authorization", "Bearer "+credential.Token)
	progressGetResponse := httptest.NewRecorder()
	handler.ServeHTTP(progressGetResponse, progressGet)
	if progressGetResponse.Code != http.StatusOK || !strings.Contains(progressGetResponse.Body.String(), `"fraction":0.25`) {
		t.Fatalf("progress get=%d %s", progressGetResponse.Code, progressGetResponse.Body.String())
	}
	progressList := httptest.NewRequest(http.MethodGet, "/api/android/v1/reading?limit=12", nil)
	progressList.Header.Set("Authorization", "Bearer "+credential.Token)
	progressListResponse := httptest.NewRecorder()
	handler.ServeHTTP(progressListResponse, progressList)
	if progressListResponse.Code != http.StatusOK || !strings.Contains(progressListResponse.Body.String(), `"title":"Dune"`) || !strings.Contains(progressListResponse.Body.String(), `"fraction":0.25`) {
		t.Fatalf("progress list=%d %s", progressListResponse.Code, progressListResponse.Body.String())
	}
	annotationPut := httptest.NewRequest(http.MethodPut, "/api/android/v1/annotations/books/one", strings.NewReader(`{"changes":[{"base_revision":0,"annotation":{"id":"mark-1","kind":"highlight","quote":"Fear is the mind-killer","device_id":"pixel"}}]}`))
	annotationPut.Header.Set("Authorization", "Bearer "+credential.Token)
	annotationPut.Header.Set("Content-Type", "application/json")
	annotationPutResponse := httptest.NewRecorder()
	handler.ServeHTTP(annotationPutResponse, annotationPut)
	if annotationPutResponse.Code != http.StatusOK || !strings.Contains(annotationPutResponse.Body.String(), `"revision":1`) {
		t.Fatalf("annotation put=%d %s", annotationPutResponse.Code, annotationPutResponse.Body.String())
	}
	annotationGet := httptest.NewRequest(http.MethodGet, "/api/android/v1/annotations/books/one?since=0", nil)
	annotationGet.Header.Set("Authorization", "Bearer "+credential.Token)
	annotationGetResponse := httptest.NewRecorder()
	handler.ServeHTTP(annotationGetResponse, annotationGet)
	if annotationGetResponse.Code != http.StatusOK || !strings.Contains(annotationGetResponse.Body.String(), `"id":"mark-1"`) {
		t.Fatalf("annotation get=%d %s", annotationGetResponse.Code, annotationGetResponse.Body.String())
	}
	annotationNotes := httptest.NewRequest(http.MethodPost, "/api/android/v1/annotations/books/one/notes", strings.NewReader(`{}`))
	annotationNotes.Header.Set("Authorization", "Bearer "+credential.Token)
	annotationNotes.Header.Set("Content-Type", "application/json")
	annotationNotesResponse := httptest.NewRecorder()
	handler.ServeHTTP(annotationNotesResponse, annotationNotes)
	if annotationNotesResponse.Code != http.StatusCreated || !strings.Contains(annotationNotesResponse.Body.String(), `"title":"Annotations — Dune"`) {
		t.Fatalf("annotation notes=%d %s", annotationNotesResponse.Code, annotationNotesResponse.Body.String())
	}
	annotationExport := httptest.NewRequest(http.MethodGet, "/api/android/v1/annotations/export", nil)
	annotationExport.Header.Set("Authorization", "Bearer "+credential.Token)
	annotationExportResponse := httptest.NewRecorder()
	handler.ServeHTTP(annotationExportResponse, annotationExport)
	if annotationExportResponse.Code != http.StatusOK || !strings.Contains(annotationExportResponse.Header().Get("Content-Disposition"), "mochil-annotations.json") || !strings.Contains(annotationExportResponse.Body.String(), `"id": "mark-1"`) {
		t.Fatalf("annotation export=%d %s", annotationExportResponse.Code, annotationExportResponse.Body.String())
	}
	annotationDelete := httptest.NewRequest(http.MethodDelete, "/api/android/v1/annotations/account", strings.NewReader(`{"confirmation":"DELETE ALL ANNOTATIONS"}`))
	annotationDelete.Header.Set("Authorization", "Bearer "+credential.Token)
	annotationDelete.Header.Set("Content-Type", "application/json")
	annotationDeleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(annotationDeleteResponse, annotationDelete)
	if annotationDeleteResponse.Code != http.StatusOK || !strings.Contains(annotationDeleteResponse.Body.String(), `"deleted":1`) {
		t.Fatalf("annotation delete=%d %s", annotationDeleteResponse.Code, annotationDeleteResponse.Body.String())
	}
}

func TestHardcoverPublicationCoverOverrideIsValidatedAndReversible(t *testing.T) {
	state := filepath.Join(t.TempDir(), "reviews.json")
	library := Library{ID: "books", Name: "Books", Kind: "books", MetadataProvider: "hardcover", Store: staticBackend{books: []catalog.Book{{ID: "one", Title: "Dune"}}}}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, MetadataReviewState: state}, []Library{library}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	put := httptest.NewRequest(http.MethodPut, "/api/publication-metadata/books/one", strings.NewReader(`{"artwork_url":"https://assets.hardcover.app/dune.jpg","provider":"Hardcover","provider_id":"1"}`))
	put.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, put)
	if response.Code != http.StatusOK {
		t.Fatalf("save response=%d %s", response.Code, response.Body.String())
	}
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/publication-metadata/books/one", nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), `"configured":true`) || !strings.Contains(get.Body.String(), `"provider_id":"1"`) {
		t.Fatalf("get response=%d %s", get.Code, get.Body.String())
	}
	unsafe := httptest.NewRequest(http.MethodPut, "/api/publication-metadata/books/one", strings.NewReader(`{"artwork_url":"https://example.com/dune.jpg","provider":"Hardcover"}`))
	unsafe.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, unsafe)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unsafe cover status=%d", response.Code)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/publication-metadata/books/one", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("delete response=%d %s", response.Code, response.Body.String())
	}
}

func TestOpenLibraryPublicationCoverOverrideIsValidatedAndProviderBound(t *testing.T) {
	state := filepath.Join(t.TempDir(), "reviews.json")
	library := Library{ID: "books", Name: "Books", Kind: "books", MetadataProvider: "openlibrary", Store: staticBackend{books: []catalog.Book{{ID: "one", Title: "Dune"}}}}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, MetadataReviewState: state}, []Library{library}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()

	put := httptest.NewRequest(http.MethodPut, "/api/publication-metadata/books/one", strings.NewReader(`{"artwork_url":"https://covers.openlibrary.org/b/id/12345-M.jpg?default=false","provider":"Open Library","provider_id":"OL1W"}`))
	put.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, put)
	if response.Code != http.StatusOK {
		t.Fatalf("save response=%d %s", response.Code, response.Body.String())
	}

	wrongProvider := httptest.NewRequest(http.MethodPut, "/api/publication-metadata/books/one", strings.NewReader(`{"artwork_url":"https://assets.hardcover.app/dune.jpg","provider":"Hardcover","provider_id":"1"}`))
	wrongProvider.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, wrongProvider)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("cross-provider cover status=%d body=%s", response.Code, response.Body.String())
	}

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/publication-metadata/books/one", nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), `"provider":"Open Library"`) || !strings.Contains(get.Body.String(), `"provider_id":"OL1W"`) {
		t.Fatalf("get response=%d %s", get.Code, get.Body.String())
	}
}

func TestTMDBSuggestionsUseVideoIdentityAndKeepTokenOffResponse(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Arrival (2016).mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("movies", root, filepath.Join(t.TempDir(), "video.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, TMDBToken: "private-token"}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", MetadataProvider: "tmdb", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	server := handler.(*managedHandler).Handler.(*multiServer)
	server.tmdb = enrichment.NewTMDB("private-token", &http.Client{Transport: metadataRoundTrip(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer private-token" {
			t.Fatal("TMDB token was not sent in the authorization header")
		}
		body := `{"results":[{"id":329865,"title":"Arrival","release_date":"2016-11-10","overview":"A linguist investigates.","poster_path":"/arrival.jpg"}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})})
	item := store.Items()[0]
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/suggestions?library=movies&video="+item.ID, nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"provider":"TMDB"`) || !strings.Contains(response.Body.String(), `"year":2016`) || strings.Contains(response.Body.String(), "private-token") {
		t.Fatalf("unexpected TMDB response: %d %s", response.Code, response.Body.String())
	}
}

func TestVideoMetadataOverrideIsReviewableAndAppearsInCatalogue(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Original.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("movies", root, filepath.Join(t.TempDir(), "video.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, VideoMetadataState: filepath.Join(t.TempDir(), "metadata.json")}, []Library{{ID: "movies", Name: "Movies", Kind: "movies", MetadataProvider: "tmdb", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	body := `{"title":"Arrival","year":2016,"overview":"A linguist investigates.","provider":"TMDB","provider_id":"329865"}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/video-metadata/movies/"+item.ID, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("save response=%d %s", response.Code, response.Body.String())
	}
	catalogue := httptest.NewRecorder()
	handler.ServeHTTP(catalogue, httptest.NewRequest(http.MethodGet, "/api/videos?library=movies", nil))
	if catalogue.Code != http.StatusOK || !strings.Contains(catalogue.Body.String(), `"title":"Arrival"`) || !strings.Contains(catalogue.Body.String(), `"year":2016`) || !strings.Contains(catalogue.Body.String(), `"metadata_override"`) {
		t.Fatalf("catalogue response=%d %s", catalogue.Code, catalogue.Body.String())
	}
	removed := httptest.NewRecorder()
	handler.ServeHTTP(removed, httptest.NewRequest(http.MethodDelete, "/api/video-metadata/movies/"+item.ID, nil))
	if removed.Code != http.StatusOK {
		t.Fatalf("remove response=%d %s", removed.Code, removed.Body.String())
	}
	restored := httptest.NewRecorder()
	handler.ServeHTTP(restored, httptest.NewRequest(http.MethodGet, "/api/videos?library=movies", nil))
	if restored.Code != http.StatusOK || !strings.Contains(restored.Body.String(), `"title":"Original"`) || strings.Contains(restored.Body.String(), `"metadata_override"`) {
		t.Fatalf("restored response=%d %s", restored.Code, restored.Body.String())
	}
}

func TestShowMetadataOverridePreservesSourceGrouping(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Original Show"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Original Show", "S01E01 Pilot.mp4"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := video.NewStore("tv", root, filepath.Join(t.TempDir(), "video.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, VideoState: filepath.Join(t.TempDir(), "progress.json"), VideoMetadataState: filepath.Join(t.TempDir(), "metadata.json")}, []Library{{ID: "tv", Name: "TV", Kind: "tv", MetadataProvider: "tmdb", Video: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	progress := httptest.NewRecorder()
	progressRequest := httptest.NewRequest(http.MethodPut, "/api/video-progress/tv/"+url.PathEscape(item.ID), strings.NewReader(`{"position_seconds":100,"duration_seconds":100,"finished":true}`))
	progressRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(progress, progressRequest)
	if progress.Code != http.StatusOK {
		t.Fatalf("progress response=%d %s", progress.Code, progress.Body.String())
	}
	body := `{"title":"Official Show","year":2024,"overview":"Official synopsis.","provider":"TMDB","provider_id":"42"}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/video-show-metadata/tv?show="+url.QueryEscape("Original Show"), strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("save response=%d %s", response.Code, response.Body.String())
	}
	shows := httptest.NewRecorder()
	handler.ServeHTTP(shows, httptest.NewRequest(http.MethodGet, "/api/video-shows?library=tv", nil))
	if shows.Code != http.StatusOK || !strings.Contains(shows.Body.String(), `"name":"Official Show"`) || !strings.Contains(shows.Body.String(), `"source_name":"Original Show"`) || !strings.Contains(shows.Body.String(), `"year":2024`) || !strings.Contains(shows.Body.String(), `"watched_count":1`) {
		t.Fatalf("shows response=%d %s", shows.Code, shows.Body.String())
	}
	episodes := httptest.NewRecorder()
	handler.ServeHTTP(episodes, httptest.NewRequest(http.MethodGet, "/api/videos?library=tv&show="+url.QueryEscape("Original Show"), nil))
	if episodes.Code != http.StatusOK || !strings.Contains(episodes.Body.String(), `"show":"Original Show"`) {
		t.Fatalf("episode grouping changed: %d %s", episodes.Code, episodes.Body.String())
	}
}

func TestMetadataInspectionWaitsForRemoteHydration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	state := filepath.Join(t.TempDir(), "metadata-reviews.json")
	handler, err := NewMulti(MultiConfig{AllowOpen: true, MetadataReviewState: state}, []Library{{
		ID: "books", Name: "Books", Kind: "books",
		Store: hydratingBackend{staticBackend{books: []catalog.Book{{ID: "one", Title: "One", Authors: []string{"Unknown"}}}}},
	}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/metadata/issues?library=books", nil),
		httptest.NewRequest(http.MethodPost, "/api/metadata/jobs?library=books", nil),
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "still loading") {
			t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
		}
	}
}

func TestMetadataReviewJobsAPI(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	state := filepath.Join(t.TempDir(), "metadata-reviews.json")
	handler, err := NewMulti(MultiConfig{AllowOpen: true, MetadataReviewState: state}, []Library{{
		ID: "books", Name: "Books", Kind: "books", SourceType: "filesystem", MetadataWrites: true,
		Store: staticBackend{books: []catalog.Book{{ID: "one", Title: "One", Authors: []string{"Unknown"}}}},
	}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/metadata/jobs?library=books", nil))
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"missing_author"`) {
		t.Fatalf("unexpected create response: %d %s", created.Code, created.Body.String())
	}
	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, httptest.NewRequest(http.MethodGet, "/api/metadata/jobs?library=books", nil))
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"issue_count":1`) {
		t.Fatalf("unexpected list response: %d %s", listed.Code, listed.Body.String())
	}
	var job struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	decided := httptest.NewRecorder()
	decisionURL := "/api/metadata/jobs/" + job.ID + "/items/one/missing_author"
	handler.ServeHTTP(decided, httptest.NewRequest(http.MethodPut, decisionURL, strings.NewReader(`{"decision":"accepted","value":"Writer"}`)))
	if decided.Code != http.StatusOK || !strings.Contains(decided.Body.String(), `"status":"resolved"`) {
		t.Fatalf("unexpected decision response: %d %s", decided.Code, decided.Body.String())
	}
	preview := httptest.NewRecorder()
	handler.ServeHTTP(preview, httptest.NewRequest(http.MethodGet, "/api/metadata/jobs/"+job.ID+"/preview", nil))
	if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(), `"mode":"mixed"`) {
		t.Fatalf("unexpected preview response: %d %s", preview.Code, preview.Body.String())
	}
}

func TestIngestPreviewIsReadOnlyBoundedAndLocal(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{
		ID: "books", Name: "Books", Kind: "books", SourceType: "filesystem",
		Store: staticBackend{books: []catalog.Book{{ID: "one", Path: "Ursula K. Le Guin/Earthsea/Earthsea - Vol 01 - A Wizard of Earthsea [isbn=9780547773742].epub", Authors: []string{"Unknown"}}}},
	}, {ID: "remote", Name: "Remote", Kind: "books", SourceType: "readonly", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/ingest-preview?library=books&limit=1", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"inspected":1`) || !strings.Contains(response.Body.String(), `"proposed":"Ursula K. Le Guin"`) || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected preview: %d %s", response.Code, response.Body.String())
	}
	remote := httptest.NewRecorder()
	handler.ServeHTTP(remote, httptest.NewRequest(http.MethodGet, "/api/metadata/ingest-preview?library=remote", nil))
	if remote.Code != http.StatusConflict {
		t.Fatalf("remote preview=%d %s", remote.Code, remote.Body.String())
	}
	write := httptest.NewRecorder()
	handler.ServeHTTP(write, httptest.NewRequest(http.MethodPost, "/api/metadata/ingest-preview?library=books", nil))
	if write.Code != http.StatusMethodNotAllowed {
		t.Fatalf("write preview=%d", write.Code)
	}
}

func TestIngestPreviewReusesOnePlanPerCatalogueGeneration(t *testing.T) {
	backend := &versionedStaticBackend{staticBackend: staticBackend{books: []catalog.Book{{ID: "one", Path: "Writer/Series/Series - Vol 01 - One.epub"}}}, version: 7}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "books", Name: "Books", Kind: "books", SourceType: "filesystem", Store: backend}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range []string{"0", "1"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/ingest-preview?library=books&limit=1&offset="+offset, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("preview %s: %d %s", offset, response.Code, response.Body.String())
		}
	}
	if backend.reads != 1 {
		t.Fatalf("catalogue was copied %d times for one generation", backend.reads)
	}
	backend.version++
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/ingest-preview?library=books", nil))
	if backend.reads != 2 {
		t.Fatalf("new generation did not invalidate preview: %d reads", backend.reads)
	}
}

func TestMetadataApplyUsesCatalogueOverrideForReadOnlyPDF(t *testing.T) {
	stateDir := t.TempDir()
	handler, err := NewMulti(MultiConfig{AllowOpen: true, MetadataReviewState: filepath.Join(stateDir, "reviews.json")}, []Library{{
		ID: "pdfs", Name: "PDFs", Kind: "books", SourceType: "readonly",
		Store: staticBackend{books: []catalog.Book{{ID: "one", Path: "Saga/One.pdf", Title: "One", Authors: []string{"Unknown"}, Format: "pdf"}}},
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/metadata/jobs?library=pdfs", nil))
	var job metadata.Job
	if err := json.Unmarshal(created.Body.Bytes(), &job); err != nil || len(job.Items) != 1 {
		t.Fatalf("create review: %#v %v", job, err)
	}
	decision := httptest.NewRecorder()
	handler.ServeHTTP(decision, httptest.NewRequest(http.MethodPut, "/api/metadata/jobs/"+job.ID+"/items/one/missing_author", strings.NewReader(`{"decision":"accepted","value":"Writer"}`)))
	applied := httptest.NewRecorder()
	handler.ServeHTTP(applied, httptest.NewRequest(http.MethodPost, "/api/metadata/jobs/"+job.ID+"/apply", strings.NewReader(`{"confirmation":"APPLY `+job.ID+`"}`)))
	if applied.Code != http.StatusOK || !strings.Contains(applied.Body.String(), `"override_books":1`) {
		t.Fatalf("apply override: %d %s", applied.Code, applied.Body.String())
	}
	books := httptest.NewRecorder()
	handler.ServeHTTP(books, httptest.NewRequest(http.MethodGet, "/api/books?library=pdfs", nil))
	if books.Code != http.StatusOK || !strings.Contains(books.Body.String(), `"authors":["Writer"]`) {
		t.Fatalf("catalogue did not expose override: %d %s", books.Code, books.Body.String())
	}
	reopened, err := catalog.OpenOverrideStore(filepath.Join(stateDir, "metadata-overrides.json"))
	if err != nil {
		t.Fatal(err)
	}
	book, _ := reopened.Wrap("pdfs", staticBackend{books: []catalog.Book{{ID: "one", Authors: []string{"Unknown"}}}}).Book("one")
	if len(book.Authors) != 1 || book.Authors[0] != "Writer" {
		t.Fatalf("override did not persist: %#v", book)
	}
}

func TestMetadataApplyRequiresConfirmationAndUpdatesTemporaryEPUB(t *testing.T) {
	libraryDir := t.TempDir()
	writeWebTestEPUB(t, filepath.Join(libraryDir, "one.epub"), "One")
	store := catalog.NewStore(libraryDir, filepath.Join(t.TempDir(), "catalog.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	stateDir := t.TempDir()
	handler, err := NewMulti(MultiConfig{AllowOpen: true, MetadataReviewState: filepath.Join(stateDir, "reviews.json"), MetadataBackupDirectory: filepath.Join(stateDir, "backups")}, []Library{{ID: "books", Name: "Books", Kind: "books", SourceType: "filesystem", MetadataWrites: true, Store: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/metadata/jobs?library=books", nil))
	var job metadata.Job
	if err := json.Unmarshal(created.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if len(job.Items) != 1 || job.Items[0].Code != "missing_author" {
		t.Fatalf("unexpected review: %#v", job)
	}
	decision := httptest.NewRecorder()
	decisionURL := "/api/metadata/jobs/" + job.ID + "/items/" + job.Items[0].BookID + "/missing_author"
	handler.ServeHTTP(decision, httptest.NewRequest(http.MethodPut, decisionURL, strings.NewReader(`{"decision":"accepted","value":"Writer"}`)))
	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, httptest.NewRequest(http.MethodPost, "/api/metadata/jobs/"+job.ID+"/apply", strings.NewReader(`{"confirmation":"wrong"}`)))
	if denied.Code != http.StatusBadRequest {
		t.Fatalf("unexpected confirmation response: %d %s", denied.Code, denied.Body.String())
	}
	applied := httptest.NewRecorder()
	handler.ServeHTTP(applied, httptest.NewRequest(http.MethodPost, "/api/metadata/jobs/"+job.ID+"/apply", strings.NewReader(`{"confirmation":"APPLY `+job.ID+`"}`)))
	if applied.Code != http.StatusOK || !strings.Contains(applied.Body.String(), `"status":"applied"`) {
		t.Fatalf("unexpected apply response: %d %s", applied.Code, applied.Body.String())
	}
	books := store.Books()
	if len(books) != 1 || len(books[0].Authors) != 1 || books[0].Authors[0] != "Writer" {
		t.Fatalf("catalog was not rescanned: %#v", books)
	}
}

func writeWebTestEPUB(t *testing.T, target, title string) {
	t.Helper()
	file, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entries := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": `<container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`,
		"OEBPS/content.opf":      `<package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>` + title + `</dc:title><dc:creator>Unknown</dc:creator></metadata><manifest/><spine/></package>`,
	}
	for _, name := range []string{"mimetype", "META-INF/container.xml", "OEBPS/content.opf"} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(entries[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOPDSContinueReading(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "reading.json")
	state, err := reading.Open(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.Put("default", readingPosition{LibraryID: "books", BookID: "one", Fraction: .42}, 0); err != nil {
		t.Fatal(err)
	}
	if err := state.Flush(); err != nil {
		t.Fatal(err)
	}
	book := catalog.Book{ID: "one", Path: "Series/One.epub", Title: "One", Authors: []string{"Author"}, MediaType: "application/epub+zip", Modified: time.Unix(10, 0)}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: statePath}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	root := httptest.NewRecorder()
	handler.ServeHTTP(root, httptest.NewRequest("GET", "/opds", nil))
	if !strings.Contains(root.Body.String(), ">Continue Reading<") || !strings.Contains(root.Body.String(), "/opds/continue") {
		t.Fatalf("continue navigation missing: %s", root.Body.String())
	}
	feed := httptest.NewRecorder()
	handler.ServeHTTP(feed, httptest.NewRequest("GET", "/opds/continue", nil))
	if feed.Code != 200 || !strings.Contains(feed.Body.String(), ">One<") || !strings.Contains(feed.Body.String(), "42% read") || !strings.Contains(feed.Body.String(), "/books/books/one/One.epub") {
		t.Fatalf("unexpected continue feed: %d %s", feed.Code, feed.Body.String())
	}
}

func TestOPDSContinueReadingIsBoundedToNewestHundred(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "reading.json")
	state, _ := reading.Open(statePath)
	books := make([]catalog.Book, 0, 101)
	for index := range 101 {
		id := fmt.Sprintf("book-%03d", index)
		books = append(books, catalog.Book{ID: id, Path: id + ".epub", Title: id, MediaType: "application/epub+zip", Modified: time.Unix(int64(index+1), 0)})
		if _, err := state.Put("default", readingPosition{LibraryID: "books", BookID: id, Fraction: .2}, 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := state.Flush(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: statePath}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: books}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	feed := httptest.NewRecorder()
	handler.ServeHTTP(feed, httptest.NewRequest(http.MethodGet, "/opds/continue", nil))
	var decoded Feed
	if err := xml.Unmarshal(feed.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Entries) != 100 {
		t.Fatalf("continue entries = %d, want 100", len(decoded.Entries))
	}
}

func TestOPDSContinueReadingIsAccountScoped(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "reading.json")
	state, _ := reading.Open(statePath)
	if _, err := state.Put("alice", readingPosition{LibraryID: "books", BookID: "one", Fraction: .2}, 0); err != nil {
		t.Fatal(err)
	}
	if err := state.Flush(); err != nil {
		t.Fatal(err)
	}
	book := catalog.Book{ID: "one", Path: "One.epub", Title: "One", MediaType: "application/epub+zip"}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: statePath}, []Library{{ID: "books", Name: "Books", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	requestFor := func(account string) *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/opds", nil)
		return request.WithContext(contextWithIdentity(request.Context(), accounts.Identity{ID: account, Username: account, Role: accounts.RoleUser}))
	}
	alice, bob := httptest.NewRecorder(), httptest.NewRecorder()
	handler.ServeHTTP(alice, requestFor("alice"))
	handler.ServeHTTP(bob, requestFor("bob"))
	if !strings.Contains(alice.Body.String(), ">Continue Reading<") || strings.Contains(bob.Body.String(), ">Continue Reading<") {
		t.Fatalf("account-scoped navigation alice=%s bob=%s", alice.Body.String(), bob.Body.String())
	}
}

func TestKoSyncProgressPopulatesCurrentlyReading(t *testing.T) {
	book := catalog.Book{ID: "one", Path: "Series/One.epub", Title: "One", Authors: []string{"Author"}, MediaType: "application/epub+zip", Modified: time.Unix(10, 0)}
	handler, err := NewMulti(MultiConfig{AllowOpen: true,
		KoSyncEnabled: true, KoSyncDatabase: filepath.Join(t.TempDir(), "kosync.json"), KoSyncUsername: "reader", KoSyncPassword: "secret",
		ReadingState: filepath.Join(t.TempDir(), "reading.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	document := koSyncFilenameMD5("One.epub")
	body := fmt.Sprintf(`{"document":%q,"progress":"/body/DocFragment[2]","percentage":0.42,"device":"KOReader","device_id":"reader-1"}`, document)
	key := md5.Sum([]byte("secret"))
	request := koSyncRequest(http.MethodPut, "/syncs/progress", "reader", hex.EncodeToString(key[:]), body)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("KoSync update returned %d: %s", response.Code, response.Body.String())
	}
	dashboard := httptest.NewRecorder()
	handler.ServeHTTP(dashboard, httptest.NewRequest(http.MethodGet, "/api/reading", nil))
	if dashboard.Code != http.StatusOK || !strings.Contains(dashboard.Body.String(), `"title":"One"`) || !strings.Contains(dashboard.Body.String(), `"fraction":0.42`) {
		t.Fatalf("KoSync book missing from dashboard: %d %s", dashboard.Code, dashboard.Body.String())
	}
}

func TestBooksAPIIncludesAccountReadingProgress(t *testing.T) {
	book := catalog.Book{ID: "one", Path: "Series/One.epub", Title: "One", Authors: []string{"Author"}, Series: "Series", MediaType: "application/epub+zip"}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	save := httptest.NewRecorder()
	handler.ServeHTTP(save, httptest.NewRequest(http.MethodPut, "/api/reading/books/one", strings.NewReader(`{"fraction":0.48,"base_revision":0,"device_id":"web"}`)))
	if save.Code != http.StatusOK {
		t.Fatalf("save progress = %d %s", save.Code, save.Body.String())
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/books?library=books&series=Series", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"reading_progress":{"fraction":0.48`) {
		t.Fatalf("books progress = %d %s", response.Code, response.Body.String())
	}
}

func TestReadingAPIAcknowledgesEquivalentStalePositionWithoutReplacingCurrent(t *testing.T) {
	book := catalog.Book{ID: "one", Path: "Series/One.epub", Title: "One", MediaType: "application/epub+zip"}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPut, "/api/reading/books/one", strings.NewReader(`{"cfi":"epubcfi(/6/4!/4/2:7)","fraction":0.5,"base_revision":0,"device_id":"tablet","locators":{"readium":"authoritative"}}`)))
	if first.Code != http.StatusOK {
		t.Fatalf("first save = %d %s", first.Code, first.Body.String())
	}

	stale := httptest.NewRecorder()
	handler.ServeHTTP(stale, httptest.NewRequest(http.MethodPut, "/api/reading/books/one", strings.NewReader(`{"fraction":0.5015,"base_revision":0,"device_id":"phone","locators":{"readium":"stale"}}`)))
	if stale.Code != http.StatusOK || !strings.Contains(stale.Body.String(), `"revision":1`) || !strings.Contains(stale.Body.String(), `"device_id":"tablet"`) || !strings.Contains(stale.Body.String(), `"readium":"authoritative"`) {
		t.Fatalf("equivalent stale save = %d %s", stale.Code, stale.Body.String())
	}
	semantic := httptest.NewRecorder()
	handler.ServeHTTP(semantic, httptest.NewRequest(http.MethodPut, "/api/reading/books/one", strings.NewReader(`{"cfi":"epubcfi(/6/4!/4/2:7)","fraction":0.75,"base_revision":0,"device_id":"phone"}`)))
	if semantic.Code != http.StatusOK || !strings.Contains(semantic.Body.String(), `"fraction":0.5`) || !strings.Contains(semantic.Body.String(), `"device_id":"tablet"`) {
		t.Fatalf("semantic stale save = %d %s", semantic.Code, semantic.Body.String())
	}

	conflict := httptest.NewRecorder()
	handler.ServeHTTP(conflict, httptest.NewRequest(http.MethodPut, "/api/reading/books/one", strings.NewReader(`{"fraction":0.52,"base_revision":0,"device_id":"phone"}`)))
	if conflict.Code != http.StatusConflict {
		t.Fatalf("different stale save = %d %s", conflict.Code, conflict.Body.String())
	}
}

func TestReadingAPIConflictsOnAdjacentFixedLayoutPages(t *testing.T) {
	book := catalog.Book{ID: "one", Path: "Series/One.cbz", Title: "One", MediaType: "application/vnd.comicbook+zip"}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")}, []Library{{ID: "comics", Name: "Comics", Kind: "comics", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPut, "/api/reading/comics/one", strings.NewReader(`{"fraction":0.5015,"base_revision":0,"device_id":"tablet","locators":{"comic":"{\"version\":1,\"page\":501,\"page_count\":1001}"}}`)))
	if first.Code != http.StatusOK {
		t.Fatalf("first save = %d %s", first.Code, first.Body.String())
	}
	stale := httptest.NewRecorder()
	handler.ServeHTTP(stale, httptest.NewRequest(http.MethodPut, "/api/reading/comics/one", strings.NewReader(`{"fraction":0.5,"base_revision":0,"device_id":"phone","locators":{"comic":"{\"version\":1,\"page\":500,\"page_count\":1001}"}}`)))
	if stale.Code != http.StatusConflict {
		t.Fatalf("adjacent fixed-layout save = %d %s", stale.Code, stale.Body.String())
	}
}

func TestDisabledWebReaderBridgeDoesNotDisableKoSync(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true,
		KoSyncEnabled: true, DisableKoSyncWebReaderBridge: true, KoSyncDatabase: filepath.Join(t.TempDir(), "kosync.json"), KoSyncUsername: "reader", KoSyncPassword: "secret",
		ReadingState: filepath.Join(t.TempDir(), "reading.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{{ID: "one", Path: "One.epub", Title: "One"}}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	syncResponse := httptest.NewRecorder()
	handler.ServeHTTP(syncResponse, httptest.NewRequest(http.MethodGet, "/api/books/books/one/sync", nil))
	if syncResponse.Code != http.StatusNotFound {
		t.Fatalf("webreader bridge returned %d", syncResponse.Code)
	}
	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthcheck", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("KoSync server returned %d", health.Code)
	}
}

func TestWebLibraryAndEmptyBooksAPI(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := catalog.NewStore(t.TempDir(), filepath.Join(t.TempDir(), "empty.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, logger)
	if err != nil {
		t.Fatal(err)
	}

	root := httptest.NewRecorder()
	handler.ServeHTTP(root, httptest.NewRequest("GET", "/", nil))
	if root.Code != 200 || !strings.Contains(root.Body.String(), "Mochil") {
		t.Fatalf("unexpected web root: %d", root.Code)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/books", nil))
	var books []bookView
	if err := json.Unmarshal(response.Body.Bytes(), &books); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if books == nil || len(books) != 0 {
		t.Fatalf("empty API must return [], got %s", response.Body.String())
	}
}

func TestKoSyncFallsBackCredentialsIndependently(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, err := NewMulti(MultiConfig{
		Username:       "reader",
		Password:       "catalog-password",
		KoSyncEnabled:  true,
		KoSyncDatabase: filepath.Join(t.TempDir(), "kosync.json"),
		KoSyncPassword: "sync-password",
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum([]byte("sync-password"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, koSyncRequest("GET", "/users/auth", "reader", hex.EncodeToString(sum[:]), ""))
	if response.Code != 200 {
		t.Fatalf("configured sync password was ignored: %d %s", response.Code, response.Body.String())
	}
}

func TestWebIndexRefreshesAfterBackendPublishesSnapshot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &versionedTestBackend{}
	store.replace([]catalog.Book{{ID: "z", Title: "The Zebra", Authors: []string{"First Author"}}})
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, logger)
	if err != nil {
		t.Fatal(err)
	}

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest("GET", "/api/books?q=first", nil))
	if first.Code != 200 || !strings.Contains(first.Body.String(), `"id":"z"`) {
		t.Fatalf("initial indexed response missing book: %d %s", first.Code, first.Body.String())
	}

	store.replace([]catalog.Book{{ID: "a", Title: "An Apple", Authors: []string{"Second Author"}}})
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest("GET", "/api/books?q=second&bucket=A", nil))
	if second.Code != 200 || !strings.Contains(second.Body.String(), `"id":"a"`) || strings.Contains(second.Body.String(), `"id":"z"`) {
		t.Fatalf("index did not follow new snapshot: %d %s", second.Code, second.Body.String())
	}

	libraries := httptest.NewRecorder()
	handler.ServeHTTP(libraries, httptest.NewRequest("GET", "/api/libraries", nil))
	if !strings.Contains(libraries.Body.String(), `"count":1`) {
		t.Fatalf("library count did not use refreshed index: %s", libraries.Body.String())
	}
}

func TestWebIndexProvidesStableSingleLibraryOrdering(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := staticBackend{books: []catalog.Book{
		{ID: "b", Title: "The Same"},
		{ID: "z", Title: "Zebra"},
		{ID: "a", Title: "Same"},
	}}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/books", nil))
	var books []bookView
	if err := json.Unmarshal(response.Body.Bytes(), &books); err != nil {
		t.Fatal(err)
	}
	if len(books) != 3 || books[0].ID != "a" || books[1].ID != "b" || books[2].ID != "z" {
		t.Fatalf("unexpected stable ordering: %+v", books)
	}
}

func TestDashboardDismissPreservesAndRestoresReadingState(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := staticBackend{books: []catalog.Book{{ID: "one", Title: "One"}}}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	put := httptest.NewRecorder()
	handler.ServeHTTP(put, httptest.NewRequest("PUT", "/api/reading/books/one", strings.NewReader(`{"cfi":"epubcfi(/6/2)","fraction":0.2,"base_revision":0}`)))
	if put.Code != 200 {
		t.Fatalf("save reading position: %d %s", put.Code, put.Body.String())
	}
	dismiss := httptest.NewRecorder()
	handler.ServeHTTP(dismiss, httptest.NewRequest("POST", "/api/reading/books/one/dismiss", nil))
	if dismiss.Code != 200 {
		t.Fatalf("dismiss reading position: %d %s", dismiss.Code, dismiss.Body.String())
	}
	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest("GET", "/api/reading", nil))
	if strings.TrimSpace(list.Body.String()) != "[]" {
		t.Fatalf("dismissed position remained on dashboard: %s", list.Body.String())
	}
	show := httptest.NewRecorder()
	handler.ServeHTTP(show, httptest.NewRequest("POST", "/api/reading/books/one/show", nil))
	if show.Code != 200 {
		t.Fatalf("restore reading position: %d %s", show.Code, show.Body.String())
	}
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest("GET", "/api/reading/books/one", nil))
	if get.Code != 200 || !strings.Contains(get.Body.String(), `"cfi":"epubcfi(/6/2)"`) || !strings.Contains(get.Body.String(), `"fraction":0.2`) {
		t.Fatalf("dismiss changed reading data: %d %s", get.Code, get.Body.String())
	}
}

func TestSeriesAPIGroupingBucketsAndOrdering(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	one, oneHalf, two := 1.0, 1.5, 2.0
	store := staticBackend{books: []catalog.Book{
		{ID: "b", Title: "Second", Series: "The Expanse", SeriesIndex: &two},
		{ID: "a", Title: "First", Authors: []string{"James S. A. Corey"}, Series: "The Expanse", SeriesIndex: &one},
		{ID: "c", Title: "Novella", Series: "The Expanse", SeriesIndex: &oneHalf},
		{ID: "d", Title: "Loose Book"},
		{ID: "e", Title: "Numeric", Series: "12 Kingdoms"},
	}}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, logger)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/series?bucket=E", nil))
	var groups []seriesView
	if err := json.Unmarshal(response.Body.Bytes(), &groups); err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Name != "The Expanse" || groups[0].Count != 3 || groups[0].Bucket != "E" || groups[0].CoverURL != "/covers/books/a" || len(groups[0].Authors) != 1 || groups[0].Authors[0] != "James S. A. Corey" {
		t.Fatalf("unexpected groups: %+v", groups)
	}

	standalone := httptest.NewRecorder()
	handler.ServeHTTP(standalone, httptest.NewRequest("GET", "/api/series", nil))
	if !strings.Contains(standalone.Body.String(), `"name":"Standalone books"`) {
		t.Fatalf("missing standalone group: %s", standalone.Body.String())
	}
	standaloneBucket := httptest.NewRecorder()
	handler.ServeHTTP(standaloneBucket, httptest.NewRequest("GET", "/api/series?bucket=L", nil))
	if !strings.Contains(standaloneBucket.Body.String(), `"name":"Standalone books"`) {
		t.Fatalf("standalone title bucket missing: %s", standaloneBucket.Body.String())
	}
	numericBucket := httptest.NewRecorder()
	handler.ServeHTTP(numericBucket, httptest.NewRequest("GET", "/api/series?bucket=%23", nil))
	if !strings.Contains(numericBucket.Body.String(), `"name":"12 Kingdoms"`) {
		t.Fatalf("numeric series missing from #: %s", numericBucket.Body.String())
	}

	books := httptest.NewRecorder()
	handler.ServeHTTP(books, httptest.NewRequest("GET", "/api/books?library=books&series=The+Expanse", nil))
	var ordered []bookView
	if err := json.Unmarshal(books.Body.Bytes(), &ordered); err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 3 || ordered[0].ID != "a" || ordered[1].ID != "c" || ordered[2].ID != "b" {
		t.Fatalf("unexpected series order: %+v", ordered)
	}
}

func TestSettingsRedactsAndPreservesKoSyncPassword(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := []byte(`{"kosync":{"provider":"external","server_url":"https://sync.example","username":"reader","password":"secret"},"libraries":[]}`)
	handler, err := NewMulti(MultiConfig{AllowOpen: true,
		ConfigJSON: func() ([]byte, error) { return config, nil },
		SaveConfig: func(data []byte) error { config = append([]byte(nil), data...); return nil },
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest("GET", "/api/settings", nil))
	if strings.Contains(get.Body.String(), `"password":"secret"`) || !strings.Contains(get.Body.String(), `"password_configured":true`) {
		t.Fatalf("password was not redacted: %s", get.Body.String())
	}
	update := []byte(`{"kosync":{"provider":"external","server_url":"https://sync.example","username":"reader","effective_username":"reader","password_configured":true},"libraries":[{"id":"movies","name":"Movies","kind":"movies","source":{"type":"filesystem","path":"/media/movies"},"inference":{"folder_layout":"auto"}}]}`)
	put := httptest.NewRecorder()
	handler.ServeHTTP(put, httptest.NewRequest("PUT", "/api/settings", bytes.NewReader(update)))
	if put.Code != 200 || !strings.Contains(string(config), `"password": "secret"`) || !strings.Contains(string(config), `"id": "movies"`) || strings.Contains(string(config), "effective_username") || strings.Contains(string(config), "password_configured") {
		t.Fatalf("saved password was not preserved: %d %s", put.Code, config)
	}
}

func TestSettingsUsesLiveApplyWhenAvailable(t *testing.T) {
	config := []byte(`{"scan_every":"10m","libraries":[]}`)
	called := false
	handler, err := NewMulti(MultiConfig{AllowOpen: true,
		ConfigJSON: func() ([]byte, error) { return config, nil },
		ApplyConfig: func(data []byte) (SettingsApplyResult, error) {
			called = true
			config = append([]byte(nil), data...)
			return SettingsApplyResult{Saved: true, Applied: true, Libraries: 2}, nil
		},
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"scan_every":"5m","libraries":[]}`)))
	if response.Code != http.StatusOK || !called || !strings.Contains(response.Body.String(), `"applied":true`) || !strings.Contains(response.Body.String(), `"libraries":2`) {
		t.Fatalf("live apply response: %d %s", response.Code, response.Body.String())
	}
}

func TestKoSyncFindsBookByContentHashWhenFilenameDiffers(t *testing.T) {
	book := catalog.Book{ID: "one", Path: "Series/One.epub", Title: "One", Authors: []string{"Author"}, MediaType: "application/epub+zip", Modified: time.Unix(10, 0)}
	handler, err := NewMulti(MultiConfig{AllowOpen: true,
		KoSyncEnabled: true, KoSyncDatabase: filepath.Join(t.TempDir(), "kosync.json"), KoSyncUsername: "reader", KoSyncPassword: "secret",
		ReadingState: filepath.Join(t.TempDir(), "reading.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	// KOReader names the document by hashing the file's content, not its (possibly renamed) filename.
	document := koSyncPartialMD5(bytes.NewReader([]byte(book.Title)))
	body := fmt.Sprintf(`{"document":%q,"progress":"/body/DocFragment[2]","percentage":0.42,"device":"KOReader","device_id":"reader-1"}`, document)
	key := md5.Sum([]byte("secret"))
	// The content index fills in the background, so this single update may arrive before it
	// is ready; it must still reach the dashboard once it is, without KOReader sending it again.
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, koSyncRequest(http.MethodPut, "/syncs/progress", "reader", hex.EncodeToString(key[:]), body))
	if response.Code != http.StatusOK {
		t.Fatalf("KoSync update returned %d: %s", response.Code, response.Body.String())
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		dashboard := httptest.NewRecorder()
		handler.ServeHTTP(dashboard, httptest.NewRequest(http.MethodGet, "/api/reading", nil))
		if strings.Contains(dashboard.Body.String(), `"fraction":0.42`) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("content-hash KoSync book never reached the dashboard: %s", dashboard.Body.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// opdsContinueHandler builds the OPDS handler for a reading state file holding the given
// positions, with one book in the catalogue for each of them.
func opdsContinueHandler(t *testing.T, positions map[string]reading.Position) http.Handler {
	t.Helper()
	statePath := filepath.Join(t.TempDir(), "reading.json")
	state, err := json.Marshal(map[string]any{"version": 1, "accounts": map[string]any{"default": positions}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, state, 0600); err != nil {
		t.Fatal(err)
	}
	books := make([]catalog.Book, 0, len(positions))
	for _, position := range positions {
		books = append(books, catalog.Book{ID: position.BookID, Path: position.BookID + ".epub", Title: strings.ToUpper(position.BookID[:1]) + position.BookID[1:],
			MediaType: "application/epub+zip", Modified: time.Unix(10, 0)})
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: statePath}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: books}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

// "Continue Reading" lists what is being read. A book only opened at page one was never read, and
// one parked as on hold or dropped was put aside on purpose: neither belongs in the feed.
func TestOPDSContinueReadingSkipsUnreadAndParkedBooks(t *testing.T) {
	handler := opdsContinueHandler(t, map[string]reading.Position{
		"books\x00reading": {LibraryID: "books", BookID: "reading", UpdatedAt: 400, Fraction: .42},
		"books\x00opened":  {LibraryID: "books", BookID: "opened", UpdatedAt: 300},
		"books\x00dropped": {LibraryID: "books", BookID: "dropped", UpdatedAt: 200, Fraction: .5, Status: reading.StatusDropped},
		"books\x00onhold":  {LibraryID: "books", BookID: "onhold", UpdatedAt: 100, Fraction: .3, Status: reading.StatusOnHold},
	})
	feed := httptest.NewRecorder()
	handler.ServeHTTP(feed, httptest.NewRequest(http.MethodGet, "/opds/continue", nil))
	if feed.Code != http.StatusOK {
		t.Fatalf("continue feed = %d: %s", feed.Code, feed.Body.String())
	}
	body := feed.Body.String()
	if !strings.Contains(body, ">Reading<") || !strings.Contains(body, "42% read") {
		t.Fatalf("the book being read is missing: %s", body)
	}
	for _, unwanted := range []string{">Opened<", ">Dropped<", ">Onhold<", "0% read"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("continue feed lists %q: %s", unwanted, body)
		}
	}
}

// The navigation entry follows the feed: with nothing actually being read there is nothing to list.
func TestOPDSRootOmitsContinueNavigationWhenNothingIsBeingRead(t *testing.T) {
	handler := opdsContinueHandler(t, map[string]reading.Position{
		"books\x00opened":  {LibraryID: "books", BookID: "opened", UpdatedAt: 200},
		"books\x00dropped": {LibraryID: "books", BookID: "dropped", UpdatedAt: 100, Fraction: .5, Status: reading.StatusDropped},
	})
	root := httptest.NewRecorder()
	handler.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/opds", nil))
	if strings.Contains(root.Body.String(), ">Continue Reading<") {
		t.Fatalf("root feed offers Continue Reading with nothing in progress: %s", root.Body.String())
	}
}
