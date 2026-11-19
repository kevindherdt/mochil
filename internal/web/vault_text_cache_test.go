package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/document"
)

func TestVaultTextRevalidatesExternalChangesWithoutScan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Note.md")
	if err := os.WriteFile(path, []byte("# First"), 0600); err != nil {
		t.Fatal(err)
	}
	store := document.NewVaultStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "vault", Name: "Vault", Kind: "vault", Documents: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	url := "/api/documents/vault/" + store.Items()[0].ID + "/text"
	get := func(etag string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, url, nil)
		r.Header.Set("If-None-Match", etag)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	first := get("")
	if first.Code != http.StatusOK || first.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("first: %d %v", first.Code, first.Header())
	}
	etag := first.Header().Get("ETag")
	unchanged := get(etag)
	if unchanged.Code != http.StatusNotModified || unchanged.Header().Get("ETag") != etag || unchanged.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unchanged: %d %v", unchanged.Code, unchanged.Header())
	}
	// Same-size replacement, deliberately without scanning the catalogue again.
	if err := os.WriteFile(path, []byte("# Other"), 0600); err != nil {
		t.Fatal(err)
	}
	changed := get(etag)
	if changed.Code != http.StatusOK || changed.Header().Get("ETag") == etag || !strings.Contains(changed.Body.String(), "# Other") {
		t.Fatalf("changed: %d %s", changed.Code, changed.Body.String())
	}
}
