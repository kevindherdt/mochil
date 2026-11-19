package web

import (
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"mochil/internal/catalog"
	catalogquery "mochil/internal/query"
	"mochil/internal/video"
)

type videoView struct {
	video.Item
	LibraryName string                  `json:"library_name"`
	StreamURL   string                  `json:"stream_url,omitempty"`
	HLSURL      string                  `json:"hls_url,omitempty"`
	Progress    *video.Progress         `json:"progress,omitempty"`
	ArtworkURL  string                  `json:"artwork_url,omitempty"`
	BackdropURL string                  `json:"backdrop_url,omitempty"`
	Year        int                     `json:"year,omitempty"`
	Overview    string                  `json:"overview,omitempty"`
	Metadata    *video.MetadataOverride `json:"metadata_override,omitempty"`
}

type videoSeasonView struct {
	Number     int    `json:"number"`
	Count      int    `json:"count"`
	ArtworkURL string `json:"artwork_url,omitempty"`
}

type videoShowView struct {
	Name             string                  `json:"name"`
	SourceName       string                  `json:"source_name"`
	ItemID           string                  `json:"item_id"`
	Count            int                     `json:"count"`
	WatchedCount     int                     `json:"watched_count,omitempty"`
	InProgressCount  int                     `json:"in_progress_count,omitempty"`
	Seasons          []videoSeasonView       `json:"seasons"`
	Other            bool                    `json:"other,omitempty"`
	ArtworkURL       string                  `json:"artwork_url,omitempty"`
	BackdropURL      string                  `json:"backdrop_url,omitempty"`
	Year             int                     `json:"year,omitempty"`
	Overview         string                  `json:"overview,omitempty"`
	MetadataOverride *video.MetadataOverride `json:"metadata_override,omitempty"`
}

func (m *multiServer) apiVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok || library.Video == nil {
		http.Error(w, "Unknown video library", http.StatusNotFound)
		return
	}
	limit := 120
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 && value <= catalogquery.MaxPageSize {
		limit = value
	}
	offset := 0
	if value, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && value > 0 {
		offset = value
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	showOverrides := map[string]video.MetadataOverride{}
	if m.videoMetadata != nil {
		for _, item := range library.Video.Items() {
			if item.Show == "" {
				continue
			}
			if value, exists := m.videoMetadata.GetShow(library.ID, item.Show); exists {
				showOverrides[item.Show] = value
			}
		}
	}
	displayShowName := func(name string) string {
		if value := showOverrides[name]; value.Title != "" {
			return value.Title
		}
		return name
	}
	show := strings.TrimSpace(r.URL.Query().Get("show"))
	other := r.URL.Query().Get("other") == "1"
	season, seasonSet := 0, false
	if value := r.URL.Query().Get("season"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			season, seasonSet = parsed, true
		}
	}
	items := library.Video.Items()
	filtered := make([]video.Item, 0, len(items))
	for _, item := range items {
		if other && item.Show != "" {
			continue
		}
		if show != "" && item.Show != show {
			continue
		}
		if seasonSet && item.Season != season {
			continue
		}
		if query == "" || strings.Contains(strings.ToLower(item.Title), query) || strings.Contains(strings.ToLower(item.Path), query) || strings.Contains(strings.ToLower(item.Show), query) || strings.Contains(strings.ToLower(displayShowName(item.Show)), query) {
			filtered = append(filtered, item)
		}
	}
	total := len(filtered)
	if offset > total {
		offset = total
	}
	end := min(offset+limit, total)
	out := make([]videoView, 0, end-offset)
	for _, item := range filtered[offset:end] {
		view := m.videoView(m.readingAccount(r), library, item)
		if show != "" {
			view = m.withEpisodeDetail(r.Context(), library, view)
		}
		out = append(out, view)
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	writeJSON(w, http.StatusOK, out)
}

func (m *multiServer) apiVideoShows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok || library.Video == nil || (library.Kind != "tv" && library.Kind != "anime") {
		http.Error(w, "Unknown episodic video library", http.StatusNotFound)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	progressByItem := map[string]video.Progress{}
	if m.videoProgress != nil {
		for _, saved := range m.videoProgress.List(m.readingAccount(r)) {
			if saved.LibraryID == library.ID {
				progressByItem[saved.ItemID] = saved
			}
		}
	}
	showOverrides := map[string]video.MetadataOverride{}
	if m.videoMetadata != nil {
		for _, item := range library.Video.Items() {
			if item.Show == "" {
				continue
			}
			if value, exists := m.videoMetadata.GetShow(library.ID, item.Show); exists {
				showOverrides[item.Show] = value
			}
		}
	}
	displayShowName := func(name string) string {
		if value := showOverrides[name]; value.Title != "" {
			return value.Title
		}
		return name
	}
	type counts struct {
		total   int
		watched int
		active  int
		seasons map[int]int
		items   map[int]string
		itemID  string
	}
	groups := map[string]*counts{}
	for _, item := range library.Video.Items() {
		if query != "" && !strings.Contains(strings.ToLower(item.Show+" "+displayShowName(item.Show)+" "+item.Title+" "+item.Path), query) {
			continue
		}
		name := item.Show
		if name == "" {
			name = "Other videos"
		}
		group := groups[name]
		if group == nil {
			group = &counts{seasons: map[int]int{}, items: map[int]string{}, itemID: item.ID}
			groups[name] = group
		}
		group.total++
		group.seasons[item.Season]++
		if group.items[item.Season] == "" {
			group.items[item.Season] = item.ID
		}
		if saved, exists := progressByItem[item.ID]; exists {
			if saved.Finished {
				group.watched++
			} else if saved.Position > 0 {
				group.active++
			}
		}
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i] == "Other videos" || names[j] == "Other videos" {
			return names[j] == "Other videos"
		}
		return catalog.SortKey(displayShowName(names[i])) < catalog.SortKey(displayShowName(names[j]))
	})
	total := len(names)
	offset := 0
	if value, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && value > 0 {
		offset = value
	}
	if offset > total {
		offset = total
	}
	end := total
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if limit, err := strconv.Atoi(rawLimit); err == nil && limit > 0 && limit <= catalogquery.MaxPageSize {
			end = min(offset+limit, total)
		}
	}
	out := make([]videoShowView, 0, end-offset)
	for _, name := range names[offset:end] {
		group := groups[name]
		numbers := make([]int, 0, len(group.seasons))
		for number := range group.seasons {
			numbers = append(numbers, number)
		}
		sort.Ints(numbers)
		seasons := make([]videoSeasonView, 0, len(numbers))
		for _, number := range numbers {
			seasonItem, _ := library.Video.Item(group.items[number])
			seasonArtwork := "/api/video-artwork/" + url.PathEscape(library.ID) + "/" + url.PathEscape(group.items[number]) + "?scope=season&v=" + url.QueryEscape(seasonItem.Revision)
			seasons = append(seasons, videoSeasonView{Number: number, Count: group.seasons[number], ArtworkURL: seasonArtwork})
		}
		item, _ := library.Video.Item(group.itemID)
		artworkURL := "/api/video-artwork/" + url.PathEscape(library.ID) + "/" + url.PathEscape(group.itemID) + "?scope=show&v=" + url.QueryEscape(item.Revision)
		displayName, backdropURL := name, ""
		var override *video.MetadataOverride
		if name != "Other videos" {
			if saved, exists := showOverrides[name]; exists {
				override = &saved
				if saved.Title != "" {
					displayName = saved.Title
				}
				if saved.ArtworkURL != "" {
					artworkURL = "/api/video-show-metadata-artwork/" + url.PathEscape(library.ID) + "?show=" + url.QueryEscape(name)
				}
				if saved.BackdropURL != "" {
					backdropURL = "/api/video-show-metadata-artwork/" + url.PathEscape(library.ID) + "?backdrop=1&show=" + url.QueryEscape(name)
				}
			}
		}
		view := videoShowView{Name: displayName, SourceName: name, ItemID: group.itemID, Count: group.total, WatchedCount: group.watched, InProgressCount: group.active, Seasons: seasons, Other: name == "Other videos", ArtworkURL: artworkURL, BackdropURL: backdropURL, MetadataOverride: override}
		if override != nil {
			view.Year, view.Overview = override.Year, override.Overview
		}
		out = append(out, view)
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	writeJSON(w, http.StatusOK, out)
}

func (m *multiServer) videoView(account string, library Library, item video.Item) videoView {
	streamURL := ""
	if item.DirectPlay {
		streamURL = "/videos/" + url.PathEscape(library.ID) + "/" + url.PathEscape(item.ID) + "/" + url.PathEscape(path.Base(item.Path))
	}
	hlsURL := ""
	if m.hls != nil && len(item.Audio) > 0 && len(item.Video) > 0 {
		audioIndex := item.Audio[0].Index
		for _, audio := range item.Audio {
			if audio.Default {
				audioIndex = audio.Index
				break
			}
		}
		hlsURL = "/api/video-hls/" + url.PathEscape(library.ID) + "/" + url.PathEscape(item.ID) + "/" + strconv.Itoa(audioIndex) + "/index.m3u8"
	}
	var progress *video.Progress
	if m.videoProgress != nil {
		if saved, exists := m.videoProgress.Get(account, library.ID, item.ID); exists {
			progress = &saved
		}
	}
	artworkURL := "/api/video-artwork/" + url.PathEscape(library.ID) + "/" + url.PathEscape(item.ID) + "?v=" + url.QueryEscape(item.Revision)
	backdropURL := ""
	var override *video.MetadataOverride
	if m.videoMetadata != nil {
		if saved, exists := m.videoMetadata.Get(library.ID, item.ID); exists {
			override = &saved
			if saved.Title != "" {
				item.Title = saved.Title
			}
			if saved.ArtworkURL != "" {
				artworkURL = "/api/video-metadata-artwork/" + url.PathEscape(library.ID) + "/" + url.PathEscape(item.ID)
			}
			if saved.BackdropURL != "" {
				backdropURL = "/api/video-metadata-artwork/" + url.PathEscape(library.ID) + "/" + url.PathEscape(item.ID) + "?backdrop=1"
			}
		}
	}
	view := videoView{Item: item, LibraryName: library.Name, StreamURL: streamURL, HLSURL: hlsURL, Progress: progress, ArtworkURL: artworkURL, BackdropURL: backdropURL, Metadata: override}
	if override != nil {
		view.Year, view.Overview = override.Year, override.Overview
	}
	return view
}

func (m *multiServer) streamVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.SplitN(strings.Trim(strings.TrimPrefix(r.URL.Path, "/videos/"), "/"), "/", 3)
	if len(parts) < 2 {
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
	file, item, err := library.Video.Open(itemID)
	if err != nil || !item.DirectPlay {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", item.MediaType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, path.Base(item.Path), item.Modified, file)
}
