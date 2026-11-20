package web

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"mochil/internal/annotationnotes"
	"mochil/internal/annotations"
)

const maximumAnnotationRequestBytes = 4 << 20

type annotationChangeRequest struct {
	Changes []annotations.Change `json:"changes"`
}

type annotationDeleteRequest struct {
	Confirmation string `json:"confirmation"`
}

type annotationNotesRequest struct{}

func (m *multiServer) annotationAPI(w http.ResponseWriter, r *http.Request) {
	if m.annotations == nil {
		http.Error(w, "Annotation sync is unavailable", http.StatusNotImplemented)
		return
	}
	if r.URL.Path == "/api/annotations/export" {
		m.exportAnnotations(w, r, m.readingAccount(r))
		return
	}
	if r.URL.Path == "/api/annotations/account" {
		m.deleteAnnotations(w, r, m.readingAccount(r))
		return
	}
	if strings.HasSuffix(r.URL.Path, "/notes") {
		libraryID, bookID, pathOK := annotationPath(strings.TrimSuffix(r.URL.Path, "/notes"), "/api/annotations/")
		if !pathOK {
			http.NotFound(w, r)
			return
		}
		m.annotationsToNotes(w, r, m.readingAccount(r), libraryID, bookID)
		return
	}
	libraryID, bookID, ok := annotationPath(r.URL.Path, "/api/annotations/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, _, found := m.findBook(libraryID + "/" + bookID); !found {
		http.NotFound(w, r)
		return
	}
	m.serveAnnotations(w, r, m.readingAccount(r), libraryID, bookID)
}

func (m *multiServer) annotationsToNotes(w http.ResponseWriter, r *http.Request, account, libraryID, bookID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.wiki == nil {
		http.Error(w, "Notes are unavailable", http.StatusServiceUnavailable)
		return
	}
	var request annotationNotesRequest
	if decodeJSONRequest(w, r, 1024, &request, false) != nil {
		return
	}
	_, book, found := m.findBook(libraryID + "/" + bookID)
	if !found {
		http.NotFound(w, r)
		return
	}
	result := m.annotations.List(account, libraryID, bookID, 0)
	publication := annotationnotes.Publication{Title: book.Title, Authors: append([]string(nil), book.Authors...), Series: book.Series, SeriesIndex: book.SeriesIndex}
	note, err := annotationnotes.Render(publication, result.Applied)
	if errors.Is(err, annotationnotes.ErrEmpty) {
		http.Error(w, "This publication has no annotations to export", http.StatusConflict)
		return
	}
	if errors.Is(err, annotationnotes.ErrTooLarge) {
		http.Error(w, "These annotations exceed the Notes page limit", http.StatusRequestEntityTooLarge)
		return
	}
	if err != nil {
		http.Error(w, "Could not prepare annotations", http.StatusInternalServerError)
		return
	}
	page, err := m.wiki.Create(account, note.Title, note.Content)
	if err != nil {
		writeWikiError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, page)
}

func (m *multiServer) exportAnnotations(w http.ResponseWriter, r *http.Request, account string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	payload, err := m.annotations.ExportAccount(account)
	if err != nil {
		http.Error(w, "Could not export annotations", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="mochil-annotations.json"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(payload)
}

func (m *multiServer) deleteAnnotations(w http.ResponseWriter, r *http.Request, account string) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", "DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request annotationDeleteRequest
	if decodeJSONRequest(w, r, 1024, &request, false) != nil {
		return
	}
	if request.Confirmation != "DELETE ALL ANNOTATIONS" {
		http.Error(w, "Type the exact annotation deletion confirmation", http.StatusBadRequest)
		return
	}
	deleted, err := m.annotations.DeleteAccount(account)
	if err != nil {
		http.Error(w, "Could not delete annotations", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}

func (m *multiServer) serveAnnotations(w http.ResponseWriter, r *http.Request, account, libraryID, bookID string) {
	switch r.Method {
	case http.MethodGet:
		since := int64(0)
		if raw := strings.TrimSpace(r.URL.Query().Get("since")); raw != "" {
			var err error
			since, err = strconv.ParseInt(raw, 10, 64)
			if err != nil || since < 0 {
				http.Error(w, "Invalid annotation revision", http.StatusBadRequest)
				return
			}
		}
		writeJSON(w, http.StatusOK, m.annotations.List(account, libraryID, bookID, since))
	case http.MethodPut:
		var request annotationChangeRequest
		if decodeJSONRequest(w, r, maximumAnnotationRequestBytes, &request, false) != nil {
			return
		}
		result, err := m.annotations.Apply(account, libraryID, bookID, request.Changes)
		if errors.Is(err, annotations.ErrInvalid) {
			http.Error(w, "Invalid annotation changes", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "Could not save annotations", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func annotationPath(path, prefix string) (string, string, bool) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, prefix), "/"), "/")
	if len(parts) != 2 {
		return "", "", false
	}
	libraryID, libraryErr := url.PathUnescape(parts[0])
	bookID, bookErr := url.PathUnescape(parts[1])
	return libraryID, bookID, libraryErr == nil && bookErr == nil && libraryID != "" && bookID != ""
}
