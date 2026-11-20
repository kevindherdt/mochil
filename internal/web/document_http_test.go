package web

import (
	"archive/zip"
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/document"
)

func TestDocumentCataloguePagesSearchesAndServesFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "First note.md"), []byte("# First"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Second.txt"), []byte("Second"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Structured.json"), []byte(`{"project":"Bobo","status":"ready"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	writeWebTestDOCX(t, filepath.Join(root, "Report.docx"), `<w:document xmlns:w="urn:word"><w:body><w:p><w:r><w:t>Quarterly result</w:t></w:r></w:p></w:body></w:document>`)
	store := document.NewStore("notes", root, filepath.Join(t.TempDir(), "documents.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "notes", Name: "Notes", Kind: "documents", Documents: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents?library=notes&q=first&limit=1", nil))
	if response.Code != http.StatusOK || response.Header().Get("X-Total-Count") != "1" || !strings.Contains(response.Body.String(), `"title":"First note"`) {
		t.Fatalf("catalogue = %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents?library=notes&limit=1", nil))
	cursor := response.Header().Get("X-Next-Cursor")
	if response.Code != http.StatusOK || cursor == "" || !strings.Contains(response.Body.String(), `"title":"First note"`) {
		t.Fatalf("first cursor page = %d %#v %s", response.Code, response.Header(), response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents?library=notes&limit=1&cursor="+url.QueryEscape(cursor), nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"title":"Report"`) {
		t.Fatalf("second cursor page = %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents?library=notes&sort=recent&cursor="+url.QueryEscape(cursor), nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("cross-sort cursor status = %d", response.Code)
	}
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents?library=notes&cursor="+url.QueryEscape(cursor), nil))
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "reload") {
		t.Fatalf("expired cursor = %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents?library=notes&q=second", nil))
	if response.Code != http.StatusOK || response.Header().Get("X-Content-Indexed") != "4" || !strings.Contains(response.Body.String(), `"match_snippet":"Second"`) {
		t.Fatalf("content search = %d %#v %s", response.Code, response.Header(), response.Body.String())
	}
	item := store.Items()[0]
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/documents/notes/"+item.ID+"/note.md", nil))
	if response.Code != http.StatusOK || response.Body.String() != "# First" || response.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(response.Header().Get("Content-Disposition"), "inline") {
		t.Fatalf("open = %d %#v %q", response.Code, response.Header(), response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/documents/notes/"+item.ID+"/note.md?download=1", nil))
	if !strings.HasPrefix(response.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("download disposition = %q", response.Header().Get("Content-Disposition"))
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents/notes/"+item.ID+"/text", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"content":"# First"`) || response.Header().Get("ETag") == "" {
		t.Fatalf("text = %d %#v %s", response.Code, response.Header(), response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents?library=notes&q=quarterly", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"format":"docx"`) || !strings.Contains(response.Body.String(), `"text_url":"/api/documents/notes/`) {
		t.Fatalf("DOCX catalogue = %d %s", response.Code, response.Body.String())
	}
	var report document.Item
	for _, candidate := range store.Items() {
		if candidate.Format == "docx" {
			report = candidate
		}
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents/notes/"+report.ID+"/text", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"content":"Quarterly result"`) {
		t.Fatalf("DOCX text = %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents?library=notes&q=project", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"format":"json"`) || !strings.Contains(response.Body.String(), `"match_snippet":"{\"project\":\"Bobo\"`) {
		t.Fatalf("JSON search = %d %s", response.Code, response.Body.String())
	}
}

func TestVaultTreeReturnsCompleteReadOnlyHierarchy(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"Home.md":            "# Home",
		"Projects/Mochil.md": "[[Home]]",
		"Assets/cover.png":   "not-decoded-by-the-document-index",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := document.NewVaultStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "vault", Name: "Vault", Kind: "vault", Documents: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents?library=vault&tree=1", nil))
	if response.Code != http.StatusOK || response.Header().Get("X-Total-Count") != "3" || !strings.Contains(response.Body.String(), `"path":"Projects/Mochil.md"`) || !strings.Contains(response.Body.String(), `"path":"Assets/cover.png"`) {
		t.Fatalf("vault tree = %d total=%q body=%s", response.Code, response.Header().Get("X-Total-Count"), response.Body.String())
	}
	var cover document.Item
	for _, item := range store.Items() {
		if item.Path == "Assets/cover.png" {
			cover = item
		}
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/documents/vault/"+cover.ID+"/cover.png", nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "image/png" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("vault image = %d %#v %q", response.Code, response.Header(), response.Body.String())
	}
	var home document.Item
	for _, item := range store.Items() {
		if item.Path == "Home.md" {
			home = item
		}
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents/vault/"+home.ID+"/backlinks", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"path":"Projects/Mochil.md"`) {
		t.Fatalf("vault backlinks = %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents/vault/tags", nil))
	if response.Code != http.StatusOK || !strings.HasPrefix(strings.TrimSpace(response.Body.String()), "[") {
		t.Fatalf("vault tags = %d %s", response.Code, response.Body.String())
	}
}

func writeWebTestDOCX(t *testing.T, path, xml string) {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entry, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entry.Write([]byte(xml)); err != nil {
		t.Fatal(err)
	}
	if err = archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}
