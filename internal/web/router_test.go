package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Import is a basic need (bringing files into a library), not the optional spreadsheet editor
// that shares the /editor URL prefix. It must stay reachable even when a fresh install has never
// turned the editor module on, or nothing links to it and nobody finds it.
func TestEditorImportPageIsReachableWithoutTheEditorModule(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true, Modules: map[string]bool{}}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	imp := httptest.NewRecorder()
	handler.ServeHTTP(imp, httptest.NewRequest(http.MethodGet, "/editor/import", nil))
	if imp.Code != http.StatusOK {
		t.Fatalf("/editor/import = %d, want 200", imp.Code)
	}
	// The spreadsheet editor itself stays an opt-in module; only Import was un-gated.
	spreadsheet := httptest.NewRecorder()
	handler.ServeHTTP(spreadsheet, httptest.NewRequest(http.MethodGet, "/editor", nil))
	if spreadsheet.Code != http.StatusNotFound {
		t.Fatalf("/editor = %d, want 404 while the editor module is off", spreadsheet.Code)
	}
}
