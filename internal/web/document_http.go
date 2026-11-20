package web

import (
	"errors"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"mochil/internal/document"
	catalogquery "mochil/internal/query"
	"mochil/internal/vaultwriter"
)

type documentView struct {
	document.Item
	LibraryName  string `json:"library_name"`
	OpenURL      string `json:"open_url"`
	DownloadURL  string `json:"download_url"`
	TextURL      string `json:"text_url,omitempty"`
	MatchSnippet string `json:"match_snippet,omitempty"`
}

type vaultTreeView struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

func (m *multiServer) apiDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok || library.Documents == nil {
		http.Error(w, "Unknown document library", http.StatusNotFound)
		return
	}
	if r.URL.Query().Get("tree") == "1" {
		if library.Kind != "vault" {
			http.Error(w, "Folder trees are available only for vault libraries", http.StatusBadRequest)
			return
		}
		items := library.Documents.Items()
		if len(items) > 20000 {
			http.Error(w, "Vault contains too many pages for one folder tree", http.StatusRequestEntityTooLarge)
			return
		}
		w.Header().Set("X-Total-Count", strconv.Itoa(len(items)))
		if library.VaultWriter != nil && m.mayEditVaults(r) {
			w.Header().Set("X-Vault-Writable", "1")
		}
		views := make([]any, 0, len(items)+len(library.Documents.Directories()))
		for _, directory := range library.Documents.Directories() {
			views = append(views, vaultTreeView{Kind: "folder", Path: directory})
		}
		for _, item := range documentViews(library, items, nil) {
			views = append(views, item)
		}
		writeJSON(w, http.StatusOK, views)
		return
	}
	limit := 120
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 && value <= catalogquery.MaxPageSize {
		limit = value
	}
	offset := 0
	if value, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && value > 0 {
		offset = value
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if len(query) > 200 {
		http.Error(w, "Document search is limited to 200 characters", http.StatusBadRequest)
		return
	}
	sortName := "title"
	if r.URL.Query().Get("sort") == "recent" {
		sortName = "recent"
	}
	cursor, cursorErr := decodeDocumentCursor(r.URL.Query().Get("cursor"))
	if cursorErr != nil || (cursor.ItemID != "" && (query != "" || cursor.Sort != sortName)) {
		http.Error(w, "Document cursor is invalid for this view", http.StatusBadRequest)
		return
	}
	if query == "" {
		page, total, version, found, hasMore := library.Documents.Page(sortName, cursor.ItemID, offset, limit)
		if cursor.ItemID != "" && (cursor.Version != version || !found) {
			http.Error(w, "Document cursor expired; reload the library", http.StatusConflict)
			return
		}
		w.Header().Set("X-Total-Count", strconv.Itoa(total))
		if hasMore {
			w.Header().Set("X-Next-Cursor", encodeDocumentCursor(documentCursor{Version: version, Sort: sortName, ItemID: page[len(page)-1].ID}))
		}
		writeJSON(w, http.StatusOK, documentViews(library, page, nil))
		return
	}
	items := library.Documents.Items()
	contentMatches := map[string]string{}
	if query != "" {
		var coverage document.TextCoverage
		var searchErr error
		contentMatches, coverage, searchErr = library.Documents.SearchText(query)
		if searchErr != nil {
			http.Error(w, "Could not search document contents", http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-Content-Indexed", strconv.Itoa(coverage.Indexed))
		w.Header().Set("X-Content-Eligible", strconv.Itoa(coverage.Eligible))
		if !coverage.Complete {
			w.Header().Set("X-Content-Search-Partial", "1")
		}
	}
	filtered := make([]document.Item, 0, len(items))
	for _, item := range items {
		if query == "" || strings.Contains(strings.ToLower(item.Title), query) || strings.Contains(strings.ToLower(item.Path), query) || contentMatches[item.ID] != "" {
			filtered = append(filtered, item)
		}
	}
	if sortName == "recent" {
		sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].Modified.After(filtered[j].Modified) })
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(len(filtered)))
	if offset > len(filtered) {
		offset = len(filtered)
	}
	end := min(offset+limit, len(filtered))
	writeJSON(w, http.StatusOK, documentViews(library, filtered[offset:end], contentMatches))
}

func documentViews(library Library, items []document.Item, contentMatches map[string]string) []documentView {
	out := make([]documentView, 0, len(items))
	for _, item := range items {
		base := "/documents/" + url.PathEscape(library.ID) + "/" + url.PathEscape(item.ID) + "/" + url.PathEscape(filepath.Base(item.Path))
		textURL := ""
		if document.TextReadable(item.Format) {
			textURL = "/api/documents/" + url.PathEscape(library.ID) + "/" + url.PathEscape(item.ID) + "/text"
		}
		out = append(out, documentView{Item: item, LibraryName: library.Name, OpenURL: base, DownloadURL: base + "?download=1", TextURL: textURL, MatchSnippet: contentMatches[item.ID]})
	}
	return out
}

func (m *multiServer) apiDocumentText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/documents/"), "/")
	if len(parts) == 2 && (parts[1] == "tags" || parts[1] == "graph" || parts[1] == "properties") {
		library, ok := m.library(parts[0])
		if !ok || library.Documents == nil || library.Kind != "vault" {
			http.NotFound(w, r)
			return
		}
		if parts[1] == "graph" {
			graph, err := library.Documents.Graph()
			if err != nil {
				http.Error(w, "Could not inspect vault links", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, graph)
			return
		}
		if parts[1] == "properties" {
			properties, err := library.Documents.Properties()
			if err != nil {
				http.Error(w, "Could not inspect vault properties", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, properties)
			return
		}
		groups, err := library.Documents.Tags()
		if err != nil {
			http.Error(w, "Could not inspect vault tags", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, groups)
		return
	}
	if len(parts) != 3 || parts[2] != "text" && parts[2] != "backlinks" {
		http.NotFound(w, r)
		return
	}
	library, ok := m.library(parts[0])
	if !ok || library.Documents == nil {
		http.NotFound(w, r)
		return
	}
	if parts[2] == "backlinks" {
		item, found := library.Documents.Item(parts[1])
		if !found || library.Kind != "vault" || !document.IsMarkdown(item.Format) {
			http.NotFound(w, r)
			return
		}
		backlinks, err := library.Documents.Backlinks(item)
		if err != nil {
			http.Error(w, "Could not inspect vault backlinks", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "private, max-age=60")
		writeJSON(w, http.StatusOK, documentViews(library, backlinks, nil))
		return
	}
	content, item, err := library.Documents.ReadText(parts[1], document.DefaultTextLimit)
	if err != nil {
		switch {
		case errors.Is(err, document.ErrTextUnsupported):
			http.Error(w, "This document opens in the native viewer", http.StatusUnsupportedMediaType)
		case errors.Is(err, document.ErrTextTooLarge):
			http.Error(w, "This text document is too large for the web reader; download it instead", http.StatusRequestEntityTooLarge)
		case errors.Is(err, document.ErrTextEncoding):
			http.Error(w, "This text document is not UTF-8; download it instead", http.StatusUnprocessableEntity)
		case errors.Is(err, document.ErrTextMalformed):
			http.Error(w, "This document could not be read safely; download the original instead", http.StatusUnprocessableEntity)
		default:
			http.NotFound(w, r)
		}
		return
	}
	etag := `"` + item.Revision + `"`
	cacheControl := "private, max-age=300"
	if library.Kind == "vault" && document.IsMarkdown(item.Format) {
		// Vault files can change outside Mochil between catalogue scans.
		etag = `"` + item.Revision + "-" + vaultwriter.Revision(content) + "-" + strconv.FormatBool(library.VaultWriter != nil) + `"`
		cacheControl = "no-store"
	}
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	response := map[string]any{"id": item.ID, "title": item.Title, "format": item.Format, "revision": item.Revision, "content": string(content)}
	if library.Kind == "vault" && document.IsMarkdown(item.Format) {
		response["edit_revision"] = vaultwriter.Revision(content)
		response["writable"] = library.VaultWriter != nil && m.mayEditVaults(r)
	}
	writeJSON(w, http.StatusOK, response)
}

func (m *multiServer) serveDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/documents/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	library, ok := m.library(parts[0])
	if !ok || library.Documents == nil {
		http.NotFound(w, r)
		return
	}
	file, item, err := library.Documents.Open(parts[1])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Type", item.MediaType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": filepath.Base(item.Path)}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, filepath.Base(item.Path), item.Modified, file)
}
