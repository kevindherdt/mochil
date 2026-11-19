package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mochil/internal/document"
	"mochil/internal/vaultwriter"
)

func TestVaultTextRevisionPreservesOriginalEncoding(t *testing.T) {
	for name, original := range map[string]string{
		"crlf":  "---\r\ntitle: Note\r\n---\r\n# Note\r\nText\r\n",
		"bom":   "\xef\xbb\xbf# Note\r\nText\r\n",
		"mixed": "# Note\r\nText\n日本語\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "Note.md")
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			store := document.NewVaultStore("vault", root, "")
			if _, err := store.Scan(); err != nil {
				t.Fatal(err)
			}
			writer, err := vaultwriter.New(root)
			if err != nil {
				t.Fatal(err)
			}
			handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "vault", Name: "Vault", Kind: "vault", Documents: store, VaultWriter: writer}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			if err != nil {
				t.Fatal(err)
			}
			id := store.Items()[0].ID
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents/vault/"+id+"/text", nil))
			var page struct {
				Content  string `json:"content"`
				Revision string `json:"edit_revision"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
				t.Fatal(err)
			}
			if page.Content != original || page.Revision != vaultwriter.Revision([]byte(original)) {
				t.Fatalf("page changed during read: %+v", page)
			}
			updated := page.Content + "More\r\n"
			body, err := json.Marshal(map[string]string{"revision": page.Revision, "content": updated})
			if err != nil {
				t.Fatal(err)
			}
			requestJSONBody(t, handler, http.MethodPut, "/api/vaults/vault/pages/"+id, string(body), http.StatusOK)
			stored, err := os.ReadFile(path)
			if err != nil || string(stored) != updated {
				t.Fatalf("round trip: %q, %v", stored, err)
			}
		})
	}
}

// Kevin's call: a .markdown note is editable in the wiki editor like a .md note. Both the exact-byte
// read (which keeps the editor's revision honest) and the page write go through format gates.
func TestVaultMarkdownExtensionIsEditableInTheWikiEditor(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Longform.markdown")
	original := "---\r\ntitle: Longform\r\n---\r\n# Longform\r\nText\r\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	store := document.NewVaultStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	writer, err := vaultwriter.New(root)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "vault", Name: "Vault", Kind: "vault", SourceType: "filesystem", Documents: store, VaultWriter: writer}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	id := store.Items()[0].ID
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents/vault/"+id+"/text", nil))
	var page struct {
		Content      string `json:"content"`
		EditRevision string `json:"edit_revision"`
		Writable     bool   `json:"writable"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Content != original || page.EditRevision != vaultwriter.Revision([]byte(original)) || !page.Writable {
		t.Fatalf("a .markdown note must open editable with exact bytes: %+v", page)
	}
	updated := page.Content + "More\r\n"
	body, err := json.Marshal(map[string]string{"revision": page.EditRevision, "content": updated})
	if err != nil {
		t.Fatal(err)
	}
	requestJSONBody(t, handler, http.MethodPut, "/api/vaults/vault/pages/"+id, string(body), http.StatusOK)
	stored, err := os.ReadFile(path)
	if err != nil || string(stored) != updated {
		t.Fatalf("round trip: %q, %v", stored, err)
	}
}
