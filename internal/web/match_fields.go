package web

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"mochil/internal/statefile"
)

// matchFieldsVersion is the shape of the stored file. A file from another version is refused rather than
// half-read, because misreading it would either widen what a match may write or silently take everything away.
const matchFieldsVersion = 1

// matchWritable is every field a Hardcover match can carry, in the order the editor lists them. Publisher,
// language, tags and the sort keys are not among them: a match does not propose those.
//
// The order is part of the answer, so a client can render the boxes without sorting them and two libraries
// holding the same choice store the same bytes.
var matchWritable = []string{
	"title",
	"authors",
	"series",
	"series_index",
	"published",
	"description",
	"identifiers",
}

// matchFieldFile is what is stored on disk: the chosen fields per library.
//
// It is a map from library to list rather than rows because every read is "this one library's choice", and
// the file only ever holds one short list per library.
type matchFieldFile struct {
	Version   int                 `json:"version"`
	Libraries map[string][]string `json:"libraries"`
}

// matchFieldsStore is which fields a match may write, kept per library.
//
// It lives on the server rather than in the browser because the choice belongs to the library: the same
// person editing from a laptop and a tablet should get the same boxes, and a browser that has never seen the
// editor should not offer to move a series name that this library's owner pinned.
type matchFieldsStore struct {
	mu        sync.Mutex
	path      string
	libraries map[string][]string
}

func openMatchFields(path string) (*matchFieldsStore, error) {
	store := &matchFieldsStore{path: path, libraries: map[string][]string{}}
	if path == "" {
		return store, nil
	}
	var stored matchFieldFile
	if err := statefile.DecodeJSON(path, 8<<20, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store, nil
		}
		return nil, err
	}
	if stored.Version != matchFieldsVersion {
		return nil, errors.New("the stored match fields in " + filepath.Base(path) + " were written by another version")
	}
	if stored.Libraries != nil {
		store.libraries = stored.Libraries
	}
	return store, nil
}

// canonicalMatchFields keeps the names a match may write, in the canonical order and without repeats.
// The result is never nil, so a library that chose nothing stores [] rather than null and reads back as
// "writes nothing" instead of "never chosen".
func canonicalMatchFields(fields []string) []string {
	chosen := make(map[string]bool, len(fields))
	for _, field := range fields {
		chosen[field] = true
	}
	kept := make([]string, 0, len(fields))
	for _, field := range matchWritable {
		if chosen[field] {
			kept = append(kept, field)
		}
	}
	return kept
}

// allowed reports whether name is a field a match may write. The endpoint refuses anything else instead of
// dropping it, so a client typo is a 400 rather than a setting that quietly lost a field.
func matchFieldAllowed(name string) bool {
	for _, field := range matchWritable {
		if field == name {
			return true
		}
	}
	return false
}

// fields is what this library's matches may write. A library with nothing saved gets every field, which is
// what a match did before the choice existed; a library that saved an empty list gets an empty list.
func (s *matchFieldsStore) fields(library string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.libraries[library]
	if !ok {
		return canonicalMatchFields(matchWritable)
	}
	return canonicalMatchFields(stored)
}

// set replaces one library's choice. The whole list is the setting, so a shorter list narrows what a match
// writes rather than adding to it; an empty list is a real choice and is kept as one.
func (s *matchFieldsStore) set(library string, fields []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := canonicalMatchFields(fields)
	previous, had := s.libraries[library]
	s.libraries[library] = next
	if err := s.saveLocked(); err != nil {
		// What the panel shows comes from memory, so memory may not claim more or less than the file holds:
		// on the next start the two would disagree.
		if had {
			s.libraries[library] = previous
		} else {
			delete(s.libraries, library)
		}
		return err
	}
	return nil
}

func (s *matchFieldsStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	// The package's atomic write: temp file, chmod, sync, close, rename. This file is read at startup, and
	// writing it in place is how a crash leaves it empty.
	return writeJSONFile(s.path, matchFieldFile{Version: matchFieldsVersion, Libraries: s.libraries}, ".match-fields-*")
}

// editorMatchFields serves GET and PUT on /api/metadata/editor/match-fields.
//
// GET answers this library's choice or all seven; PUT replaces it and answers 204 with no body. The list is
// the whole setting, not a patch, because a checkbox the person unticked is missing from the list and a
// merge could not tell that apart from a field they never chose.
func (m *multiServer) editorMatchFields(w http.ResponseWriter, r *http.Request) {
	if m.matchFields == nil {
		http.Error(w, "The match fields cannot be read or saved right now.", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		lib, ok := m.editorLibrary(w, r.URL.Query().Get("library"))
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, map[string][]string{"fields": m.matchFields.fields(lib.ID)})
	case http.MethodPut:
		var request struct {
			Library string   `json:"library"`
			Fields  []string `json:"fields"`
		}
		if err := decodeJSONRequest(w, r, 1<<20, &request, true); err != nil {
			http.Error(w, "The request could not be read.", http.StatusBadRequest)
			return
		}
		if request.Fields == nil {
			http.Error(w, "The request did not name the fields a match may write.", http.StatusBadRequest)
			return
		}
		lib, ok := m.editorLibrary(w, request.Library)
		if !ok {
			return
		}
		for _, field := range request.Fields {
			if !matchFieldAllowed(field) {
				http.Error(w, "A match cannot write "+field+".", http.StatusBadRequest)
				return
			}
		}
		if err := m.matchFields.set(lib.ID, request.Fields); err != nil {
			http.Error(w, "The match fields could not be saved.", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
