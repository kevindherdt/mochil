package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"mochil/internal/accounts"
	"mochil/internal/audio"
	"mochil/internal/externallibrary"
	"mochil/internal/reading"
	"mochil/internal/statefile"
	"mochil/internal/video"
)

const maximumAccountExportBytes int64 = 128 << 20

type externalTitleExport struct {
	Title    externallibrary.Title      `json:"title"`
	Progress []externallibrary.Progress `json:"progress,omitempty"`
}

type accountDataExport struct {
	Version        int                   `json:"version"`
	ExportedAt     time.Time             `json:"exported_at"`
	Account        accounts.Summary      `json:"account"`
	Reading        reading.AccountExport `json:"reading"`
	Annotations    json.RawMessage       `json:"annotations,omitempty"`
	AudioProgress  []audio.Progress      `json:"audio_progress"`
	VideoProgress  []video.Progress      `json:"video_progress"`
	Queue          []audio.QueueEntry    `json:"queue"`
	Favorites      []audio.QueueEntry    `json:"favorites"`
	Playlists      []audio.Playlist      `json:"playlists"`
	Collections    []savedCollection     `json:"collections"`
	Wiki           json.RawMessage       `json:"wiki,omitempty"`
	ExternalTitles []externalTitleExport `json:"external_titles"`
}

func (m *multiServer) exportAccount(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	summary, found := m.accountSummary(id)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Account not found"})
		return
	}
	export := accountDataExport{
		Version: 1, ExportedAt: time.Now().UTC(), Account: summary,
		Reading:       reading.AccountExport{Version: 1, Items: []reading.ExportEntry{}},
		AudioProgress: []audio.Progress{}, VideoProgress: []video.Progress{}, Queue: []audio.QueueEntry{}, Favorites: []audio.QueueEntry{},
		Playlists: []audio.Playlist{}, Collections: []savedCollection{}, ExternalTitles: []externalTitleExport{},
	}
	if m.reading != nil {
		export.Reading = m.reading.ExportAccount(id)
	}
	if m.annotations != nil {
		data, err := m.annotations.ExportAccount(id)
		if err != nil {
			m.accountExportError(w, err)
			return
		}
		export.Annotations = data
	} else {
		export.Annotations = emptyAnnotationExport()
	}
	if m.audioProgress != nil {
		export.AudioProgress = m.audioProgress.List(id)
	}
	if m.videoProgress != nil {
		export.VideoProgress = m.videoProgress.List(id)
	}
	if m.audioQueue != nil {
		export.Queue = m.audioQueue.Get(id)
	}
	if m.audioFavorites != nil {
		export.Favorites = m.audioFavorites.Get(id)
	}
	if m.audioPlaylists != nil {
		export.Playlists = m.audioPlaylists.List(id)
	}
	if m.collections != nil {
		export.Collections = m.collections.list(id)
	}
	if m.wiki != nil {
		data, err := m.wiki.Export(id)
		if err != nil {
			m.accountExportError(w, err)
			return
		}
		export.Wiki = data
	}
	if m.externalLibrary != nil {
		for _, title := range m.externalLibrary.List(id) {
			export.ExternalTitles = append(export.ExternalTitles, externalTitleExport{Title: title, Progress: m.externalLibrary.ListProgress(id, title.ID)})
		}
	}
	sort.Slice(export.AudioProgress, func(i, j int) bool {
		return mediaExportLess(export.AudioProgress[i].LibraryID, export.AudioProgress[i].ItemID, export.AudioProgress[j].LibraryID, export.AudioProgress[j].ItemID)
	})
	sort.Slice(export.VideoProgress, func(i, j int) bool {
		return mediaExportLess(export.VideoProgress[i].LibraryID, export.VideoProgress[i].ItemID, export.VideoProgress[j].LibraryID, export.VideoProgress[j].ItemID)
	})
	var output bytes.Buffer
	if err := statefile.EncodeJSON(&output, maximumAccountExportBytes, export, true); err != nil {
		m.accountExportError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="bobobooks-account-export.json"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(output.Bytes())
	}
}

func (m *multiServer) accountSummary(id string) (accounts.Summary, bool) {
	for _, item := range m.accounts.List() {
		if item.ID == id {
			return item, true
		}
	}
	return accounts.Summary{}, false
}

func (m *multiServer) accountExportError(w http.ResponseWriter, err error) {
	_ = err
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Account data could not be exported"})
}

func emptyAnnotationExport() json.RawMessage {
	return json.RawMessage(`{"version":1,"publications":[]}`)
}

func mediaExportLess(leftLibrary, leftItem, rightLibrary, rightItem string) bool {
	if leftLibrary == rightLibrary {
		return leftItem < rightItem
	}
	return leftLibrary < rightLibrary
}
