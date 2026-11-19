package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/catalog"
)

func TestReadiumEntriesAreServedInertWhenOpenedDirectly(t *testing.T) {
	root := t.TempDir()
	writeTestEPUB(t, filepath.Join(root, "One.epub"), `<p>A chapter</p><script>document.title = "ran"</script>`)
	store := catalog.NewFilesystemStore(root, filepath.Join(t.TempDir(), "cache.json"), catalog.FilesystemOptions{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	book := store.Books()[0]
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []string{"c0.xhtml", "manifest.json"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/readium/books/"+book.ID+"/"+entry, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s: status %d", entry, response.Code)
		}
		policy := response.Header().Get("Content-Security-Policy")
		if !strings.Contains(policy, "sandbox") || !strings.Contains(policy, "script-src 'none'") {
			t.Fatalf("%s: policy %q does not keep scripts from running", entry, policy)
		}
	}
}
