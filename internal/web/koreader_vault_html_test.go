package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/document"
)

func vaultLibrary(t *testing.T, id, name string) Library {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Grocery list.md"), []byte("Milk\n\n*Eggs*"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Longform note.markdown"), []byte("A note saved with the other Markdown extension.\n\n*Still* Markdown."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "image.png"), []byte("not actually a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := document.NewVaultStore(id, root, filepath.Join(t.TempDir(), "cache.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	return Library{ID: id, Name: name, Kind: "vault", Documents: store}
}

func TestKoReaderHomeListsVaultLibrariesButNotInTheBookShelf(t *testing.T) {
	server, token := browseTestServer(t)
	server.libraries = append(server.libraries, vaultLibrary(t, "wiki", "wiki"))
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/home", token)
	if response.Code != http.StatusOK {
		t.Fatalf("home = %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"vaults"`) || !strings.Contains(response.Body.String(), `"wiki"`) {
		t.Fatalf("home must list the vault library: %s", response.Body.String())
	}
	libraries := browseRequest(t, server, token, "kind=libraries")
	for _, group := range libraries.Groups {
		if group.Library == "wiki" {
			t.Fatalf("a vault library must never appear in the ordinary book library list: %+v", libraries)
		}
	}
}

func TestKoReaderVaultHTMLRendersMarkdownNotesButSkipsNonMarkdownFiles(t *testing.T) {
	server, token := browseTestServer(t)
	server.libraries = append(server.libraries, vaultLibrary(t, "wiki", "My notes"))
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/vault/wiki.html", token)
	if response.Code != http.StatusOK {
		t.Fatalf("vault.html = %d %s", response.Code, response.Body.String())
	}
	if ct := response.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content type = %q", ct)
	}
	body := response.Body.String()
	if !strings.Contains(body, "Grocery list") || !strings.Contains(body, "<em>Eggs</em>") {
		t.Fatalf("expected the rendered note: %s", body)
	}
	if !strings.Contains(body, "Longform note") || !strings.Contains(body, "<em>Still</em>") {
		t.Fatalf("a .markdown note is a Markdown note and belongs in the bundle: %s", body)
	}
	if strings.Contains(body, "image.png") {
		t.Fatalf("a non-markdown vault file must not appear in the bundle: %s", body)
	}
}

func TestKoReaderVaultHTMLRejectsAnUnknownOrNonVaultLibrary(t *testing.T) {
	server, token := browseTestServer(t)
	if response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/vault/does-not-exist.html", token); response.Code != http.StatusNotFound {
		t.Fatalf("unknown vault = %d", response.Code)
	}
	if response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/vault/books.html", token); response.Code != http.StatusNotFound {
		t.Fatalf("a books library is not a vault, want 404, got %d", response.Code)
	}
}
