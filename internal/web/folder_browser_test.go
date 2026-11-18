package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func folderTestHandler(t *testing.T, roots ...string) http.Handler {
	t.Helper()
	handler, err := NewMulti(MultiConfig{AllowOpen: true, BrowseRoots: roots}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func browse(handler http.Handler, path string) (*httptest.ResponseRecorder, folderListing) {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/folders?path="+url.QueryEscape(path), nil))
	var listing folderListing
	_ = json.Unmarshal(response.Body.Bytes(), &listing)
	return response, listing
}

func TestFolderBrowserListsOnlyFoldersInsideRoots(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	for _, name := range []string{"Books/Calibre", "Books/Light Novels", "Books/.hidden", "audio"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "Books", "note.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "Books", "escape")); err != nil {
		t.Fatal(err)
	}
	handler := folderTestHandler(t, root)

	response, listing := browse(handler, "")
	if response.Code != http.StatusOK || len(listing.Directories) != 1 || listing.Directories[0].Path != root {
		t.Fatalf("roots = %d %s", response.Code, response.Body.String())
	}
	response, listing = browse(handler, filepath.Join(root, "Books"))
	names := []string{}
	for _, directory := range listing.Directories {
		names = append(names, directory.Name)
	}
	if response.Code != http.StatusOK || len(names) != 2 || names[0] != "Calibre" || names[1] != "Light Novels" || listing.Parent != root {
		t.Fatalf("listing = %d %v parent=%q (files, hidden folders and escaping links must be absent)", response.Code, names, listing.Parent)
	}
	if _, listing = browse(handler, root); listing.Parent != "" {
		t.Fatalf("a root must not offer a parent: %+v", listing)
	}
}

func TestFolderBrowserRefusesEscapes(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	handler := folderTestHandler(t, root)
	for _, path := range []string{outside, filepath.Join(root, "escape"), filepath.Join(root, "..", filepath.Base(outside)), "/etc", "relative/path"} {
		if response, _ := browse(handler, path); response.Code < 400 {
			t.Fatalf("%q was listed with %d", path, response.Code)
		}
	}
	if response, _ := browse(handler, filepath.Join(root, "missing")); response.Code != http.StatusNotFound {
		t.Fatalf("missing folder = %d", response.Code)
	}
}
