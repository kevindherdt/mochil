package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mochil/internal/catalog"
	"mochil/internal/health"
)

func TestDetailedHealthIsReadOnlyAndDoesNotExposePaths(t *testing.T) {
	store := staticBackend{books: []catalog.Book{{ID: "one", Title: "One"}}}
	m := &multiServer{libraries: []Library{{ID: "books", Name: "Books", Kind: "books", SourcePath: "/secret/books", Store: store}}}
	m.health = health.New(m.healthSnapshot)
	w := httptest.NewRecorder()
	m.apiHealth(w, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var report health.Report
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Status != "ok" || len(report.Shelves) != 1 || report.Shelves[0].Items != 1 {
		t.Fatalf("report: %#v", report)
	}
	if w.Body.Len() == 0 || strings.Contains(w.Body.String(), "/secret/books") {
		t.Fatalf("health leaked source path: %s", w.Body.String())
	}
}
