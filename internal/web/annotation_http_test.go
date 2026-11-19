package web

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/catalog"
)

func TestAnnotationAPIAppliesAndPullsIncrementalChanges(t *testing.T) {
	book := catalog.Book{ID: "one", Title: "One", Path: "one.epub"}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/annotations/books/one", bytes.NewBufferString(`{"changes":[{"base_revision":0,"annotation":{"id":"mark-1","kind":"highlight","quote":"A useful sentence"}}]}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"revision":1`) {
		t.Fatalf("put: %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/annotations/books/one?since=0", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"mark-1"`) {
		t.Fatalf("get: %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/annotations/books/missing", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing book returned %d", response.Code)
	}
}

func TestAnnotationAPIExportsAndRequiresExplicitDeletionConfirmation(t *testing.T) {
	book := catalog.Book{ID: "one", Title: "One", Path: "one.epub"}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	put := httptest.NewRequest(http.MethodPut, "/api/annotations/books/one", strings.NewReader(`{"changes":[{"base_revision":0,"annotation":{"id":"mark-1","kind":"highlight","quote":"Export me"}}]}`))
	put.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), put)

	exported := httptest.NewRecorder()
	handler.ServeHTTP(exported, httptest.NewRequest(http.MethodGet, "/api/annotations/export", nil))
	if exported.Code != http.StatusOK || !strings.Contains(exported.Header().Get("Content-Disposition"), "mochil-annotations.json") || !strings.Contains(exported.Body.String(), `"quote": "Export me"`) {
		t.Fatalf("export: %d %s %s", exported.Code, exported.Header().Get("Content-Disposition"), exported.Body.String())
	}

	wrong := httptest.NewRequest(http.MethodDelete, "/api/annotations/account", strings.NewReader(`{"confirmation":"delete"}`))
	wrong.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, wrong)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("wrong confirmation returned %d", response.Code)
	}

	remove := httptest.NewRequest(http.MethodDelete, "/api/annotations/account", strings.NewReader(`{"confirmation":"DELETE ALL ANNOTATIONS"}`))
	remove.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, remove)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"deleted":1`) {
		t.Fatalf("delete: %d %s", response.Code, response.Body.String())
	}
	after := httptest.NewRecorder()
	handler.ServeHTTP(after, httptest.NewRequest(http.MethodGet, "/api/annotations/books/one", nil))
	if after.Code != http.StatusOK || strings.Contains(after.Body.String(), "mark-1") {
		t.Fatalf("annotations remain: %d %s", after.Code, after.Body.String())
	}
}

func TestAnnotationAPICreatesExplicitWikiNote(t *testing.T) {
	index := 3.0
	book := catalog.Book{ID: "one", Title: "One", Authors: []string{"A. Writer"}, Series: "Saga", SeriesIndex: &index, Path: "one.epub"}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	put := httptest.NewRequest(http.MethodPut, "/api/annotations/books/one", strings.NewReader(`{"changes":[{"base_revision":0,"annotation":{"id":"mark-1","kind":"highlight","quote":"A useful sentence","note":"Remember this","chapter":"Chapter 4"}}]}`))
	put.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), put)

	created := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/annotations/books/one/notes", strings.NewReader(`{}`))
	createRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(created, createRequest)
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"title":"Annotations — One"`) || !strings.Contains(created.Body.String(), `A useful sentence`) || !strings.Contains(created.Body.String(), `Remember this`) {
		t.Fatalf("created note: %d %s", created.Code, created.Body.String())
	}
	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, httptest.NewRequest(http.MethodGet, "/api/wiki", nil))
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"title":"Annotations — One"`) {
		t.Fatalf("listed notes: %d %s", listed.Code, listed.Body.String())
	}
}

func TestAnnotationAPINotesExportRejectsEmptyAndMissingPublications(t *testing.T) {
	book := catalog.Book{ID: "one", Title: "One", Path: "one.epub"}
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: []catalog.Book{book}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	empty := httptest.NewRecorder()
	emptyRequest := httptest.NewRequest(http.MethodPost, "/api/annotations/books/one/notes", strings.NewReader(`{}`))
	emptyRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(empty, emptyRequest)
	if empty.Code != http.StatusConflict {
		t.Fatalf("empty export returned %d: %s", empty.Code, empty.Body.String())
	}
	missing := httptest.NewRecorder()
	missingRequest := httptest.NewRequest(http.MethodPost, "/api/annotations/books/missing/notes", strings.NewReader(`{}`))
	missingRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(missing, missingRequest)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing export returned %d", missing.Code)
	}
}
