package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/document"
	"mochil/internal/vaultwriter"
)

func TestVaultTaskSaveRejectsUnindexedExternalEdit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Tasks.md")
	original := "\ufeff---\r\ntitle: 日本語\r\n---\r\n- [ ] repeat\r\n- [ ] repeat\r\n"
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
	read := func() (string, string) {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/documents/vault/"+id+"/text", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("read status=%d", response.Code)
		}
		var page struct {
			Content  string `json:"content"`
			Revision string `json:"edit_revision"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		return page.Content, page.Revision
	}
	put := func(content, revision string, status int) {
		t.Helper()
		body, err := json.Marshal(map[string]string{"content": content, "revision": revision})
		if err != nil {
			t.Fatal(err)
		}
		requestJSONBody(t, handler, http.MethodPut, "/api/vaults/vault/pages/"+id, string(body), status)
	}
	loaded, revision := read()
	if loaded != original {
		t.Fatal("initial content changed")
	}
	// Simulate another editor modifying the same file without a catalogue rescan.
	external := original + "External edit must survive.\r\n"
	if err := os.WriteFile(path, []byte(external), 0600); err != nil {
		t.Fatal(err)
	}
	put(strings.Replace(original, "- [ ]", "- [x]", 1), revision, http.StatusConflict)
	stored, err := os.ReadFile(path)
	if err != nil || string(stored) != external {
		t.Fatalf("stale task overwrote external edit: %q, %v", stored, err)
	}
	fresh, nextRevision := read()
	if fresh != external || nextRevision == revision {
		t.Fatal("reload did not expose external revision")
	}
	offset := strings.LastIndex(fresh, "[ ]") + 1
	updated := fresh[:offset] + "x" + fresh[offset+1:]
	put(updated, nextRevision, http.StatusOK)
	stored, err = os.ReadFile(path)
	if err != nil || string(stored) != updated {
		t.Fatalf("task changed unrelated bytes: %q, %v", stored, err)
	}
}
