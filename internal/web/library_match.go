package web

import (
	"errors"
	"net/http"
	"os"
	"sort"
	"strings"

	"mochil/internal/customcolumns"
	"mochil/internal/enrichment"
)

type matchCandidate struct {
	enrichment.Suggestion
	Score float64 `json:"score"`
	// Overview shadows the suggestion's own blurb so it can be left out of a list response. It is read in exactly one
	// place — staging a match fills the description only when the book has none — and it is the largest field on the
	// row (about 700 bytes each), so sending it for a book that already has a description is pure waste.
	Overview string `json:"overview,omitempty"`
}

// forList returns the candidate as a list row should carry it: the blurb only when it is the only description there is.
//
// The value can be in either place: a candidate built right now has it on the embedded suggestion, and one that came
// back from the on-disk cache has it on this struct's own field, because that is the field JSON writes. Read both
// before clearing, or the second shape silently loses every blurb.
func (c *matchCandidate) forList(row *editorRow) *matchCandidate {
	blurb := c.Overview
	if blurb == "" {
		blurb = c.Suggestion.Overview
	}
	c.Overview = ""
	if row.Description == "" {
		c.Overview = blurb
	}
	return c
}

// matchScore ranks a candidate against a book: the title counts most, the author confirms it. A book with no known
// author is judged on its title alone, so a missing author does not push every candidate down.
func matchScore(title string, authors []string, candidate enrichment.Suggestion) float64 {
	want, got := importKey(title), importKey(candidate.Title)
	var titleScore float64
	switch {
	case want == "" || got == "":
	case want == got:
		titleScore = 1
	case strings.Contains(got, want) || strings.Contains(want, got):
		titleScore = 0.7
	default:
		titleWords := map[string]bool{}
		for _, word := range strings.Fields(strings.ToLower(title)) {
			titleWords[importKey(word)] = true
		}
		shared, total := 0, len(titleWords)
		for _, word := range strings.Fields(strings.ToLower(candidate.Title)) {
			if key := importKey(word); key != "" && titleWords[key] {
				shared++
			} else {
				total++
			}
		}
		if total > 0 {
			titleScore = 0.6 * float64(shared) / float64(total)
		}
	}
	if len(authors) == 0 {
		return titleScore
	}
	authorScore := 0.0
	for _, author := range authors {
		for _, other := range candidate.Authors {
			if importKey(author) != "" && importKey(author) == importKey(other) {
				authorScore = 1
			}
		}
	}
	return 0.65*titleScore + 0.35*authorScore
}

// editorMatch finds Hardcover candidates for one book, best first. It only reads: the person chooses the candidate and
// which of its fields to take, and those go through the ordinary edit flow so they can be reviewed and undone.
func (m *multiServer) editorMatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !m.moduleOn("hardcover") || m.hardcover == nil {
		http.Error(w, "Turn on the Hardcover module in Settings to find matches.", http.StatusConflict)
		return
	}
	query := r.URL.Query()
	lib, ok := m.editorLibrary(w, query.Get("library"))
	if !ok {
		return
	}
	book, found := lib.Store.Book(query.Get("book"))
	if !found {
		http.NotFound(w, r)
		return
	}
	title := strings.TrimSpace(query.Get("title"))
	if title == "" {
		title = book.Title
	}
	author := strings.TrimSpace(query.Get("author"))
	authors := book.Authors
	if author != "" {
		authors = []string{author}
	}
	search := enrichment.Query{Title: title}
	if len(authors) > 0 {
		search.Author = authors[0]
	}
	results, err := m.hardcover.Search(r.Context(), search)
	if err == nil && len(results) == 0 && search.Author != "" {
		results, err = m.hardcover.Search(r.Context(), enrichment.Query{Title: title})
	}
	if err != nil {
		status, message := providerConnectionError(err)
		http.Error(w, message, status)
		return
	}
	candidates := make([]matchCandidate, 0, len(results))
	for _, result := range results {
		candidates = append(candidates, matchCandidate{
			Suggestion: result,
			// Copied across because the field on this struct shadows the one on the suggestion: without this the blurb
			// never leaves the server, and Replace could not fix a wrong description. The bulk path drops it on purpose
			// (700 bytes a row); this is one book, opened because somebody is deciding about it.
			Overview: result.Overview,
			Score:    matchScore(title, authors, result),
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	if len(candidates) > 8 {
		candidates = candidates[:8]
	}
	// "clear" means one candidate is well ahead; otherwise the person is asked to pick.
	clear := len(candidates) > 0 && candidates[0].Score >= 0.9 && (len(candidates) == 1 || candidates[0].Score-candidates[1].Score >= 0.2)
	writeJSON(w, http.StatusOK, map[string]any{"title": title, "authors": authors, "candidates": candidates, "clear": clear})
}

func (m *multiServer) editorColumns() []customcolumns.Column {
	if m.customColumns == nil {
		return []customcolumns.Column{}
	}
	return m.customColumns.Columns()
}

// editorColumnsRequest lists, adds and removes the editor's own columns.
func (m *multiServer) editorColumnsRequest(w http.ResponseWriter, r *http.Request) {
	if m.customColumns == nil {
		http.Error(w, "Custom columns cannot be saved right now.", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"columns": m.editorColumns()})
	case http.MethodPost:
		var input struct {
			Label string `json:"label"`
			Type  string `json:"type"`
		}
		if decodeJSONRequest(w, r, 4096, &input, true) != nil {
			http.Error(w, "The request could not be read.", http.StatusBadRequest)
			return
		}
		column, err := m.customColumns.Add(input.Label, input.Type)
		if errors.Is(err, customcolumns.ErrInvalid) {
			http.Error(w, strings.TrimPrefix(err.Error(), customcolumns.ErrInvalid.Error()+": "), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "The column could not be saved.", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, column)
	case http.MethodDelete:
		err := m.customColumns.Remove(r.URL.Query().Get("id"))
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "The column could not be removed.", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"removed": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
