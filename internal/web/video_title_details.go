package web

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mochil/internal/enrichment"
)

// apiVideoDetails answers with the genres, runtime, age rating and score TMDB has for one matched show (?show=) or movie (?item=).
// It is asked for the one title being opened, so a long list never makes TMDB calls of its own.
func (m *multiServer) apiVideoDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	libraryID, err := url.PathUnescape(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-details/"), "/"))
	library, ok := m.library(libraryID)
	if err != nil || !ok || library.Video == nil || m.tmdb == nil || m.videoMetadata == nil || library.MetadataProvider != "tmdb" {
		http.NotFound(w, r)
		return
	}
	mediaType := "movie"
	var saved videoSaved
	if show := strings.TrimSpace(r.URL.Query().Get("show")); show != "" {
		mediaType = "tv"
		value, exists := m.videoMetadata.GetShow(libraryID, show)
		saved = videoSaved{provider: value.Provider, id: value.ProviderID, exists: exists}
	} else {
		value, exists := m.videoMetadata.Get(libraryID, r.URL.Query().Get("item"))
		saved = videoSaved{provider: value.Provider, id: value.ProviderID, exists: exists}
	}
	if !saved.exists || saved.provider != "TMDB" || saved.id == "" {
		writeJSON(w, http.StatusOK, enrichment.TitleDetails{})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	details, detailsErr := m.tmdb.Details(ctx, mediaType, saved.id)
	if detailsErr != nil {
		writeJSON(w, http.StatusOK, enrichment.TitleDetails{})
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	writeJSON(w, http.StatusOK, details)
}

type videoSaved struct {
	provider, id string
	exists       bool
}
