package web

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mochil/internal/enrichment"
	"mochil/internal/video"
)

// episodeDetail is TMDB's name, story and still for one episode of a matched show, when there is one.
func (m *multiServer) episodeDetail(ctx context.Context, library Library, item video.Item) (enrichment.EpisodeDetail, bool) {
	if m.tmdb == nil || m.videoMetadata == nil || library.MetadataProvider != "tmdb" || item.Show == "" || item.Episode <= 0 {
		return enrichment.EpisodeDetail{}, false
	}
	show, exists := m.videoMetadata.GetShow(library.ID, item.Show)
	if !exists || show.Provider != "TMDB" || show.ProviderID == "" {
		return enrichment.EpisodeDetail{}, false
	}
	episodes, err := m.tmdb.SeasonEpisodes(ctx, show.ProviderID, item.Season)
	if err != nil {
		return enrichment.EpisodeDetail{}, false
	}
	detail, found := episodes[item.Episode]
	return detail, found
}

// withEpisodeDetail lets TMDB name and describe an episode and give it a still, keeping the file's own values when TMDB has none.
func (m *multiServer) withEpisodeDetail(ctx context.Context, library Library, view videoView) videoView {
	detail, ok := m.episodeDetail(ctx, library, view.Item)
	if !ok {
		return view
	}
	if detail.Name != "" {
		view.EpisodeTitle = detail.Name
	}
	if detail.Overview != "" {
		view.Overview = detail.Overview
	}
	if detail.StillURL != "" {
		view.ArtworkURL = "/api/video-metadata-artwork/" + url.PathEscape(library.ID) + "/" + url.PathEscape(view.ID) + "?still=1"
	}
	return view
}

// videoEpisodeStill serves an episode's TMDB still through the same cached, host-restricted fetch as other artwork.
func (m *multiServer) videoEpisodeStill(w http.ResponseWriter, r *http.Request, library Library, itemID string) {
	if m.metadataArtwork == nil {
		http.NotFound(w, r)
		return
	}
	var found video.Item
	for _, item := range library.Video.Items() {
		if item.ID == itemID {
			found = item
			break
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	detail, ok := m.episodeDetail(ctx, library, found)
	if !ok || detail.StillURL == "" || !strings.HasPrefix(detail.StillURL, "https://image.tmdb.org/") {
		http.NotFound(w, r)
		return
	}
	path, contentType, err := m.metadataArtwork.Fetch(ctx, detail.StillURL)
	if err != nil {
		http.Error(w, "Artwork is temporarily unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, path)
}
