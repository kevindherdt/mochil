package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"mochil/internal/catalog"
)

func TestBookCursorBindsViewAndExpiresWithCatalogue(t *testing.T) {
	backend := &versionedTestBackend{}
	backend.replace([]catalog.Book{{ID: "a", Title: "Alpha"}, {ID: "b", Title: "Beta"}, {ID: "c", Title: "Gamma"}})
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: backend}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/books?library=books&limit=1", nil))
	cursor := first.Header().Get("X-Next-Cursor")
	if first.Code != http.StatusOK || cursor == "" || !strings.Contains(first.Body.String(), `"title":"Alpha"`) {
		t.Fatalf("first page = %d cursor=%q body=%s", first.Code, cursor, first.Body.String())
	}
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/books?library=books&limit=1&cursor="+url.QueryEscape(cursor), nil))
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"title":"Beta"`) {
		t.Fatalf("second page = %d %s", second.Code, second.Body.String())
	}
	wrongView := httptest.NewRecorder()
	handler.ServeHTTP(wrongView, httptest.NewRequest(http.MethodGet, "/api/books?library=books&bucket=A&cursor="+url.QueryEscape(cursor), nil))
	if wrongView.Code != http.StatusBadRequest {
		t.Fatalf("cross-view cursor = %d", wrongView.Code)
	}
	backend.replace([]catalog.Book{{ID: "a", Title: "Alpha"}, {ID: "b", Title: "Beta"}, {ID: "c", Title: "Gamma"}, {ID: "d", Title: "Delta"}})
	expired := httptest.NewRecorder()
	handler.ServeHTTP(expired, httptest.NewRequest(http.MethodGet, "/api/books?library=books&cursor="+url.QueryEscape(cursor), nil))
	if expired.Code != http.StatusConflict {
		t.Fatalf("expired cursor = %d %s", expired.Code, expired.Body.String())
	}
}

func TestRecentBookCursorIsOrderedAndCannotReplayIntoTitleView(t *testing.T) {
	backend := &versionedTestBackend{}
	base := time.Unix(1_700_000_000, 0)
	backend.replace([]catalog.Book{{ID: "old", Title: "Alpha", Modified: base}, {ID: "new", Title: "Zulu", Modified: base.Add(time.Hour)}})
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: backend}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/books?library=books&sort=recent&limit=1", nil))
	cursor := first.Header().Get("X-Next-Cursor")
	if first.Code != http.StatusOK || cursor == "" || !strings.Contains(first.Body.String(), `"title":"Zulu"`) {
		t.Fatalf("recent first=%d cursor=%q body=%s", first.Code, cursor, first.Body.String())
	}
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/books?library=books&sort=recent&limit=1&cursor="+url.QueryEscape(cursor), nil))
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"title":"Alpha"`) {
		t.Fatalf("recent second=%d %s", second.Code, second.Body.String())
	}
	wrong := httptest.NewRecorder()
	handler.ServeHTTP(wrong, httptest.NewRequest(http.MethodGet, "/api/books?library=books&cursor="+url.QueryEscape(cursor), nil))
	if wrong.Code != http.StatusBadRequest {
		t.Fatalf("recent cursor replay status=%d", wrong.Code)
	}
}

func TestSeriesCursorContinuesByStableLibraryAndName(t *testing.T) {
	backend := &versionedTestBackend{}
	backend.replace([]catalog.Book{{ID: "a", Title: "One", Series: "Alpha"}, {ID: "b", Title: "Two", Series: "Beta"}, {ID: "c", Title: "Three", Series: "Gamma"}})
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: backend}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/series?library=books&limit=1", nil))
	cursor := first.Header().Get("X-Next-Cursor")
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/series?library=books&limit=1&cursor="+url.QueryEscape(cursor), nil))
	if cursor == "" || second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"name":"Beta"`) {
		t.Fatalf("series continuation = cursor %q, status %d, body %s", cursor, second.Code, second.Body.String())
	}
}

func TestAuthorCursorAndBookFilterRemainBoundToAuthorView(t *testing.T) {
	backend := &versionedTestBackend{}
	backend.replace([]catalog.Book{{ID: "a", Title: "Alpha", Authors: []string{"Ada"}}, {ID: "b", Title: "Beta", Authors: []string{"Bea"}}})
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: backend}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/authors?library=books&limit=1", nil))
	cursor := first.Header().Get("X-Next-Cursor")
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/authors?library=books&limit=1&cursor="+url.QueryEscape(cursor), nil))
	if cursor == "" || second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"name":"Bea"`) {
		t.Fatalf("author continuation = %q %d %s", cursor, second.Code, second.Body.String())
	}
	books := httptest.NewRecorder()
	handler.ServeHTTP(books, httptest.NewRequest(http.MethodGet, "/api/books?library=books&author=Ada", nil))
	if books.Code != http.StatusOK || !strings.Contains(books.Body.String(), `"title":"Alpha"`) || strings.Contains(books.Body.String(), `"title":"Beta"`) {
		t.Fatalf("author books = %d %s", books.Code, books.Body.String())
	}
}

func TestPublicationCursorRejectsMalformedValues(t *testing.T) {
	for _, value := range []string{"%%%", strings.Repeat("a", 1025), encodePublicationCursor(publicationCursor{Kind: "books", Version: 1, View: "view", LibraryID: "books"})} {
		if _, err := decodePublicationCursor(value); err == nil {
			t.Fatalf("accepted malformed cursor %q", value)
		}
	}
}
