package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mochil/internal/wiki"
)

type wikiRequest struct {
	Title            string `json:"title"`
	Folder           string `json:"folder"`
	Content          string `json:"content"`
	ExpectedRevision int64  `json:"expected_revision"`
	Revision         int64  `json:"revision"`
	Confirmation     string `json:"confirmation"`
}

const wikiTrashRetention = 7 * 24 * time.Hour

func (m *multiServer) wikiAPI(w http.ResponseWriter, r *http.Request) {
	if m.wiki == nil {
		http.Error(w, "Wiki is unavailable", http.StatusServiceUnavailable)
		return
	}
	account, suffix := m.readingAccount(r), strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/wiki"), "/")
	if suffix == "folders" {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, m.wiki.ListFolders(account))
		case http.MethodPost:
			var q struct {
				Folder string `json:"folder"`
			}
			if e := decodeJSONRequest(w, r, 2048, &q, true); e != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			folder, e := m.wiki.CreateFolder(account, q.Folder)
			if e != nil {
				writeWikiError(w, e)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]string{"folder": folder})
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if suffix == "" {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, m.wiki.List(account, r.URL.Query().Get("deleted") == "1"))
		case http.MethodPost:
			var q wikiRequest
			if !decodeWikiRequest(w, r, &q) {
				return
			}
			p, e := m.wiki.CreateInFolder(account, q.Title, q.Folder, q.Content)
			writeWikiResult(w, p, e, http.StatusCreated)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if suffix == "export" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		b, e := m.wiki.Export(account)
		if e != nil {
			http.Error(w, "Could not export wiki", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="mochil-wiki.json"`)
		_, _ = w.Write(b)
		return
	}
	parts := strings.Split(suffix, "/")
	id := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			p, ok := m.wiki.Get(account, id)
			if !ok {
				http.NotFound(w, r)
				return
			}
			writeJSON(w, http.StatusOK, p)
		case http.MethodPut:
			var q wikiRequest
			if !decodeWikiRequest(w, r, &q) {
				return
			}
			p, e := m.wiki.UpdateInFolder(account, id, q.Title, q.Folder, q.Content, q.ExpectedRevision)
			writeWikiResult(w, p, e, http.StatusOK)
		case http.MethodDelete:
			expected, e := strconv.ParseInt(r.URL.Query().Get("expected_revision"), 10, 64)
			if e != nil {
				http.Error(w, "expected_revision is required", http.StatusBadRequest)
				return
			}
			p, e := m.wiki.Delete(account, id, expected)
			writeWikiResult(w, p, e, http.StatusOK)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	switch {
	case parts[1] == "history" && len(parts) == 2 && r.Method == http.MethodGet:
		h, e := m.wiki.History(account, id)
		if e != nil {
			writeWikiError(w, e)
			return
		}
		writeJSON(w, http.StatusOK, h)
	case parts[1] == "backlinks" && len(parts) == 2 && r.Method == http.MethodGet:
		page, ok := m.wiki.Get(account, id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, m.wiki.Backlinks(account, page.Title))
	case parts[1] == "restore" && len(parts) == 2 && r.Method == http.MethodPost:
		var q wikiRequest
		if !decodeWikiRequest(w, r, &q) {
			return
		}
		p, e := m.wiki.Restore(account, id, q.ExpectedRevision)
		writeWikiResult(w, p, e, http.StatusOK)
	case parts[1] == "purge" && len(parts) == 2 && r.Method == http.MethodPost:
		var q wikiRequest
		if !decodeWikiRequest(w, r, &q) {
			return
		}
		page, ok := m.wiki.Get(account, id)
		if !ok || page.DeletedAt == nil {
			http.NotFound(w, r)
			return
		}
		if q.Confirmation != "PURGE "+page.Title {
			http.Error(w, "Type the exact purge confirmation", http.StatusBadRequest)
			return
		}
		if e := m.wiki.Purge(account, id, q.ExpectedRevision, wikiTrashRetention); e != nil {
			writeWikiError(w, e)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case parts[1] == "restore-version" && len(parts) == 2 && r.Method == http.MethodPost:
		var q wikiRequest
		if !decodeWikiRequest(w, r, &q) {
			return
		}
		p, e := m.wiki.RestoreVersion(account, id, q.Revision, q.ExpectedRevision)
		writeWikiResult(w, p, e, http.StatusOK)
	default:
		http.NotFound(w, r)
	}
}

func decodeWikiRequest(w http.ResponseWriter, r *http.Request, q *wikiRequest) bool {
	if e := decodeJSONRequest(w, r, (2<<20)+4096, q, true); e != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return false
	}
	return true
}
func writeWikiResult(w http.ResponseWriter, p wiki.Page, e error, status int) {
	if e != nil {
		writeWikiError(w, e)
		return
	}
	writeJSON(w, status, p)
}
func writeWikiError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, wiki.ErrNotFound):
		http.Error(w, e.Error(), http.StatusNotFound)
	case errors.Is(e, wiki.ErrConflict):
		http.Error(w, e.Error(), http.StatusConflict)
	case errors.Is(e, wiki.ErrInvalid):
		http.Error(w, e.Error(), http.StatusBadRequest)
	case errors.Is(e, wiki.ErrRetention):
		http.Error(w, "This page must remain in Trash for 7 days before permanent deletion.", http.StatusConflict)
	default:
		http.Error(w, "Could not save wiki", http.StatusInternalServerError)
	}
}
