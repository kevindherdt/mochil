package web

import (
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"mochil/internal/audio"
	"mochil/internal/podcast"
)

// apiAudioProgress owns resumable listening state independently from catalogue
// listing and analysis, including optimistic cross-device revisions.
func (m *multiServer) apiAudioProgress(w http.ResponseWriter, r *http.Request) {
	if m.audioProgress == nil {
		http.Error(w, "Audio progress unavailable", http.StatusNotFound)
		return
	}
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/audio-progress"), "/")
	if suffix == "" {
		m.listAudioProgress(w, r)
		return
	}
	parts := strings.SplitN(suffix, "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	libraryID, libraryErr := url.PathUnescape(parts[0])
	itemID, itemErr := url.PathUnescape(parts[1])
	library, ok := m.library(libraryID)
	if libraryErr != nil || itemErr != nil || !ok || library.Audio == nil || !m.audioProgressItemExists(library, itemID) {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		saved, exists := m.audioProgress.Get(m.readingAccount(r), libraryID, itemID)
		if !exists {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	case http.MethodPut:
		m.putAudioProgress(w, r, libraryID, itemID)
	case http.MethodPatch, http.MethodPost:
		m.patchAudioProgress(w, r, library, itemID)
	case http.MethodDelete:
		if err := m.audioProgress.Delete(m.readingAccount(r), libraryID, itemID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) patchAudioProgress(w http.ResponseWriter, r *http.Request, library Library, itemID string) {
	var value struct {
		Finished  *bool `json:"finished"`
		Dismissed *bool `json:"dismissed"`
	}
	if decodeJSONRequest(w, r, 4096, &value, false) != nil || value.Finished == nil && value.Dismissed == nil {
		http.Error(w, "Invalid progress status", http.StatusBadRequest)
		return
	}
	duration := 0.0
	if item, ok := library.Audio.Item(itemID); ok {
		duration = item.Duration
	} else if library.Kind == "podcasts" && m.podcasts != nil {
		if episode, ok := m.podcasts.Episode(itemID); ok {
			duration = podcast.DurationSeconds(episode.Duration)
		}
	}
	saved, err := m.audioProgress.PatchStatus(m.readingAccount(r), library.ID, itemID, duration, value.Finished, value.Dismissed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (m *multiServer) audioProgressItemExists(library Library, itemID string) bool {
	if _, local := library.Audio.Item(itemID); local {
		return true
	}
	if library.Kind != "podcasts" || m.podcasts == nil {
		return false
	}
	_, exists := m.podcasts.Episode(itemID)
	return exists
}

func (m *multiServer) listAudioProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	account := m.readingAccount(r)
	progress := m.audioProgress.List(account)
	sort.Slice(progress, func(i, j int) bool { return progress[i].UpdatedAt > progress[j].UpdatedAt })
	historyLibraryID := strings.TrimSpace(r.URL.Query().Get("music_library"))
	historyLimit := 0
	continueOnly := r.URL.Query().Get("continue") == "1"
	continueLimit := 0
	if continueOnly {
		continueLimit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
		if continueLimit < 1 || continueLimit > 20 {
			continueLimit = 5
		}
	}
	if historyLibraryID != "" {
		library, ok := m.library(historyLibraryID)
		if !ok || library.Audio == nil || library.Kind != "music" {
			http.Error(w, "Unknown music library", http.StatusBadRequest)
			return
		}
		historyLimit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
		if historyLimit < 1 || historyLimit > 20 {
			historyLimit = 8
		}
	}
	writeJSON(w, http.StatusOK, m.audioProgressViews(account, progress, historyLibraryID, historyLimit, continueOnly, continueLimit))
}

func (m *multiServer) audioProgressViews(account string, progress []audio.Progress, historyLibraryID string, historyLimit int, continueOnly bool, continueLimit int) []audioView {
	out := make([]audioView, 0, len(progress))
	musicIncluded := false
	for _, saved := range progress {
		if continueOnly && (saved.Finished || saved.Dismissed || saved.Position <= 0) {
			continue
		}
		if historyLibraryID != "" && saved.LibraryID != historyLibraryID {
			continue
		}
		library, ok := m.library(saved.LibraryID)
		if !ok || library.Audio == nil {
			continue
		}
		if continueOnly && library.Kind != "audiobooks" && library.Kind != "podcasts" {
			continue
		}
		if historyLibraryID == "" && library.Kind == "music" {
			if musicIncluded {
				continue
			}
			musicIncluded = true
		}
		item, local := library.Audio.Item(saved.ItemID)
		var view audioView
		if local {
			view = m.audioView(account, library, item)
		} else if library.Kind == "podcasts" && m.podcasts != nil {
			episode, exists := m.podcasts.Episode(saved.ItemID)
			if !exists {
				continue
			}
			view = m.podcastView(account, library, episode)
		} else {
			continue
		}
		view.Progress = &saved
		out = append(out, view)
		if continueLimit > 0 && len(out) == continueLimit {
			break
		}
		if historyLimit > 0 && len(out) == historyLimit {
			break
		}
	}
	return out
}

func (m *multiServer) putAudioProgress(w http.ResponseWriter, r *http.Request, libraryID, itemID string) {
	var value struct {
		audio.Progress
		ExpectedRevision *uint64 `json:"expected_revision"`
		ExplicitSeek     bool    `json:"explicit_seek"`
	}
	if decodeJSONRequest(w, r, 4096, &value, false) != nil {
		http.Error(w, "Invalid progress", http.StatusBadRequest)
		return
	}
	value.LibraryID, value.ItemID = libraryID, itemID
	var err error
	if value.ExpectedRevision == nil {
		err = m.audioProgress.Put(m.readingAccount(r), value.Progress)
	} else if value.ExplicitSeek {
		err = m.audioProgress.PutIfRevisionExplicit(m.readingAccount(r), value.Progress, *value.ExpectedRevision)
	} else {
		err = m.audioProgress.PutIfRevision(m.readingAccount(r), value.Progress, *value.ExpectedRevision)
	}
	if errors.Is(err, audio.ErrProgressConflict) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	saved, _ := m.audioProgress.Get(m.readingAccount(r), libraryID, itemID)
	writeJSON(w, http.StatusOK, saved)
}
