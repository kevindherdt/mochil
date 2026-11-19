package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/wiki"
)

func TestKoReaderWikiHTMLRendersEveryPageAsReadableHTML(t *testing.T) {
	server, token := browseTestServer(t)
	store, err := wiki.Open(filepath.Join(t.TempDir(), "wiki.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.wiki = store
	if _, err := store.Create("reader", "Annotations — Dune", "# Dune\n\n> A great quote\n\nSome *thoughts*."); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create("reader", "Second page", "plain text"); err != nil {
		t.Fatal(err)
	}
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/wiki.html", token)
	if response.Code != http.StatusOK {
		t.Fatalf("wiki.html = %d %s", response.Code, response.Body.String())
	}
	if ct := response.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content-type = %q, want text/html", ct)
	}
	body := response.Body.String()
	if !strings.Contains(body, "<h1>Dune</h1>") {
		t.Fatalf("markdown heading was not rendered to HTML: %s", body)
	}
	if !strings.Contains(body, "<blockquote>") || !strings.Contains(body, "A great quote") {
		t.Fatalf("blockquote was not rendered: %s", body)
	}
	if !strings.Contains(body, "<em>thoughts</em>") {
		t.Fatalf("emphasis was not rendered: %s", body)
	}
	if !strings.Contains(body, "Second page") {
		t.Fatalf("the second page must also be included: %s", body)
	}
}

func TestKoReaderWikiHTMLRefusesWithoutWikiSupport(t *testing.T) {
	server, token := browseTestServer(t)
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/wiki.html", token)
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("wiki.html without a store = %d %s, want 501", response.Code, response.Body.String())
	}
}

func TestKoReaderWikiHTMLWithNoPagesSaysSo(t *testing.T) {
	server, token := browseTestServer(t)
	store, err := wiki.Open(filepath.Join(t.TempDir(), "wiki.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.wiki = store
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/wiki.html", token)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "No notes yet") {
		t.Fatalf("empty wiki.html = %d %s", response.Code, response.Body.String())
	}
}

func TestKoReaderWikiHTMLEscapesUntrustedContent(t *testing.T) {
	server, token := browseTestServer(t)
	store, err := wiki.Open(filepath.Join(t.TempDir(), "wiki.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.wiki = store
	if _, err := store.Create("reader", "<script>alert(1)</script>", "safe <b>not bold</b> text"); err != nil {
		t.Fatal(err)
	}
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/wiki.html", token)
	if response.Code != http.StatusOK {
		t.Fatalf("wiki.html = %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatalf("the page title must be escaped, not injected raw: %s", body)
	}
	_ = httptest.NewRecorder
}
