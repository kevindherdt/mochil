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

func TestVaultCreateReportsCommittedWhenIndexPersistenceFails(t *testing.T) {
	root := t.TempDir()
	// A directory at the cache-file path deterministically rejects cache replacement.
	cache := t.TempDir()
	store := document.NewVaultStore("vault", root, cache)
	writer, err := vaultwriter.New(root)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "vault", Name: "Vault", Kind: "vault", Documents: store, VaultWriter: writer}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/vaults/vault/pages", strings.NewReader(`{"path":"Saved.md","content":"# Saved"}`)))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var outcome struct {
		Committed      bool   `json:"committed"`
		IndexRefreshed bool   `json:"index_refreshed"`
		Code           string `json:"code"`
		Error          string `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &outcome); err != nil {
		t.Fatal(err)
	}
	if !outcome.Committed || outcome.IndexRefreshed || outcome.Code != "vault_index_refresh_failed" || !strings.Contains(outcome.Error, "do not repeat") {
		t.Fatalf("outcome=%+v", outcome)
	}
	content, err := os.ReadFile(filepath.Join(root, "Saved.md"))
	if err != nil || string(content) != "# Saved" {
		t.Fatalf("content=%q err=%v", content, err)
	}
	// A genuine failed write must not claim a commit, even with the same index fault.
	conflict := httptest.NewRecorder()
	handler.ServeHTTP(conflict, httptest.NewRequest(http.MethodPost, "/api/vaults/vault/pages", strings.NewReader(`{"path":"Saved.md","content":"replacement"}`)))
	if conflict.Code != http.StatusConflict || strings.Contains(conflict.Body.String(), `"committed":true`) {
		t.Fatalf("conflict=%d %s", conflict.Code, conflict.Body.String())
	}
}

func TestVaultMutationOutcomesAcrossOperations(t *testing.T) {
	for _, operation := range []string{"folder", "update", "move", "trash"} {
		for _, failIndex := range []bool{false, true} {
			name := operation + "/index-ok"
			if failIndex {
				name = operation + "/index-failed"
			}
			t.Run(name, func(t *testing.T) {
				root := t.TempDir()
				cache := filepath.Join(t.TempDir(), "index.json")
				writer, err := vaultwriter.New(root)
				if err != nil {
					t.Fatal(err)
				}
				original := []byte("# Original")
				revision, err := writer.CreatePage("Original.md", original)
				if err != nil {
					t.Fatal(err)
				}
				store := document.NewVaultStore("vault", root, cache)
				if _, err := store.Scan(); err != nil {
					t.Fatal(err)
				}
				item := store.Items()[0]
				handler, err := NewMulti(MultiConfig{AllowOpen: true}, []Library{{ID: "vault", Name: "Vault", Kind: "vault", Documents: store, VaultWriter: writer}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
				if err != nil {
					t.Fatal(err)
				}
				if failIndex {
					if err := os.Remove(cache); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(cache, 0o700); err != nil {
						t.Fatal(err)
					}
				}
				method, endpoint := http.MethodPut, "/api/vaults/vault/pages/"+item.ID
				input := vaultMutationRequest{Revision: revision, Content: "# Updated"}
				resultPath, expected := "Original.md", "# Updated"
				status := http.StatusOK
				switch operation {
				case "folder":
					method, endpoint = http.MethodPost, "/api/vaults/vault/folders"
					input.Path, resultPath = "Projects", "Projects"
					status = http.StatusCreated
				case "move":
					method, input.Path = http.MethodPatch, "Moved.md"
					resultPath, expected = "Moved.md", string(original)
				case "trash":
					method = http.MethodDelete
					resultPath, expected = ".trash/Original.md", string(original)
				}
				body, err := json.Marshal(input)
				if err != nil {
					t.Fatal(err)
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(method, endpoint, strings.NewReader(string(body))))
				if failIndex {
					status = http.StatusInternalServerError
				}
				var outcome struct {
					OK             bool `json:"ok"`
					Committed      bool `json:"committed"`
					IndexRefreshed bool `json:"index_refreshed"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &outcome); err != nil {
					t.Fatal(err)
				}
				if response.Code != status || !outcome.Committed || outcome.IndexRefreshed != !failIndex || outcome.OK != !failIndex {
					t.Fatalf("status=%d outcome=%+v body=%s", response.Code, outcome, response.Body.String())
				}
				if operation == "folder" {
					info, err := os.Stat(filepath.Join(root, resultPath))
					if err != nil || !info.IsDir() {
						t.Fatalf("folder=%v err=%v", info, err)
					}
				} else {
					content, err := os.ReadFile(filepath.Join(root, resultPath))
					if err != nil || string(content) != expected {
						t.Fatalf("content=%q err=%v", content, err)
					}
				}
				if operation == "move" || operation == "trash" {
					if _, err := os.Stat(filepath.Join(root, "Original.md")); !os.IsNotExist(err) {
						t.Fatalf("old path still present: %v", err)
					}
				}
			})
		}
	}
}
