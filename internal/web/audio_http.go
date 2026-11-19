package web

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"mochil/internal/audio"
	"mochil/internal/catalog"
	"mochil/internal/jobs"
	"mochil/internal/podcast"
)

type audioView struct {
	audio.Item
	LibraryName      string          `json:"library_name"`
	LibraryKind      string          `json:"library_kind"`
	StreamURL        string          `json:"stream_url"`
	CompatibilityURL string          `json:"compatibility_url,omitempty"`
	DownloadURL      string          `json:"download_url"`
	ArtworkURL       string          `json:"artwork_url,omitempty"`
	PlayerArtworkURL string          `json:"player_artwork_url,omitempty"`
	Progress         *audio.Progress `json:"progress,omitempty"`
	FeedID           string          `json:"feed_id,omitempty"`
	Description      string          `json:"description,omitempty"`
	ServerDownloaded bool            `json:"server_downloaded,omitempty"`
}

type audioNavigationView struct {
	Bucket string `json:"bucket"`
	Offset int    `json:"offset"`
	Count  int    `json:"count"`
}

func (m *multiServer) apiAudioNavigation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok || library.Audio == nil || library.Kind != "music" {
		http.Error(w, "Unknown music library", http.StatusNotFound)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	sortOrder := r.URL.Query().Get("sort")
	mode := r.URL.Query().Get("mode")
	if mode == "albums" || mode == "artists" {
		groups := buildAudioGroups(library.Audio.Items(), library.ID, mode, query)
		labels := make([]string, 0, len(groups))
		for _, group := range groups {
			labels = append(labels, group.Name)
		}
		writeAudioNavigation(w, labels)
		return
	}
	items := make([]audio.Item, 0)
	for _, item := range library.Audio.Items() {
		if matchesAudioQuery(item, query) {
			items = append(items, item)
		}
	}
	sortAudioItems(items, sortOrder, library.Kind)
	labels := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		label := item.Title
		key := strings.ToLower(strings.TrimSpace(label))
		if label == "" || mode != "tracks" && seen[key] {
			continue
		}
		seen[key] = true
		labels = append(labels, label)
	}
	writeAudioNavigation(w, labels)
}

func writeAudioNavigation(w http.ResponseWriter, labels []string) {
	positions, counts := map[string]int{}, map[string]int{}
	for index, label := range labels {
		bucket := catalog.Bucket(label)
		if counts[bucket] == 0 {
			positions[bucket] = index
		}
		counts[bucket]++
	}
	out := make([]audioNavigationView, 0, len(counts))
	for _, character := range "#ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		bucket := string(character)
		if counts[bucket] > 0 {
			out = append(out, audioNavigationView{Bucket: bucket, Offset: positions[bucket], Count: counts[bucket]})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *multiServer) audioView(account string, library Library, item audio.Item) audioView {
	base := "/audio/" + url.PathEscape(library.ID) + "/" + url.PathEscape(item.ID) + "/" + url.PathEscape(path.Base(item.Path))
	var progress *audio.Progress
	// Spoken media resumes per item. Music exposes only its latest session through
	// the Continue Listening endpoint instead of making every track resumable.
	if m.audioProgress != nil && library.Kind != "music" {
		if saved, ok := m.audioProgress.Get(account, library.ID, item.ID); ok {
			progress = &saved
		}
	}
	artworkURL := audioArtworkURL(library.ID, item, audio.ArtworkCard)
	playerArtworkURL := audioArtworkURL(library.ID, item, audio.ArtworkPlayer)
	compatibilityURL := ""
	if m.audioHLS != nil {
		compatibilityURL = "/api/audio-hls/" + url.PathEscape(library.ID) + "/" + url.PathEscape(item.ID) + "/index.m3u8"
	}
	return audioView{Item: item, LibraryName: library.Name, LibraryKind: library.Kind, StreamURL: base, CompatibilityURL: compatibilityURL, DownloadURL: base + "?download=1", ArtworkURL: artworkURL, PlayerArtworkURL: playerArtworkURL, Progress: progress}
}

func audioArtworkURL(libraryID string, item audio.Item, variant audio.ArtworkVariant) string {
	return "/api/audio-artwork/" + url.PathEscape(libraryID) + "/" + url.PathEscape(item.ID) + "?size=" + url.QueryEscape(string(variant)) + "&v=" + url.QueryEscape(item.Revision)
}

func (m *multiServer) podcastView(account string, library Library, episode podcast.Episode) audioView {
	item := audio.Item{ID: episode.ID, LibraryID: library.ID, Title: episode.Title, Artist: episode.FeedTitle, Album: episode.FeedTitle, MediaType: episode.MediaType, Size: episode.Size, Modified: episode.Published, Duration: podcast.DurationSeconds(episode.Duration)}
	base := "/podcast-media/" + url.PathEscape(episode.ID)
	view := audioView{Item: item, LibraryName: library.Name, LibraryKind: library.Kind, StreamURL: base, DownloadURL: base + "?download=1", FeedID: episode.FeedID, Description: episode.Description}
	if episode.ArtworkURL != "" {
		view.ArtworkURL = "/api/podcast-artwork/" + url.PathEscape(episode.ID)
	}
	if m.audioProgress != nil {
		if saved, ok := m.audioProgress.Get(account, library.ID, episode.ID); ok {
			view.Progress = &saved
		}
	}
	return view
}

func matchesPlayedFilter(filter string, progress *audio.Progress) bool {
	played := progress != nil && progress.Finished
	switch filter {
	case "continue":
		return progress != nil && !progress.Finished && progress.Position > 0
	case "played":
		return played
	case "unplayed":
		return !played
	default:
		return true
	}
}

func (m *multiServer) apiAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok || library.Audio == nil {
		http.Error(w, "Unknown audio library", http.StatusNotFound)
		return
	}
	if library.Kind == "podcasts" && m.podcasts != nil {
		m.schedulePodcastRefresh()
	}
	limit := 120
	if n, e := strconv.Atoi(r.URL.Query().Get("limit")); e == nil && n > 0 && n <= 500 {
		limit = n
	}
	offset := 0
	if n, e := strconv.Atoi(r.URL.Query().Get("offset")); e == nil && n > 0 {
		offset = n
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	show := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("show")))
	feedID := strings.TrimSpace(r.URL.Query().Get("feed"))
	group := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("group")))
	sortOrder := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort")))
	playedFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("played")))
	showView := show
	if feedID != "" {
		showView = "feed:" + feedID
	}
	viewKey := audioViewKey(library.ID, query, showView, group, sortOrder, playedFilter)
	cursorValue := r.URL.Query().Get("cursor")
	cursor, cursorErr := decodeAudioCursor(cursorValue)
	if cursorErr != nil || (cursor.ItemID != "" && cursor.View != viewKey) {
		http.Error(w, "Audio cursor is invalid for this view", http.StatusBadRequest)
		return
	}
	localItems, audioVersion := library.Audio.Snapshot()
	var podcastEpisodes []podcast.Episode
	localPodcastEpisodes := map[string]audio.Item{}
	podcastVersion := uint64(0)
	if library.Kind == "podcasts" && m.podcasts != nil {
		podcastEpisodes, podcastVersion = m.podcasts.EpisodesSnapshot()
		localPodcastEpisodes = matchLocalPodcastEpisodes(localItems, podcastEpisodes)
	}
	progressByItem := make(map[string]audio.Progress)
	progressVersion := uint64(0)
	// Only filtering needs the account-wide snapshot. Visible spoken-media rows
	// retrieve their own progress below; ordinary pages need no full-history copy.
	if m.audioProgress != nil && playedFilter != "" {
		progressItems, version := m.audioProgress.Snapshot(m.readingAccount(r))
		progressVersion = version
		for _, saved := range progressItems {
			if saved.LibraryID == library.ID {
				progressByItem[saved.ItemID] = saved
			}
		}
	}
	filtered := make([]audio.Item, 0)
	remote := map[string]podcast.Episode{}
	for _, item := range localItems {
		if feedID != "" {
			continue
		}
		if show != "" && strings.ToLower(strings.TrimSpace(item.Album)) != show && strings.ToLower(strings.TrimSpace(item.Artist)) != show {
			continue
		}
		if group != "" && strings.ToLower(audiobookGroupName(item)) != group {
			continue
		}
		var progress *audio.Progress
		if saved, exists := progressByItem[item.ID]; exists {
			progress = &saved
		}
		if matchesPlayedFilter(playedFilter, progress) && matchesAudioQuery(item, query) {
			filtered = append(filtered, item)
		}
	}
	if library.Kind == "podcasts" && m.podcasts != nil {
		for _, episode := range podcastEpisodes {
			if feedID != "" && episode.FeedID != feedID {
				continue
			}
			if show != "" && strings.ToLower(strings.TrimSpace(episode.FeedTitle)) != show {
				continue
			}
			localItem, inLibrary := localPodcastEpisodes[episode.ID]
			serverDownloaded := inLibrary || m.podcastSavedOnServer(episode.ID)
			if playedFilter == "downloaded" && !serverDownloaded {
				continue
			}
			var progress *audio.Progress
			if saved, exists := progressByItem[episode.ID]; exists {
				progress = &saved
			}
			if matchesPlayedFilter(playedFilter, progress) && (query == "" || strings.Contains(strings.ToLower(episode.Title+" "+episode.FeedTitle+" "+episode.Description), query)) {
				item := audio.Item{ID: episode.ID, LibraryID: library.ID, Title: episode.Title, Artist: episode.FeedTitle, Album: episode.FeedTitle, Modified: episode.Published}
				if inLibrary {
					item.Path = localItem.Path
				}
				filtered = append(filtered, item)
				remote[item.ID] = episode
			}
		}
	}
	sortAudioItems(filtered, sortOrder, library.Kind)
	total := len(filtered)
	if cursor.ItemID != "" {
		if cursor.AudioVersion != audioVersion || cursor.PodcastVersion != podcastVersion || cursor.ProgressVersion != 0 && cursor.ProgressVersion != progressVersion {
			http.Error(w, "Audio cursor expired; reload the shelf", http.StatusConflict)
			return
		}
		position := -1
		for index := range filtered {
			if filtered[index].ID == cursor.ItemID {
				position = index
				break
			}
		}
		if position < 0 {
			http.Error(w, "Audio cursor expired; reload the shelf", http.StatusConflict)
			return
		}
		offset = position + 1
	} else if offset > total {
		offset = total
	}
	end := min(total, offset+limit)
	out := make([]audioView, 0, end-offset)
	for _, item := range filtered[offset:end] {
		if episode, ok := remote[item.ID]; ok {
			view := m.podcastView(m.readingAccount(r), library, episode)
			if localItem, downloaded := localPodcastEpisodes[episode.ID]; downloaded {
				localView := m.audioView(m.readingAccount(r), library, localItem)
				view.StreamURL = localView.StreamURL
				view.DownloadURL = localView.DownloadURL
				view.CompatibilityURL = localView.CompatibilityURL
				view.ServerDownloaded = true
			} else if m.podcastSavedOnServer(episode.ID) {
				view.ServerDownloaded = true
			}
			out = append(out, view)
		} else {
			out = append(out, m.audioView(m.readingAccount(r), library, item))
		}
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	if end < total && end > offset {
		cursorProgressVersion := uint64(0)
		if playedFilter != "" {
			cursorProgressVersion = progressVersion
		}
		w.Header().Set("X-Next-Cursor", encodeAudioCursor(audioCursor{AudioVersion: audioVersion, PodcastVersion: podcastVersion, ProgressVersion: cursorProgressVersion, View: viewKey, ItemID: filtered[end-1].ID}))
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *multiServer) apiAudioAnalysis(w http.ResponseWriter, r *http.Request) {
	if m.audioAnalysis == nil {
		http.Error(w, "Audio analysis is unavailable", http.StatusNotFound)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok || library.Audio == nil || (library.Kind != "music" && library.Kind != "audiobooks") {
		http.Error(w, "Unknown local audio library", http.StatusNotFound)
		return
	}
	key := "audio-analysis:" + library.ID
	switch r.Method {
	case http.MethodGet:
		var state any
		for _, candidate := range m.audioAnalysis.Jobs() {
			if candidate.Key == key {
				state = candidate
				break
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"missing": len(library.Audio.MissingAnalysis()), "job": state})
	case http.MethodPost:
		queued, err := m.audioAnalysis.Submit(library.Audio)
		if errors.Is(err, jobs.ErrDuplicate) {
			http.Error(w, "Analysis is already running", http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(w, "Analysis could not be queued", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"key": queued})
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
