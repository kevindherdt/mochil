package web

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"mochil/internal/document"
	"mochil/internal/vaultwriter"
)

type vaultMutationRequest struct {
	Path     string `json:"path"`
	From     string `json:"from"`
	Content  string `json:"content"`
	Revision string `json:"revision"`
}

func (m *multiServer) apiVaultMutation(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/vaults/"), "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	library, ok := m.library(parts[0])
	if !ok || library.Kind != "vault" || library.Documents == nil {
		http.NotFound(w, r)
		return
	}
	if library.VaultWriter == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Wiki editing is disabled for this library"})
		return
	}
	if r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "attachments" {
		m.vaultAttachment(w, r, library)
		return
	}
	var input vaultMutationRequest
	if err := decodeJSONRequest(w, r, vaultwriter.MaximumPageBytes+4096, &input, true); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Could not read that Wiki change"})
		return
	}
	var err error
	var revision string
	status := http.StatusCreated
	switch {
	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "folders":
		err = library.VaultWriter.CreateFolder(input.Path)
	case r.Method == http.MethodPatch && len(parts) == 2 && parts[1] == "folders":
		status = http.StatusOK
		err = library.VaultWriter.MoveFolder(input.From, input.Path)
	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "pages":
		_, err = library.VaultWriter.CreatePage(input.Path, []byte(input.Content))
	case r.Method == http.MethodPut && len(parts) == 3 && parts[1] == "pages":
		item, found := library.Documents.Item(parts[2])
		if !found || !document.IsMarkdown(item.Format) {
			http.NotFound(w, r)
			return
		}
		status = http.StatusOK
		revision, err = library.VaultWriter.UpdatePage(item.Path, input.Revision, []byte(input.Content))
	case r.Method == http.MethodPatch && len(parts) == 3 && parts[1] == "pages":
		item, found := library.Documents.Item(parts[2])
		if !found || !document.IsMarkdown(item.Format) {
			http.NotFound(w, r)
			return
		}
		status = http.StatusOK
		err = library.VaultWriter.MovePage(item.Path, input.Path, input.Revision)
	case r.Method == http.MethodDelete && len(parts) == 3 && parts[1] == "pages":
		item, found := library.Documents.Item(parts[2])
		if !found || !document.IsMarkdown(item.Format) {
			http.NotFound(w, r)
			return
		}
		status = http.StatusOK
		err = library.VaultWriter.TrashPage(item.Path, input.Revision)
	default:
		w.Header().Set("Allow", "POST, PUT, PATCH, DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err != nil {
		writeVaultMutationError(w, err)
		return
	}
	if _, err = library.Documents.Scan(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":           "Your Wiki change was saved, but the library index could not refresh. Refresh the library before making another change; do not repeat this operation.",
			"code":            "vault_index_refresh_failed",
			"committed":       true,
			"index_refreshed": false,
		})
		return
	}
	// A save returns the page's new revision so the editor can keep saving without reloading.
	writeJSON(w, status, map[string]any{"ok": true, "committed": true, "index_refreshed": true, "revision": revision})
}

func writeVaultMutationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, vaultwriter.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "This page changed in Obsidian or another browser. Reload it before saving."})
	case errors.Is(err, vaultwriter.ErrExists):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "A page or folder with that name already exists"})
	case errors.Is(err, vaultwriter.ErrInvalidPath):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Choose a valid location inside the vault"})
	case errors.Is(err, vaultwriter.ErrUnsupportedAttachment):
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "Only PNG, JPEG, GIF and WebP images can be added"})
	case errors.Is(err, fs.ErrPermission):
		slog.Error("vault write refused by the file system", "error", err)
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Mochil is not allowed to write to this vault folder. Run it as the folder's owner (set MOCHIL_UID and MOCHIL_GID) or make the folder writable for it. Nothing was changed."})
	default:
		slog.Error("vault write failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Mochil could not safely write that Wiki change"})
	}
}

// vaultAttachment stores a pasted or uploaded image: the file name comes in the
// query, the bytes in the body. The response says where the page can show it.
func (m *multiServer) vaultAttachment(w http.ResponseWriter, r *http.Request, library Library) {
	limitRequestBody(w, r, vaultwriter.MaximumAttachmentBytes+1, uploadBodyBudget)
	data, err := io.ReadAll(r.Body)
	if err != nil || len(data) > vaultwriter.MaximumAttachmentBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "That image is too large (12 MB at most)"})
		return
	}
	name, err := library.VaultWriter.CreateAttachment(r.URL.Query().Get("name"), data)
	if err != nil {
		writeVaultMutationError(w, err)
		return
	}
	if _, err = library.Documents.Scan(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "The image was saved, but the library index could not refresh. Refresh the library.",
			"code":  "vault_index_refresh_failed", "committed": true, "index_refreshed": false,
		})
		return
	}
	response := map[string]any{"ok": true, "name": name}
	for _, item := range library.Documents.Items() {
		if item.Path == name {
			response["url"] = documentViews(library, []document.Item{item}, nil)[0].OpenURL
		}
	}
	writeJSON(w, http.StatusCreated, response)
}
