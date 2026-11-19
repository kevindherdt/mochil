package web

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLibraryScanQueuesConfiguredLibrary(t *testing.T) {
	var selected string
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ScanLibrary: func(id string) (string, error) {
		selected = id
		return "library-scan:manga:abc", nil
	}}, []Library{{ID: "manga", Name: "Manga", Kind: "manga", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/library-scan", bytes.NewBufferString(`{"library_id":"manga"}`)))
	if response.Code != http.StatusAccepted || selected != "manga" || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"queued"`)) {
		t.Fatalf("scan: %d %s selected=%q", response.Code, response.Body.String(), selected)
	}
}

func TestLibraryScanQueuesAllConfiguredLibraries(t *testing.T) {
	var selected []string
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ScanLibrary: func(id string) (string, error) {
		selected = append(selected, id)
		return "library-scan:" + id + ":abc", nil
	}}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}, {ID: "manga", Name: "Manga", Kind: "manga", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/library-scan", bytes.NewBufferString(`{"all":true}`)))
	if response.Code != http.StatusAccepted || len(selected) != 2 || selected[0] != "books" || selected[1] != "manga" || !bytes.Contains(response.Body.Bytes(), []byte(`"job_keys"`)) {
		t.Fatalf("scan all: %d %s selected=%v", response.Code, response.Body.String(), selected)
	}
}
