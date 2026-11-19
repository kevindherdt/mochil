package web

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"mochil/internal/audio"
	"mochil/internal/catalog"
)

const standaloneAudiobooks = "Standalone audiobooks"

type audiobookSeriesSummary struct {
	Name       string `json:"name"`
	Count      int    `json:"count"`
	Artist     string `json:"artist,omitempty"`
	ArtworkURL string `json:"artwork_url,omitempty"`
}

func audiobookGroupName(item audio.Item) string {
	if value := strings.TrimSpace(item.Series); value != "" {
		return value
	}
	if value := strings.TrimSpace(item.Album); value != "" {
		return value
	}
	return standaloneAudiobooks
}

func matchesAudioQuery(item audio.Item, query string) bool {
	return query == "" || strings.Contains(strings.ToLower(item.Title+" "+item.Artist+" "+item.Album+" "+item.Series+" "+item.Path), query)
}

func (m *multiServer) apiAudiobookSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok || library.Kind != "audiobooks" || library.Audio == nil {
		http.Error(w, "Unknown audiobook library", http.StatusNotFound)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	sortOrder := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort")))
	items, _ := library.Audio.Snapshot()
	sortAudioItems(items, sortOrder, library.Kind)
	groups := make([]audiobookSeriesSummary, 0)
	positions := make(map[string]int)
	for _, item := range items {
		if !matchesAudioQuery(item, query) {
			continue
		}
		name := audiobookGroupName(item)
		key := strings.ToLower(name)
		if position, exists := positions[key]; exists {
			groups[position].Count++
			continue
		}
		positions[key] = len(groups)
		groups = append(groups, audiobookSeriesSummary{
			Name:       name,
			Count:      1,
			Artist:     item.Artist,
			ArtworkURL: audioArtworkURL(library.ID, item, audio.ArtworkCard),
		})
	}
	if sortOrder == "" || sortOrder == "series" {
		sort.SliceStable(groups, func(i, j int) bool { return catalog.NaturalLess(groups[i].Name, groups[j].Name) })
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(len(groups)))
	writeJSON(w, http.StatusOK, groups)
}
