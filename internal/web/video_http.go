package web

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"mochil/internal/video"
)

var videoHLSSegment = regexp.MustCompile(`^seg([0-9]{5})\.ts$`)

func (m *multiServer) videoArtworkHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.SplitN(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-artwork/"), "/"), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	libraryID, libraryErr := url.PathUnescape(parts[0])
	itemID, itemErr := url.PathUnescape(parts[1])
	library, ok := m.library(libraryID)
	if libraryErr != nil || itemErr != nil || !ok || library.Video == nil {
		http.NotFound(w, r)
		return
	}
	item, exists := library.Video.Item(itemID)
	if !exists {
		http.NotFound(w, r)
		return
	}
	scope := video.ArtworkItem
	switch r.URL.Query().Get("scope") {
	case "":
	case "show":
		scope = video.ArtworkShow
	case "season":
		scope = video.ArtworkSeason
	default:
		http.Error(w, "Unknown artwork scope", http.StatusBadRequest)
		return
	}
	if file, extension, found := library.Video.OpenSidecarFor(itemID, scope); found {
		defer file.Close()
		// The sidecar is its own entity: validating it with the video's mtime kept
		// clients on a replaced poster until the video file itself changed.
		modified := item.Modified
		if info, statErr := file.Stat(); statErr == nil {
			modified = info.ModTime()
		}
		serveVideoArtwork(w, r, file, extension, modified)
		return
	}
	if m.videoArtwork == nil {
		http.NotFound(w, r)
		return
	}
	source, _, err := library.Video.Open(itemID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	generated, err := m.videoArtwork.Poster(source, item)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(generated)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	serveVideoArtwork(w, r, file, filepath.Ext(generated), info.ModTime())
}

func serveVideoArtwork(w http.ResponseWriter, r *http.Request, file *os.File, extension string, modified time.Time) {
	mediaType := mime.TypeByExtension(extension)
	if mediaType == "" {
		mediaType = "image/jpeg"
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, filepath.Base(file.Name()), modified, file)
}

func (m *multiServer) videoPreviewHTTP(w http.ResponseWriter, r *http.Request) {
	if (r.Method != http.MethodGet && r.Method != http.MethodHead) || m.videoPreviews == nil {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-previews/"), "/"), "/")
	if len(parts) != 3 || !strings.HasSuffix(parts[2], ".jpg") {
		http.NotFound(w, r)
		return
	}
	libraryID, firstErr := url.PathUnescape(parts[0])
	itemID, secondErr := url.PathUnescape(parts[1])
	position, thirdErr := strconv.Atoi(strings.TrimSuffix(parts[2], ".jpg"))
	library, ok := m.library(libraryID)
	if firstErr != nil || secondErr != nil || thirdErr != nil || position < 0 || !ok || library.Video == nil {
		http.NotFound(w, r)
		return
	}
	file, item, err := library.Video.Open(itemID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	generated, resolved, err := m.videoPreviews.Frame(file, item, position)
	if err != nil {
		http.Error(w, "This preview could not be prepared", http.StatusServiceUnavailable)
		return
	}
	preview, err := os.Open(generated)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer preview.Close()
	info, err := preview.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Video-Preview-Position", strconv.Itoa(resolved))
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, "preview.jpg", info.ModTime(), preview)
}

func (m *multiServer) apiVideoPreferences(w http.ResponseWriter, r *http.Request) {
	if m.videoPreferences == nil {
		http.Error(w, "Video preferences are unavailable", http.StatusNotFound)
		return
	}
	library, show := strings.TrimSpace(r.URL.Query().Get("library")), strings.TrimSpace(r.URL.Query().Get("show"))
	if library == "" {
		http.Error(w, "library is required", http.StatusBadRequest)
		return
	}
	if _, ok := m.library(library); !ok {
		http.Error(w, "Unknown library", http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		effective, override := m.videoPreferences.Effective(library, show)
		writeJSON(w, http.StatusOK, map[string]any{"effective": effective, "override": override})
	case http.MethodPut:
		if show == "" {
			http.Error(w, "show is required", http.StatusBadRequest)
			return
		}
		var value video.LanguagePreferences
		if err := decodeJSONRequest(w, r, 4096, &value, false); err != nil {
			http.Error(w, "Invalid preferences", http.StatusBadRequest)
			return
		}
		if err := m.videoPreferences.Set(library, show, value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		effective, override := m.videoPreferences.Effective(library, show)
		writeJSON(w, http.StatusOK, map[string]any{"effective": effective, "override": override})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) videoHLS(w http.ResponseWriter, r *http.Request) {
	isStop := r.Method == http.MethodDelete || r.Method == http.MethodPost
	if (r.Method != http.MethodGet && !isStop) || m.hls == nil {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-hls/"), "/"), "/")
	if len(parts) != 4 && len(parts) != 5 {
		http.NotFound(w, r)
		return
	}
	libraryID, firstErr := url.PathUnescape(parts[0])
	itemID, secondErr := url.PathUnescape(parts[1])
	audioIndex, thirdErr := strconv.Atoi(parts[2])
	quality, resource := video.HLSQualityHigh, parts[3]
	if len(parts) == 5 {
		var valid bool
		quality, valid = video.ParseHLSQuality(parts[3])
		if !valid {
			http.Error(w, "Unknown video quality", http.StatusBadRequest)
			return
		}
		resource = parts[4]
	}
	library, ok := m.library(libraryID)
	if firstErr != nil || secondErr != nil || thirdErr != nil || !ok || library.Video == nil {
		http.NotFound(w, r)
		return
	}
	item, exists := library.Video.Item(itemID)
	if !exists {
		http.NotFound(w, r)
		return
	}
	if isStop {
		key := m.hls.Key(item, audioIndex, quality)
		// One encoder serves everyone watching the same episode, so a player closing
		// must not cancel an encode someone else is still streaming. With no other
		// viewer the stop is honoured immediately; otherwise the passive idle timeout
		// finishes the job instead.
		if m.videoViewers.remembered(key, m.readingAccount(r)) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// The client closing the player (or navigating away, via
		// sendBeacon — which can only POST, never DELETE) tells the server
		// to stop encoding right away, instead of relying on the passive
		// idle timeout to notice nobody is fetching segments anymore up to
		// a minute later. Stopping something that already finished, or was
		// never started, is a harmless no-op either way: the client may
		// race a close against an encoder that is already done, so this
		// always reports success.
		m.hls.Stop(key)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if resource == "index.m3u8" || strings.HasSuffix(resource, ".ts") || strings.HasSuffix(resource, ".m4s") {
		m.videoViewers.observe(m.hls.Key(item, audioIndex, quality), m.readingAccount(r))
	}
	key := ""
	if resource == "index.m3u8" {
		file, opened, err := library.Video.Open(itemID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		key, err = m.hls.Start(file, opened, audioIndex, quality)
		if err != nil {
			// Start also fails for transient conditions (a retry window, the previous
			// encoder still stopping, a full job queue), which the client should retry
			// rather than treat as a bad request.
			http.Error(w, "This audio track cannot be prepared", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		prefix := "/api/video-hls/"
		if r.Header.Get("X-Mochil-Android-Video") == "1" {
			prefix = "/api/android/v1/video-hls/"
		}
		base := prefix + url.PathEscape(libraryID) + "/" + url.PathEscape(itemID) + "/" + strconv.Itoa(audioIndex) + "/" + string(quality) + "/"
		_, _ = w.Write(video.RewriteHLSPlaylist(m.hls.Playlist(opened), base, key))
		return
	} else {
		key = m.hls.Key(item, audioIndex, quality)
	}
	if matches := videoHLSSegment.FindStringSubmatch(resource); len(matches) == 2 {
		segment, _ := strconv.Atoi(matches[1])
		file, opened, err := library.Video.Open(itemID)
		if err != nil || m.hls.EnsureSegment(file, opened, audioIndex, quality, segment) != nil {
			http.Error(w, "This part of the movie could not be prepared", http.StatusServiceUnavailable)
			return
		}
	}
	file, mediaType, err := m.hls.Open(key, resource)
	if err != nil {
		http.Error(w, "Compatibility stream is not ready", http.StatusServiceUnavailable)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, resource, info.ModTime(), file)
}

func (m *multiServer) videoSubtitle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || m.subtitles == nil {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-subtitles/"), "/"), "/")
	// Two shapes: <lib>/<item>/<track>.vtt for text tracks (extracted on the fly), and
	// <lib>/<item>/<track>/manifest.json plus <lib>/<item>/<track>/c00001.png for image-based tracks (PGS), which
	// cannot be WebVTT at all. Android and older clients only ever asked for the first shape.
	imageShape := len(parts) == 4 && (parts[3] == "manifest.json" || strings.HasSuffix(parts[3], ".png"))
	if !imageShape && (len(parts) != 3 || !strings.HasSuffix(parts[2], ".vtt")) {
		http.NotFound(w, r)
		return
	}
	libraryID, firstErr := url.PathUnescape(parts[0])
	itemID, secondErr := url.PathUnescape(parts[1])
	trackToken := parts[2]
	if imageShape {
		trackToken = parts[2]
	}
	streamIndex, thirdErr := strconv.Atoi(strings.TrimSuffix(trackToken, ".vtt"))
	library, ok := m.library(libraryID)
	if firstErr != nil || secondErr != nil || thirdErr != nil || !ok || library.Video == nil {
		http.NotFound(w, r)
		return
	}
	file, item, err := library.Video.Open(itemID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	if imageShape {
		m.serveSubtitleImage(w, r, library.Video.Open, item, streamIndex, parts[3])
		return
	}
	allowed := false
	for _, subtitle := range item.Subtitles {
		if subtitle.Index == streamIndex && subtitle.Text {
			allowed = true
			break
		}
	}
	if !allowed {
		http.NotFound(w, r)
		return
	}
	// Each extraction demuxes the whole video in its own process. Two at a time
	// keeps a burst of requests from becoming a fork storm; the player retries.
	if m.subtitleSlots != nil {
		select {
		case m.subtitleSlots <- struct{}{}:
			defer func() { <-m.subtitleSlots }()
		default:
			w.Header().Set("Retry-After", "5")
			http.Error(w, "Subtitle extraction is busy, try again shortly", http.StatusTooManyRequests)
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()
	data, err := m.subtitles.Extract(ctx, file, streamIndex)
	if err != nil {
		http.Error(w, "Could not prepare subtitles", http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(data)
}

func (m *multiServer) apiVideoProgress(w http.ResponseWriter, r *http.Request) {
	if m.videoProgress == nil {
		http.Error(w, "Video progress storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	account := m.readingAccount(r)
	if r.URL.Path == "/api/video-progress" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		progressItems := m.videoProgress.List(account)
		sort.Slice(progressItems, func(i, j int) bool { return progressItems[i].UpdatedAt > progressItems[j].UpdatedAt })
		continueOnly := r.URL.Query().Get("continue") == "1"
		limit := len(progressItems)
		if continueOnly {
			limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
			if limit < 1 || limit > 20 {
				limit = 5
			}
		}
		writeJSON(w, http.StatusOK, m.videoProgressViews(account, progressItems, continueOnly, limit))
		return
	}
	parts := strings.SplitN(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/video-progress/"), "/"), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	libraryID, firstErr := url.PathUnescape(parts[0])
	itemID, secondErr := url.PathUnescape(parts[1])
	library, ok := m.library(libraryID)
	if firstErr != nil || secondErr != nil || !ok || library.Video == nil {
		http.NotFound(w, r)
		return
	}
	item, exists := library.Video.Item(itemID)
	if !exists {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		progress, exists := m.videoProgress.Get(account, libraryID, itemID)
		if !exists {
			writeJSON(w, http.StatusOK, map[string]any{"position_seconds": 0, "duration_seconds": 0})
			return
		}
		writeJSON(w, http.StatusOK, progress)
	case http.MethodPut:
		var value struct {
			video.Progress
			ExpectedRevision      *uint64 `json:"expected_revision"`
			ExplicitSeek          bool    `json:"explicit_seek"`
			ExpectedMediaRevision string  `json:"expected_media_revision"`
		}
		if decodeJSONRequest(w, r, 16<<10, &value, false) != nil {
			http.Error(w, "Invalid video progress", http.StatusBadRequest)
			return
		}
		if value.ExpectedMediaRevision != "" && value.ExpectedMediaRevision != item.Revision {
			writeJSON(w, http.StatusPreconditionFailed, map[string]string{"error": "This video file changed. Reopen it before syncing progress.", "code": "video_media_changed"})
			return
		}
		value.LibraryID, value.ItemID = libraryID, itemID
		var saveErr error
		if value.ExpectedRevision == nil {
			saveErr = m.videoProgress.Put(account, value.Progress)
		} else if value.ExplicitSeek {
			saveErr = m.videoProgress.PutIfRevisionExplicit(account, value.Progress, *value.ExpectedRevision)
		} else {
			saveErr = m.videoProgress.PutIfRevision(account, value.Progress, *value.ExpectedRevision)
		}
		if errors.Is(saveErr, video.ErrProgressConflict) {
			http.Error(w, saveErr.Error(), http.StatusConflict)
			return
		}
		if saveErr != nil {
			http.Error(w, saveErr.Error(), http.StatusBadRequest)
			return
		}
		progress, _ := m.videoProgress.Get(account, libraryID, itemID)
		writeJSON(w, http.StatusOK, progress)
	case http.MethodPatch, http.MethodPost:
		var value struct {
			Finished  *bool `json:"finished"`
			Dismissed *bool `json:"dismissed"`
		}
		if decodeJSONRequest(w, r, 4096, &value, false) != nil || value.Finished == nil && value.Dismissed == nil {
			http.Error(w, "Invalid video progress status", http.StatusBadRequest)
			return
		}
		item, _ := library.Video.Item(itemID)
		progress, err := m.videoProgress.PatchStatus(account, libraryID, itemID, item.Duration, value.Finished, value.Dismissed)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, progress)
	case http.MethodDelete:
		if err := m.videoProgress.Delete(account, libraryID, itemID); err != nil {
			http.Error(w, "Could not remove video progress", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) videoProgressViews(account string, progressItems []video.Progress, continueOnly bool, limit int) []videoView {
	items := make([]videoView, 0, min(limit, len(progressItems)))
	for _, progress := range progressItems {
		if continueOnly && (progress.Finished || progress.Dismissed || progress.Position <= 0 || progress.Duration > 0 && progress.Position/progress.Duration >= .95) {
			continue
		}
		library, ok := m.library(progress.LibraryID)
		if !ok || library.Video == nil {
			continue
		}
		item, ok := library.Video.Item(progress.ItemID)
		if ok {
			items = append(items, m.videoView(account, library, item))
			if continueOnly && len(items) == limit {
				break
			}
		}
	}
	return items
}
