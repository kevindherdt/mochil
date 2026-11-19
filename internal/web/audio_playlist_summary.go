package web

import (
	"net/http"
	"slices"

	"mochil/internal/audio"
)

type audioPlaylistSummary struct {
	Name        string   `json:"name"`
	ItemCount   int      `json:"item_count"`
	ArtworkURLs []string `json:"artwork_urls"`
}

// Summaries intentionally have no items field: artwork previews are not tracks
// and must never be submitted as a replacement playlist.
type audioPlaylistSummaries struct {
	Version   int                    `json:"version"`
	Playlists []audioPlaylistSummary `json:"playlists"`
}

func (m *multiServer) writePlaylistSummaries(w http.ResponseWriter, lists []audio.Playlist) {
	out := audioPlaylistSummaries{Version: 1, Playlists: make([]audioPlaylistSummary, 0, len(lists))}
	for _, list := range lists {
		summary := audioPlaylistSummary{Name: list.Name, ArtworkURLs: make([]string, 0, 4)}
		for _, entry := range list.Entries {
			library, ok := m.library(entry.LibraryID)
			if !ok || library.Audio == nil || library.Kind != "music" {
				continue
			}
			item, ok := library.Audio.Item(entry.ItemID)
			if !ok {
				continue
			}
			summary.ItemCount++
			if len(summary.ArtworkURLs) < 4 {
				artwork := audioArtworkURL(library.ID, item, audio.ArtworkCard)
				if !slices.Contains(summary.ArtworkURLs, artwork) {
					summary.ArtworkURLs = append(summary.ArtworkURLs, artwork)
				}
			}
		}
		out.Playlists = append(out.Playlists, summary)
	}
	writeJSON(w, http.StatusOK, out)
}
