package web

// Writes real server response fixtures to contracts/api/<endpoint>.json (plan docs/rewrite-plan.md §5.4): the
// server-side half of the contract between Go and the future TypeScript UI. These are not hand-authored shapes that
// could drift from what the server actually returns — they are the real JSON a real handler produces for seeded,
// deterministic data, so a server change that breaks the UI's assumptions fails here before anyone wires up zod in
// Phase 1.
//
// Run with WRITE_CONTRACT_FIXTURES=1 to (re)write the files after a deliberate API change. Without it, the test
// fails if the handler's current output no longer matches the committed fixture — the same "regenerate and diff"
// shape as `npm run tokens:check` (scripts/gen-tokens.mjs), so a fixture can only drift on purpose.
//
// Covered: books, libraries, authors, collections, audio, reading, accounts, modules.
//
// Deliberately NOT covered: /api/version (tried and dropped — its `started`/`uptime`/`pid` fields change on every
// process, so a frozen fixture for it drifts on every run and proves nothing; it keeps its own behavioural tests in
// about_http_test.go) and /api/video-details (needs a live TMDB network call — `apiVideoDetails` only answers once
// `m.tmdb` and `library.MetadataProvider == "tmdb"` are set, and the handler itself calls out to the real TMDB API
// with no seam in `NewMulti` to inject a fake transport. Mocking that would mean changing production wiring just to
// make a test possible, which is backwards; this endpoint is left for whenever that seam is added for its own
// reason, not for this slice's sake).

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"mochil/internal/audio"
	"mochil/internal/catalog"
	"mochil/internal/reading"
)

var (
	// A reading position's save time, in the millisecond-epoch shape the server writes it in.
	redactUpdatedAtMillis = regexp.MustCompile(`"updated_at": ?\d+`)
	// A scanned file's on-disk modification time, and the random revision token minted from it (also embedded in
	// the two artwork URLs as ?v=<token>) -- all three are volatile because the test writes the file fresh each run.
	redactModifiedTimestamp = regexp.MustCompile(`"modified": ?"[^"]*"`)
	redactRevisionToken     = regexp.MustCompile(`[0-9a-f]{24}`)
	// A collection's generated id, minted fresh each test run.
	redactCollectionID = regexp.MustCompile(`"id": ?"[0-9a-f]{16}"`)
)

// contractFixture names the endpoint, the request that produces it, and the server that must answer it.
type contractFixture struct {
	name    string // contracts/api/<name>.json
	method  string
	path    string
	cookies bool // whether to attach the login cookie before the request
	// redact rewrites genuinely non-deterministic bytes (a timestamp, a random revision token) to a fixed
	// placeholder before the fixture is compared or recorded. Unlike /api/version (dropped entirely, see the file
	// comment), these endpoints are mostly stable data with one or two volatile fields worth keeping real -- the
	// alternative is a fixture that drifts on every run and proves nothing, which is worse than no fixture.
	redact func([]byte) []byte
}

func writeOrCheckContractFixture(t *testing.T, handler http.Handler, loginCookie *http.Cookie, fixture contractFixture) {
	t.Helper()

	request := httptest.NewRequest(fixture.method, fixture.path, nil)
	if fixture.cookies && loginCookie != nil {
		request.AddCookie(loginCookie)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s %s = %d, want 200: %s", fixture.method, fixture.path, recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.Bytes()
	if fixture.redact != nil {
		body = fixture.redact(body)
	}

	// Re-indent so the fixture is readable in a diff, rather than committing whatever single-line form the
	// handler's json.Encoder happened to produce.
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("%s is not valid JSON after redaction: %v (%s)", fixture.path, err, body)
	}
	pretty, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	pretty = append(pretty, '\n')

	fixturePath := filepath.Join("..", "..", "contracts", "api", fixture.name+".json")
	current, readErr := os.ReadFile(fixturePath)

	if os.Getenv("WRITE_CONTRACT_FIXTURES") == "1" {
		if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fixturePath, pretty, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	if readErr != nil {
		t.Fatalf("no committed fixture at %s — run WRITE_CONTRACT_FIXTURES=1 go test ./internal/web -run TestContractFixtures", fixturePath)
	}
	if string(current) != string(pretty) {
		t.Fatalf("%s has changed since it was recorded — run WRITE_CONTRACT_FIXTURES=1 go test ./internal/web -run TestContractFixtures to re-record it (only do this for a deliberate API change)", fixturePath)
	}
}

func TestContractFixtures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Deterministic, fixed data: a contract fixture that regenerates with different content every run (timestamps,
	// random ids) is worse than none, because the diff never settles.
	books := staticBackend{books: []catalog.Book{
		{ID: "owner-book", Path: "Owner.epub", Title: "The Bellwether Line", Authors: []string{"Nora Ashcombe"}, Series: "Bellwether", Format: "epub"},
		{ID: "reader-book", Path: "Reader.epub", Title: "A Quiet Harbour", Authors: []string{"Nora Ashcombe"}, Format: "epub"},
	}}

	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{
		{ID: "books", Name: "Books", Kind: "books", Store: books},
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := handler.(interface{ Close() }); ok {
		defer closer.Close()
	}

	for _, fixture := range []contractFixture{
		{name: "books", method: http.MethodGet, path: "/api/books"},
		{name: "libraries", method: http.MethodGet, path: "/api/libraries"},
		{name: "authors", method: http.MethodGet, path: "/api/authors?library=books"},
		{name: "modules", method: http.MethodGet, path: "/api/modules"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			writeOrCheckContractFixture(t, handler, nil, fixture)
		})
	}
}

// TestContractFixtureCollections needs its own server: m.collections is only wired up when CollectionsState is
// set, and a fixture worth having needs one real collection in it, not an empty list indistinguishable from a bug.
func TestContractFixtureCollections(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	books := staticBackend{books: []catalog.Book{
		{ID: "owner-book", Path: "Owner.epub", Title: "The Bellwether Line", Authors: []string{"Nora Ashcombe"}, Series: "Bellwether", Format: "epub"},
	}}
	handler, err := NewMulti(MultiConfig{
		AllowOpen:        true,
		CollectionsState: filepath.Join(t.TempDir(), "collections.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: books}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := handler.(interface{ Close() }); ok {
		defer closer.Close()
	}

	// readingAccount(r) resolves to "default" on an open install with no login (account_identity.go), so an
	// unauthenticated POST here is the real shape of the data an open-install fixture would actually contain.
	create := httptest.NewRequest(http.MethodPost, "/api/collections", strings.NewReader(`{"name":"Favourites","library_id":"books"}`))
	create.Header.Set("Content-Type", "application/json")
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, create)
	if createResponse.Code != http.StatusCreated && createResponse.Code != http.StatusOK {
		t.Fatalf("create collection = %d: %s", createResponse.Code, createResponse.Body.String())
	}

	writeOrCheckContractFixture(t, handler, nil, contractFixture{
		name: "collections", method: http.MethodGet, path: "/api/collections",
		redact: func(body []byte) []byte {
			return redactCollectionID.ReplaceAll(body, []byte(`"id": "0000000000000000"`))
		},
	})
}

// TestContractFixtureAudio needs a real audio library: apiAudio 404s without one. The files are not valid audio
// (no real codec, no tags), the same shape as the existing playlist-download tests in this package — apiAudio lists
// what Scan() found on disk, which does not require decoding, and falls back to the filename for a title.
func TestContractFixtureAudio(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "A Steady Light.mp3"), []byte("first track"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := audio.NewStore("music", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{
		{ID: "music", Name: "Music", Kind: "music", Audio: store},
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := handler.(interface{ Close() }); ok {
		defer closer.Close()
	}

	writeOrCheckContractFixture(t, handler, nil, contractFixture{
		name: "audio", method: http.MethodGet, path: "/api/audio?library=music",
		redact: func(body []byte) []byte {
			body = redactModifiedTimestamp.ReplaceAll(body, []byte(`"modified": "<redacted>"`))
			return redactRevisionToken.ReplaceAll(body, []byte("000000000000000000000000"))
		},
	})
}

// TestContractFixtureReading needs ReadingState wired and a seeded position: /api/reading 404s with m.reading == nil,
// and an empty reading list is indistinguishable from "the endpoint returns nothing", which is not the contract.
func TestContractFixtureReading(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	directory := t.TempDir()
	readingPath := filepath.Join(directory, "reading.json")
	positions, err := reading.Open(readingPath)
	if err != nil {
		t.Fatal(err)
	}
	// "default" is exactly what readingAccount(r) resolves to for an open install with no login
	// (account_identity.go) -- this fixture is what an unauthenticated open install actually returns.
	if _, err = positions.Put("default", reading.Position{LibraryID: "books", BookID: "owner-book", CFI: "epubcfi(/6/4)", Fraction: .42}, 0); err != nil {
		t.Fatal(err)
	}
	if err := positions.Flush(); err != nil {
		t.Fatal(err)
	}
	books := staticBackend{books: []catalog.Book{
		{ID: "owner-book", Path: "Owner.epub", Title: "The Bellwether Line", Authors: []string{"Nora Ashcombe"}, Series: "Bellwether", Format: "epub"},
	}}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: readingPath}, []Library{
		{ID: "books", Name: "Books", Kind: "books", Store: books},
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := handler.(interface{ Close() }); ok {
		defer closer.Close()
	}

	writeOrCheckContractFixture(t, handler, nil, contractFixture{
		name: "reading", method: http.MethodGet, path: "/api/reading",
		redact: func(body []byte) []byte {
			return redactUpdatedAtMillis.ReplaceAll(body, []byte(`"updated_at": 0`))
		},
	})
}

// TestContractFixtureAccounts needs its own server: m.accounts stays nil (first run, closed) unless either
// cfg.Username seeds an administrator or an accounts.json already exists, and AllowOpen:true does not change that --
// it only decides whether a request needs a session, not whether the store exists.
func TestContractFixtureAccounts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// Listing accounts is an administrator-only view, and /login itself only exists while the wall is up --
	// AllowOpen:true serves the SPA shell at /login instead (404 from this handler's own router), which is correct
	// product behaviour (an open install has no sign-in flow at all) but means this fixture needs the ordinary
	// walled configuration, not AllowOpen, to reach a real admin session.
	handler, err := NewMulti(MultiConfig{
		Username:      "owner",
		Password:      "owner password",
		AccountsState: filepath.Join(t.TempDir(), "accounts.json"),
	}, nil, logger)
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := handler.(interface{ Close() }); ok {
		defer closer.Close()
	}

	loginRequest := loginPOST(t, handler, url.Values{"username": {"owner"}, "password": {"owner password"}})
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusSeeOther {
		t.Fatalf("admin login = %d: %s", loginResponse.Code, loginResponse.Body.String())
	}
	cookies := loginResponse.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("admin login set no cookie")
	}

	writeOrCheckContractFixture(t, handler, cookies[0], contractFixture{name: "accounts", method: http.MethodGet, path: "/api/accounts", cookies: true})
}
