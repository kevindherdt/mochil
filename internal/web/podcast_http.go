package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mochil/internal/jobs"
	"mochil/internal/podcast"
)

var podcastPublicClient = podcast.NewPublicHTTPClient(0)

func (m *multiServer) podcastArtwork(w http.ResponseWriter, r *http.Request) {
	if m.podcasts == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/podcast-artwork/"), "/")
	artworkURL := ""
	if strings.HasPrefix(suffix, "feed/") {
		id, err := url.PathUnescape(strings.TrimPrefix(suffix, "feed/"))
		feed, ok := m.podcasts.Feed(id)
		if err == nil && ok {
			artworkURL = feed.ArtworkURL
		}
	} else {
		id, err := url.PathUnescape(suffix)
		episode, ok := m.podcasts.Episode(id)
		if err == nil && ok {
			artworkURL = episode.ArtworkURL
		}
	}
	if artworkURL == "" {
		http.NotFound(w, r)
		return
	}
	parsed, err := url.Parse(artworkURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	response, err := podcastPublicClient.Do(request)
	if err != nil {
		http.Error(w, "Artwork unavailable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 || response.ContentLength > 20<<20 {
		http.Error(w, "Artwork unavailable", http.StatusBadGateway)
		return
	}
	mediaType := response.Header.Get("Content-Type")
	if mediaType == "" {
		mediaType = mime.TypeByExtension(filepath.Ext(parsed.Path))
	}
	mediaType, imageOK := rasterImageType(mediaType)
	if !imageOK {
		http.Error(w, "Artwork is not an image", http.StatusUnsupportedMediaType)
		return
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (20<<20)+1))
	if err != nil || len(data) > 20<<20 {
		http.Error(w, "Artwork unavailable", http.StatusBadGateway)
		return
	}
	setRemoteImageHeaders(w, mediaType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(data)
}

func (m *multiServer) apiPodcastFeeds(w http.ResponseWriter, r *http.Request) {
	if m.podcasts == nil {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		m.schedulePodcastRefresh()
		writeJSON(w, http.StatusOK, summarizePodcastFeeds(m.podcasts.Feeds()))
	case http.MethodPost:
		var value struct {
			URL string `json:"url"`
		}
		if decodeJSONRequest(w, r, 4096, &value, false) != nil {
			http.Error(w, "Invalid feed", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		feed, err := m.podcasts.Subscribe(ctx, value.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		m.reconcilePodcastFeed(feed)
		writeJSON(w, http.StatusCreated, feed)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) schedulePodcastRefresh() {
	if m.podcasts == nil || m.backgroundJobs == nil {
		return
	}
	_ = m.backgroundJobs.Submit("podcasts:refresh-stale", func(parent context.Context) error {
		ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
		defer cancel()
		m.podcasts.RefreshStale(ctx, 30*time.Minute)
		m.reconcilePodcastFeeds()
		return nil
	})
}

func (m *multiServer) apiPodcastSearch(w http.ResponseWriter, r *http.Request) {
	if m.podcasts == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	results, err := podcast.SearchDirectory(ctx, nil, r.URL.Query().Get("q"), 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	for i := range results {
		if results[i].ArtworkURL != "" {
			prefix := "/api/podcast-search-artwork"
			if strings.HasPrefix(r.URL.Path, "/api/android/v1/") {
				prefix = "/api/android/v1/podcast-search-artwork"
			}
			results[i].ArtworkURL = prefix + "?url=" + url.QueryEscape(results[i].ArtworkURL)
		}
	}
	writeJSON(w, http.StatusOK, results)
}

func (m *multiServer) podcastSearchArtwork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	parsed, err := url.Parse(r.URL.Query().Get("url"))
	if err != nil || parsed == nil {
		http.NotFound(w, r)
		return
	}
	host := strings.ToLower(parsed.Hostname())
	if parsed.Scheme != "https" || (host != "mzstatic.com" && !strings.HasSuffix(host, ".mzstatic.com")) {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	client := &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, "Artwork unavailable", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		http.Error(w, "Artwork unavailable", http.StatusBadGateway)
		return
	}
	mediaType, imageOK := rasterImageType(res.Header.Get("Content-Type"))
	if !imageOK {
		http.Error(w, "Artwork unavailable", http.StatusUnsupportedMediaType)
		return
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (5<<20)+1))
	if err != nil || len(data) > 5<<20 {
		http.Error(w, "Artwork unavailable", http.StatusBadGateway)
		return
	}
	setRemoteImageHeaders(w, mediaType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(data)
}

func (m *multiServer) apiPodcastFeed(w http.ResponseWriter, r *http.Request) {
	if m.podcasts == nil {
		http.NotFound(w, r)
		return
	}
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/podcast-feeds/"), "/")
	parts := strings.SplitN(suffix, "/", 2)
	id, err := url.PathUnescape(parts[0])
	if err != nil || id == "" {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 2 && parts[1] == "refresh" && r.Method == http.MethodPost {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		feed, refreshErr := m.podcasts.Refresh(ctx, id)
		if refreshErr != nil {
			writeJSON(w, http.StatusBadGateway, feed)
			return
		}
		m.reconcilePodcastFeed(feed)
		writeJSON(w, http.StatusOK, feed)
		return
	}
	if len(parts) == 2 && parts[1] == "policy" && r.Method == http.MethodPut {
		var policy podcast.FeedPolicy
		if decodeJSONRequest(w, r, 4096, &policy, false) != nil {
			http.Error(w, "Invalid podcast policy", http.StatusBadRequest)
			return
		}
		feed, policyErr := m.podcasts.SetPolicy(id, policy)
		if policyErr != nil {
			http.Error(w, policyErr.Error(), http.StatusBadRequest)
			return
		}
		m.reconcilePodcastFeed(feed)
		writeJSON(w, http.StatusOK, feed)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodDelete {
		if m.podcastDownloads != nil {
			for _, feed := range m.podcasts.Feeds() {
				if feed.ID == id {
					for _, episode := range feed.Episodes {
						_ = m.podcastDownloads.Delete(episode.ID)
					}
					break
				}
			}
		}
		if m.podcasts.Remove(id) != nil {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (m *multiServer) podcastMedia(w http.ResponseWriter, r *http.Request) {
	if m.podcasts == nil || m.podcastDownloads == nil || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		http.NotFound(w, r)
		return
	}
	id, err := url.PathUnescape(strings.Trim(strings.TrimPrefix(r.URL.Path, "/podcast-media/"), "/"))
	episode, ok := m.podcasts.Episode(id)
	if err != nil || !ok {
		http.NotFound(w, r)
		return
	}
	if downloaded, exists := m.podcastDownloads.Path(id); exists {
		file, openErr := os.Open(downloaded)
		if openErr != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		info, _ := file.Stat()
		mediaType := episode.MediaType
		if mediaType == "" {
			mediaType = mime.TypeByExtension(filepath.Ext(downloaded))
		}
		setRemoteMediaHeaders(w, mediaType)
		w.Header().Set("Accept-Ranges", "bytes")
		if r.URL.Query().Get("download") == "1" {
			w.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(episode.Title+filepath.Ext(episode.MediaURL)))
		}
		http.ServeContent(w, r, episode.Title, info.ModTime(), file)
		return
	}
	if strings.TrimSpace(episode.MediaURL) == "" && episode.FeedID != "" {
		// Continue Listening can be the first podcast request after a restart,
		// before catalogue-triggered background refresh has run. Repair legacy
		// state synchronously once so the visible episode is immediately usable.
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		_, refreshErr := m.podcasts.Refresh(ctx, episode.FeedID)
		cancel()
		if refreshErr == nil {
			episode, ok = m.podcasts.Episode(id)
		}
	}
	if !ok || strings.TrimSpace(episode.MediaURL) == "" {
		http.Error(w, "Episode unavailable", http.StatusBadGateway)
		return
	}
	request, requestErr := http.NewRequestWithContext(r.Context(), r.Method, episode.MediaURL, nil)
	if requestErr != nil {
		http.Error(w, "Episode unavailable", http.StatusBadGateway)
		return
	}
	// Media3 uses conditional byte ranges when resuming long-form audio. Keep
	// those headers together and prevent transparent content encoding, whose
	// byte offsets would no longer describe the original enclosure.
	for _, header := range []string{"Range", "If-Range"} {
		if value := r.Header.Get(header); value != "" {
			request.Header.Set(header, value)
		}
	}
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("User-Agent", "Mochil podcast relay")
	upstreamStarted := time.Now()
	response, responseErr := podcastPublicClient.Do(request)
	if responseErr != nil {
		m.podcastStreamLogger().Warn("podcast stream upstream failed", "episode_id", id, "range", request.Header.Get("Range") != "", "duration_ms", time.Since(upstreamStarted).Milliseconds())
		http.Error(w, "Episode unavailable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	for _, header := range []string{"Content-Length", "Content-Range", "Accept-Ranges", "ETag", "Last-Modified"} {
		if value := response.Header.Get(header); value != "" {
			w.Header().Set(header, value)
		}
	}
	setRemoteMediaHeaders(w, response.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(response.StatusCode)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	upstreamMS := time.Since(upstreamStarted).Milliseconds()
	var copied int64
	var copyErr error
	if r.Method != http.MethodHead {
		copied, copyErr = io.Copy(w, response.Body)
	}
	m.podcastStreamLogger().Info("podcast stream", "episode_id", id, "status", response.StatusCode, "range", request.Header.Get("Range") != "", "upstream_ms", upstreamMS, "bytes", copied, "duration_ms", time.Since(upstreamStarted).Milliseconds(), "copy_error", copyErr != nil)
}

// podcastSavedOnServer says whether the server itself has downloaded the episode (as opposed to a file that happens
// to sit in the podcast folder), so the shelf can mark it as available without the feed.
func (m *multiServer) podcastSavedOnServer(id string) bool {
	if m.podcastDownloads == nil {
		return false
	}
	_, saved := m.podcastDownloads.Path(id)
	return saved
}

func (m *multiServer) podcastStreamLogger() *slog.Logger {
	if m.root != nil && m.root.log != nil {
		return m.root.log
	}
	return slog.Default()
}

func (m *multiServer) apiPodcastDownload(w http.ResponseWriter, r *http.Request) {
	if m.podcasts == nil || m.podcastDownloads == nil {
		http.NotFound(w, r)
		return
	}
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/podcast-downloads"), "/")
	if suffix == "" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, m.podcastDownloads.Stats())
		return
	}
	if suffix == "retry" && r.Method == http.MethodPost {
		retried := m.podcastDownloads.RetryFailed(m.podcasts.Episodes())
		writeJSON(w, http.StatusAccepted, map[string]int{"retried": retried})
		return
	}
	id, err := url.PathUnescape(suffix)
	episode, ok := m.podcasts.Episode(id)
	if err != nil || !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, m.podcastDownloads.Status(id))
	case http.MethodPost:
		state, err := m.podcastDownloads.StartWithError(episode)
		status := http.StatusAccepted
		switch {
		case errors.Is(err, jobs.ErrDuplicate):
			status = http.StatusConflict
		case errors.Is(err, jobs.ErrQueueFull), errors.Is(err, jobs.ErrClosed):
			status = http.StatusServiceUnavailable
		case err != nil:
			status = http.StatusInternalServerError
		case state.Status == "ready":
			status = http.StatusOK
		}
		writeJSON(w, status, state)
	case http.MethodDelete:
		if err := m.podcastDownloads.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) reconcilePodcastFeed(feed podcast.Feed) {
	if m.podcastDownloads != nil {
		m.podcastDownloads.Reconcile(feed, m.podcastPlayedAt())
	}
}

func (m *multiServer) reconcilePodcastFeeds() {
	if m.podcasts == nil || m.podcastDownloads == nil {
		return
	}
	for _, feed := range m.podcasts.Feeds() {
		m.podcastDownloads.Reconcile(feed, m.podcastPlayedAt())
	}
}

func (m *multiServer) podcastPlayedAt() podcast.PlaybackLookup {
	if m.audioProgress == nil {
		return nil
	}
	libraryID := ""
	for _, library := range m.libraries {
		if library.Kind == "podcasts" {
			libraryID = library.ID
			break
		}
	}
	if libraryID == "" {
		return nil
	}
	account := m.readingAccount(nil)
	return func(episodeID string) (time.Time, bool) {
		progress, ok := m.audioProgress.Get(account, libraryID, episodeID)
		if !ok || !progress.Finished || progress.UpdatedAt <= 0 {
			return time.Time{}, false
		}
		return time.Unix(progress.UpdatedAt, 0), true
	}
}
