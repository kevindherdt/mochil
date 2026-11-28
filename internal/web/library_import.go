package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mochil/internal/catalog"
	"mochil/internal/ingest"
	settingssvc "mochil/internal/settings"
)

const importScanPages = 60

// editorImport lists the books waiting in the incoming folder, each compared with the library they would join. With a
// status filter it keeps reading pages until it has a full page of matches, so "only new" does not mean fifty requests.
func (m *multiServer) editorImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.managedIngest == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "items": []importRow{}})
		return
	}
	query := r.URL.Query()
	naming := query.Get("naming")
	if !ingest.ValidNamingPreset(naming) {
		http.Error(w, "Naming must be default, calibre or kavita", http.StatusBadRequest)
		return
	}
	var books []catalog.Book
	if id := query.Get("library"); id != "" {
		lib, ok := m.library(id)
		if !ok || lib.Store == nil || !publicationKind(lib.Kind) {
			http.Error(w, "That library has no books to compare with.", http.StatusNotFound)
			return
		}
		books = lib.Store.Books()
	}
	// Built once for the whole request: the page loop below can walk sixty discovery pages, and each one used to rebuild four
	// maps over the entire library before matching a hundred rows.
	matches := buildImportIndex(books)
	wanted := map[string]bool{}
	for _, status := range strings.Split(query.Get("status"), ",") {
		if status = strings.TrimSpace(status); status != "" {
			wanted[status] = true
		}
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 || limit > ingest.MaximumDiscoveryItems {
		limit = ingest.MaximumDiscoveryItems
	}
	cursor := query.Get("cursor")
	items := make([]importRow, 0, limit)
	scanned := 0
	truncated := false
	for page := 0; page < importScanPages && len(items) < limit; page++ {
		found, err := m.managedIngest.DiscoverNamed(r.Context(), ingest.Publication, cursor, ingest.MaximumDiscoveryItems, naming)
		if err != nil {
			http.Error(w, "Could not inspect the incoming folder", http.StatusBadRequest)
			return
		}
		scanned += len(found.Items)
		// A drop folder larger than the discovery walk's visit cap is a partial answer, and saying so is the difference
		// between "these are your books" and "these are the books I could see".
		truncated = truncated || found.Truncated
		for _, row := range matches.match(found.Items) {
			row.Copied = m.managedIngest.Imported().Has(row.SourcePath)
			if len(wanted) == 0 || wanted[row.Status] {
				items = append(items, row)
			}
		}
		cursor = found.NextCursor
		if cursor == "" {
			break
		}
	}
	next := cursor
	if len(items) > limit {
		// a page of discovery can overshoot the limit; resume right after the last row that is returned
		next = items[limit-1].SourcePath
		items = items[:limit]
	}
	reply := map[string]any{"enabled": true, "items": items, "next_cursor": next, "scanned": scanned, "truncated": truncated, "incoming_directory": m.managedIngest.SourceRoot(), "managed_directory": m.managedIngest.TargetRoot(), "destinations": m.importDestinations()}
	if library, found := m.libraryHolding(m.managedIngest.TargetRoot()); found {
		reply["managed_library"] = map[string]string{"id": library.ID, "name": library.Name}
	}
	writeJSON(w, http.StatusOK, reply)
}

// editorImportNames works out the managed-library path for books whose fields were edited on the import page, using
// the same layouts as discovery. A book that cannot be named comes back with an empty target.
func (m *multiServer) editorImportNames(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Naming string `json:"naming"`
		Items  []struct {
			SourcePath  string   `json:"source_path"`
			Title       string   `json:"title"`
			Authors     []string `json:"authors"`
			Series      string   `json:"series"`
			SeriesIndex *float64 `json:"series_index"`
		} `json:"items"`
	}
	if decodeJSONRequest(w, r, editorMaximumBody, &input, true) != nil || len(input.Items) == 0 || len(input.Items) > ingest.MaximumDiscoveryItems || !ingest.ValidNamingPreset(input.Naming) {
		http.Error(w, "Send between 1 and 100 books and a known layout.", http.StatusBadRequest)
		return
	}
	type named struct {
		SourcePath string `json:"source_path"`
		Target     string `json:"target"`
	}
	out := make([]named, 0, len(input.Items))
	for _, item := range input.Items {
		// ok is false for a book that cannot be named, and a target plan would refuse must not be shown as the name a book
		// is going to get — both come back empty, which the screen reads as "needs a look".
		target, ok := ingest.TargetFor(input.Naming, item.SourcePath, ingest.Proposal{Kind: ingest.Publication, Title: item.Title, Authors: item.Authors, Series: item.Series, SeriesIndex: item.SeriesIndex})
		if !ok || !ingest.ValidManagedPath(target) {
			target = ""
		}
		out = append(out, named{SourcePath: item.SourcePath, Target: target})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// libraryHolding finds the configured library whose folder contains a path, so the import page can say whether the
// books it copies will show up in Mochil.
func (m *multiServer) libraryHolding(path string) (Library, bool) {
	path = filepath.Clean(path)
	for _, lib := range m.libraries {
		if lib.SourcePath == "" || !publicationKind(lib.Kind) {
			continue
		}
		root := filepath.Clean(lib.SourcePath)
		if relative, err := filepath.Rel(root, path); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return lib, true
		}
	}
	return Library{}, false
}

type importDestination struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var importLibraryID = regexp.MustCompile(`[^a-z0-9]+`)

// editorImportLibrary makes a new, empty library inside the managed folder and adds it to the configuration, the same
// way the Settings page does. It takes effect after a restart, like any library change.
func (m *multiServer) editorImportLibrary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.managedIngest == nil || !m.settingsService.Available() {
		http.Error(w, "Libraries cannot be added here: managed ingest or the config file is not available.", http.StatusConflict)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if decodeJSONRequest(w, r, 4096, &input, true) != nil {
		http.Error(w, "The request could not be read.", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(input.Name)
	id := strings.Trim(importLibraryID.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if name == "" || len([]rune(name)) > 60 || id == "" || len(id) > 40 {
		http.Error(w, "Give the library a short name.", http.StatusBadRequest)
		return
	}
	m.metadataApplyMu.Lock()
	defer m.metadataApplyMu.Unlock()
	data, err := m.settingsService.Read()
	if err != nil {
		http.Error(w, "The settings could not be read.", http.StatusInternalServerError)
		return
	}
	var document map[string]any
	if json.Unmarshal(data, &document) != nil {
		http.Error(w, "The settings could not be read.", http.StatusInternalServerError)
		return
	}
	libraries, _ := document["libraries"].([]any)
	for _, existing := range libraries {
		entry, _ := existing.(map[string]any)
		if strings.EqualFold(fmt.Sprint(entry["id"]), id) || strings.EqualFold(fmt.Sprint(entry["name"]), name) {
			http.Error(w, "There is already a library with that name.", http.StatusConflict)
			return
		}
	}
	folder := filepath.Join(m.managedIngest.TargetRoot(), id)
	if err := os.Mkdir(folder, 0o755); err != nil {
		// Only an existing folder is the reader's problem. A permission, read-only or full-disk failure used to be reported as
		// "that folder already exists", which sent an admin looking for a name collision that was not there.
		if errors.Is(err, fs.ErrExist) {
			http.Error(w, "That folder already exists in the managed folder, so it was not used.", http.StatusConflict)
			return
		}
		slog.Error("could not create the managed folder", "folder", folder, "error", err)
		http.Error(w, "The folder could not be created.", http.StatusInternalServerError)
		return
	}
	document["libraries"] = append(libraries, map[string]any{
		"id": id, "name": name, "kind": "books",
		"source":            map[string]any{"type": "filesystem", "path": folder},
		"inference":         map[string]any{"filename_author": false, "folder_layout": "auto"},
		"preferred_formats": []string{"epub", "pdf", "mobi", "azw3"},
	})
	updated, err := json.Marshal(document)
	var applied settingssvc.ApplyResult
	if err == nil {
		applied, err = m.settingsService.Update(updated)
	}
	if err != nil {
		_ = os.Remove(folder)
		http.Error(w, "The library could not be added: "+err.Error(), http.StatusBadRequest)
		return
	}
	// A running server applies the change at once; only a setup that cannot (no live apply) needs a restart.
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": name, "folder": id, "restart_required": len(applied.RestartRequired) > 0})
}
