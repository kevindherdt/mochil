package web

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mochil/internal/enrichment"
)

// apiVideoMetadataSearch is the per-item half of TMDB matching: type a title, see candidates, pick one.
//
// The library-wide matcher (apiVideoMetadataMatch) is deliberately a batch job — it walks every unmatched film with
// a confidence rule and applies what it is sure about. That is the "big button" Kevin did not want: "I dont want a
// big button like before it should be out of the way somewhere or being able to click on the series/movie/ etc and
// have a edit button like a pencil". So this answers exactly one question, for exactly one title, and the person
// chooses; nothing is applied here.
//
// GET /api/video-metadata-search/<libraryID>?q=<title>&show=1
func (m *multiServer) apiVideoMetadataSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	libraryID, err := url.PathUnescape(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-metadata-search/"), "/"))
	library, ok := m.library(libraryID)
	if err != nil || !ok || library.Video == nil || library.MetadataProvider != "tmdb" || m.tmdb == nil {
		http.NotFound(w, r)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Error(w, "Type a title to search for.", http.StatusBadRequest)
		return
	}
	// A show is searched as TV so TMDB answers with series, not films that happen to share a name.
	kind := library.Kind
	if r.URL.Query().Get("show") == "1" {
		kind = "tv"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	results, searchErr := m.tmdb.Search(ctx, enrichment.TMDBQuery{Title: query, Kind: kind})
	if searchErr != nil {
		http.Error(w, "TMDB could not be reached.", http.StatusBadGateway)
		return
	}
	// Ten is enough to recognise the right one and few enough to read on a phone.
	if len(results) > 10 {
		results = results[:10]
	}
	out := make([]map[string]any, 0, len(results))
	for _, result := range results {
		out = append(out, map[string]any{
			"provider_id":  result.ProviderID,
			"title":        result.Title,
			"year":         result.Year,
			"overview":     result.Overview,
			"poster_url":   result.CoverURL,
			"backdrop_url": result.BackdropURL,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": out})
}
