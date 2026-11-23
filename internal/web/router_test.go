package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Import is a basic need (bringing files into a library), not the optional spreadsheet editor
// that shares the /editor URL prefix. It must stay reachable even when a fresh install has never
// turned the editor module on, or nothing links to it and nobody finds it.
// Import lived in the retired shell and moved with it. It stays reachable — bringing files into a library is a basic
// need, not the optional spreadsheet editor that shared its prefix — and it is still not gated behind that module.
func TestEditorImportPageIsReachableWithoutTheEditorModule(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true, Modules: map[string]bool{"editor": false}}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	imp := httptest.NewRecorder()
	handler.ServeHTTP(imp, httptest.NewRequest(http.MethodGet, "/legacy/editor/import", nil))
	if imp.Code != http.StatusOK {
		t.Fatalf("/legacy/editor/import = %d, want 200", imp.Code)
	}
}

// The Import page is useless if the API it calls is still behind the same gate its
// HTML page was freed from — that half-fix is exactly what shipped once already.
func TestEditorImportAPIsAreReachableWithoutTheEditorModule(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true, Modules: map[string]bool{}}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"import", "import-names", "import-library"} {
		method := http.MethodGet
		if route != "import" {
			method = http.MethodPost
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(method, "/api/metadata/editor/"+route, strings.NewReader("{}")))
		if rec.Code == http.StatusNotFound {
			t.Fatalf("/api/metadata/editor/%s = 404 while the editor module is off, want it reachable", route)
		}
	}
	// A spreadsheet-only route stays gated.
	spreadsheet := httptest.NewRecorder()
	handler.ServeHTTP(spreadsheet, httptest.NewRequest(http.MethodGet, "/api/metadata/editor/books", nil))
	if spreadsheet.Code != http.StatusNotFound {
		t.Fatalf("/api/metadata/editor/books = %d, want 404 while the editor module is off", spreadsheet.Code)
	}
}
