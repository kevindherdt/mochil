package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
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

func TestVaultMutationCreatesAndConflictChecksMarkdown(t *testing.T) {
	root := t.TempDir()
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

	requestJSONBody(t, handler, http.MethodPost, "/api/vaults/vault/folders", `{"path":"Projects"}`, http.StatusCreated)
	requestJSONBody(t, handler, http.MethodPost, "/api/vaults/vault/pages", `{"path":"Projects/Plan.md","content":"---\ntags: [bobo]\n---\n# Plan\n[[Home]]"}`, http.StatusCreated)

	tree := httptest.NewRecorder()
	handler.ServeHTTP(tree, httptest.NewRequest(http.MethodGet, "/api/documents?library=vault&tree=1", nil))
	if tree.Code != http.StatusOK || tree.Header().Get("X-Vault-Writable") != "1" || !strings.Contains(tree.Body.String(), `"path":"Projects/Plan.md"`) {
		t.Fatalf("tree = %d %#v %s", tree.Code, tree.Header(), tree.Body.String())
	}
	item := store.Items()[0]
	textResponse := httptest.NewRecorder()
	handler.ServeHTTP(textResponse, httptest.NewRequest(http.MethodGet, "/api/documents/vault/"+item.ID+"/text", nil))
	var textPayload struct {
		Content      string `json:"content"`
		EditRevision string `json:"edit_revision"`
		Writable     bool   `json:"writable"`
	}
	if err := json.Unmarshal(textResponse.Body.Bytes(), &textPayload); err != nil {
		t.Fatal(err)
	}
	if !textPayload.Writable || textPayload.EditRevision == "" || !strings.Contains(textPayload.Content, "[[Home]]") {
		t.Fatalf("text payload = %+v", textPayload)
	}

	update := `{"revision":"` + textPayload.EditRevision + `","content":"# Updated\n[[Home]]"}`
	saved := httptest.NewRecorder()
	saveRequest := httptest.NewRequest(http.MethodPut, "/api/vaults/vault/pages/"+item.ID, bytes.NewBufferString(update))
	saveRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(saved, saveRequest)
	var savedPayload struct {
		Revision string `json:"revision"`
	}
	if saved.Code != http.StatusOK || json.Unmarshal(saved.Body.Bytes(), &savedPayload) != nil || savedPayload.Revision != vaultwriter.Revision([]byte("# Updated\n[[Home]]")) {
		t.Fatalf("save = %d %s; the response must carry the new revision so the editor can keep saving", saved.Code, saved.Body.String())
	}
	requestJSONBody(t, handler, http.MethodPut, "/api/vaults/vault/pages/"+item.ID, update, http.StatusConflict)
	content, err := os.ReadFile(filepath.Join(root, "Projects", "Plan.md"))
	if err != nil || string(content) != "# Updated\n[[Home]]" {
		t.Fatalf("saved content = %q, %v", content, err)
	}
	move := `{"revision":"` + vaultwriter.Revision(content) + `","path":"Plan renamed.md"}`
	requestJSONBody(t, handler, http.MethodPatch, "/api/vaults/vault/pages/"+item.ID, move, http.StatusOK)
	if moved, err := os.ReadFile(filepath.Join(root, "Plan renamed.md")); err != nil || string(moved) != string(content) {
		t.Fatalf("moved content = %q, %v", moved, err)
	}
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	movedItems := store.Items()
	if len(movedItems) != 1 {
		t.Fatalf("moved pages indexed = %d", len(movedItems))
	}
	movedItem := movedItems[0]
	remove := `{"revision":"` + vaultwriter.Revision(content) + `"}`
	requestJSONBody(t, handler, http.MethodDelete, "/api/vaults/vault/pages/"+movedItem.ID, remove, http.StatusOK)
	if trashed, err := os.ReadFile(filepath.Join(root, ".trash", "Plan renamed.md")); err != nil || string(trashed) != string(content) {
		t.Fatalf("trashed content = %q, %v", trashed, err)
	}
	requestJSONBody(t, handler, http.MethodPost, "/api/vaults/vault/pages", `{"path":".obsidian/plugins.md","content":"unsafe"}`, http.StatusBadRequest)
}

func TestAndroidDeviceTokenCanReadAndWriteVaultWithoutBrowserSession(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Welcome.md"), []byte("# Welcome"), 0o644); err != nil {
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
	devices, err := newAndroidDeviceStore(filepath.Join(t.TempDir(), "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := devices.issue("reader", "Phone")
	if err != nil {
		t.Fatal(err)
	}
	server := &multiServer{
		libraries:      []Library{{ID: "vault", Name: "Vault", Kind: "vault", Documents: store, VaultWriter: writer}},
		androidDevices: devices,
	}

	request := httptest.NewRequest(http.MethodGet, "/api/android/v1/documents?library=vault&tree=1", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.androidAPI(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"path":"Welcome.md"`) {
		t.Fatalf("tree = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/android/v1/vaults/vault/pages", strings.NewReader(`{"path":"Notes.md","content":"# Notes"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	server.androidAPI(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/android/v1/documents?library=vault&tree=1", nil)
	response = httptest.NewRecorder()
	server.androidAPI(response, request)
	if response.Code != http.StatusUnauthorized || response.Code == http.StatusSeeOther {
		t.Fatalf("unauthenticated tree = %d, want 401 without redirect", response.Code)
	}
}

func requestJSONBody(t *testing.T, handler http.Handler, method, path, body string, want int) {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != want {
		t.Fatalf("%s %s = %d, want %d: %s", method, path, response.Code, want, response.Body.String())
	}
}

func TestVaultPermissionFailureExplainsItself(t *testing.T) {
	response := httptest.NewRecorder()
	writeVaultMutationError(response, fmt.Errorf("prepare vault save: %w", fs.ErrPermission))
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "MOCHIL_UID") {
		t.Fatalf("permission failure = %d %s", response.Code, response.Body.String())
	}
	other := httptest.NewRecorder()
	writeVaultMutationError(other, errors.New("disk on fire"))
	if other.Code != http.StatusInternalServerError || strings.Contains(other.Body.String(), "disk on fire") {
		t.Fatalf("unexpected failures stay generic: %d %s", other.Code, other.Body.String())
	}
}

func TestVaultFolderRenameAndImageUpload(t *testing.T) {
	root := t.TempDir()
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
	requestJSONBody(t, handler, http.MethodPost, "/api/vaults/vault/folders", `{"path":"Old"}`, http.StatusCreated)
	requestJSONBody(t, handler, http.MethodPost, "/api/vaults/vault/pages", `{"path":"Old/A.md","content":"# A"}`, http.StatusCreated)
	requestJSONBody(t, handler, http.MethodPatch, "/api/vaults/vault/folders", `{"from":"Old","path":"New"}`, http.StatusOK)
	requestJSONBody(t, handler, http.MethodPatch, "/api/vaults/vault/folders", `{"from":"New","path":"New"}`, http.StatusBadRequest)
	if _, err := os.Stat(filepath.Join(root, "New", "A.md")); err != nil {
		t.Fatal(err)
	}

	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")
	upload := httptest.NewRecorder()
	handler.ServeHTTP(upload, httptest.NewRequest(http.MethodPost, "/api/vaults/vault/attachments?name=Pasted%20image.png", bytes.NewReader(png)))
	if upload.Code != http.StatusCreated || !strings.Contains(upload.Body.String(), `"name":"Pasted image.png"`) || !strings.Contains(upload.Body.String(), `"url":"/documents/vault/`) {
		t.Fatalf("upload = %d %s", upload.Code, upload.Body.String())
	}
	bad := httptest.NewRecorder()
	handler.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, "/api/vaults/vault/attachments?name=x.png", strings.NewReader("not an image")))
	if bad.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("fake image = %d %s", bad.Code, bad.Body.String())
	}
}
