package web

import (
	"errors"
	"net/http"
	"strings"

	"mochil/internal/audio"
)

func (m *multiServer) apiAudioQueue(w http.ResponseWriter, r *http.Request) {
	if m.audioQueue == nil {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		entries := m.audioQueue.Get(m.readingAccount(r))
		out := make([]audioView, 0, len(entries))
		for _, entry := range entries {
			library, ok := m.library(entry.LibraryID)
			if !ok || library.Audio == nil {
				continue
			}
			item, ok := library.Audio.Item(entry.ItemID)
			if !ok {
				continue
			}
			out = append(out, m.audioView(m.readingAccount(r), library, item))
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPut:
		var entries []audio.QueueEntry
		if decodeJSONRequest(w, r, 2<<20, &entries, false) != nil {
			http.Error(w, "Invalid queue", http.StatusBadRequest)
			return
		}
		for _, entry := range entries {
			library, ok := m.library(entry.LibraryID)
			if !ok || library.Audio == nil || library.Kind != "music" {
				http.Error(w, "Queue contains an unknown music item", http.StatusBadRequest)
				return
			}
			if _, ok = library.Audio.Item(entry.ItemID); !ok {
				http.Error(w, "Queue contains an unknown music item", http.StatusBadRequest)
				return
			}
		}
		if err := m.audioQueue.Put(m.readingAccount(r), entries); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) apiAudioFavorites(w http.ResponseWriter, r *http.Request) {
	if m.audioFavorites == nil {
		http.NotFound(w, r)
		return
	}
	account := m.readingAccount(r)
	switch r.Method {
	case http.MethodGet:
		entries := m.audioFavorites.Get(account)
		out := make([]audioView, 0, len(entries))
		for _, entry := range entries {
			library, ok := m.library(entry.LibraryID)
			if !ok || library.Audio == nil {
				continue
			}
			item, ok := library.Audio.Item(entry.ItemID)
			if ok {
				out = append(out, m.audioView(account, library, item))
			}
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPut:
		var value struct {
			audio.QueueEntry
			Favorite bool `json:"favorite"`
		}
		if decodeJSONRequest(w, r, 4096, &value, false) != nil {
			http.Error(w, "Invalid favorite", http.StatusBadRequest)
			return
		}
		library, ok := m.library(value.LibraryID)
		if !ok || library.Audio == nil || library.Kind != "music" {
			http.Error(w, "Unknown music item", http.StatusBadRequest)
			return
		}
		if _, ok = library.Audio.Item(value.ItemID); !ok {
			http.Error(w, "Unknown music item", http.StatusBadRequest)
			return
		}
		var err error
		if value.Favorite {
			err = m.audioFavorites.Add(account, value.QueueEntry)
		} else {
			err = m.audioFavorites.Remove(account, value.QueueEntry)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

type audioPlaylistView struct {
	Name  string      `json:"name"`
	Items []audioView `json:"items"`
}

func (m *multiServer) apiAudioPlaylists(w http.ResponseWriter, r *http.Request) {
	if m.audioPlaylists == nil {
		http.NotFound(w, r)
		return
	}
	account := m.readingAccount(r)
	// URL.Path is already decoded by net/http. Unescaping again would change
	// literal names such as "100%25" or reject a valid percent sign.
	suffix := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/audio-playlists"), "/")
	if suffix == "" {
		m.audioPlaylistCollection(w, r, account)
		return
	}
	m.audioPlaylist(w, r, account, suffix)
}

func (m *multiServer) audioPlaylistCollection(w http.ResponseWriter, r *http.Request, account string) {
	switch r.Method {
	case http.MethodGet:
		lists := m.audioPlaylists.List(account)
		if r.URL.Query().Get("view") == "summary" {
			m.writePlaylistSummaries(w, lists)
			return
		}
		out := make([]audioPlaylistView, 0, len(lists))
		for _, list := range lists {
			out = append(out, m.playlistView(account, list))
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		var value struct {
			Name string `json:"name"`
		}
		if decodeJSONRequest(w, r, 4096, &value, false) != nil {
			http.Error(w, "Invalid playlist", http.StatusBadRequest)
			return
		}
		if err := m.audioPlaylists.Create(account, value.Name); err != nil {
			if errors.Is(err, audio.ErrPlaylistExists) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) audioPlaylist(w http.ResponseWriter, r *http.Request, account, name string) {
	if r.Method == http.MethodPut && r.URL.Query().Get("edit") == "visible" {
		m.editVisiblePlaylist(w, r, account, name)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("download") == "1" {
			m.downloadAudioPlaylist(w, r, account, name)
			return
		}
		list, found := m.audioPlaylists.Get(account, name)
		if !found {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("ETag", audio.PlaylistRevision(list))
		w.Header().Set("X-Mochil-Playlist-Edit", "visible-v1")
		writeJSON(w, http.StatusOK, m.playlistView(account, list))
	case http.MethodPatch:
		var value struct {
			Name string `json:"name"`
		}
		if decodeJSONRequest(w, r, 4096, &value, false) != nil {
			http.Error(w, "Invalid playlist name", http.StatusBadRequest)
			return
		}
		if err := m.audioPlaylists.RenameIfRevision(account, name, value.Name, r.Header.Get("If-Match")); err != nil {
			if errors.Is(err, audio.ErrPlaylistChanged) {
				http.Error(w, err.Error(), http.StatusPreconditionFailed)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPost:
		var entry audio.QueueEntry
		if decodeJSONRequest(w, r, 4096, &entry, false) != nil {
			http.Error(w, "Invalid playlist entry", http.StatusBadRequest)
			return
		}
		if !m.validMusicEntry(entry) {
			http.Error(w, "Playlist contains an unknown music item", http.StatusBadRequest)
			return
		}
		if err := m.audioPlaylists.Add(account, name, entry); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPut:
		var entries []audio.QueueEntry
		if decodeJSONRequest(w, r, 2<<20, &entries, false) != nil {
			http.Error(w, "Invalid playlist", http.StatusBadRequest)
			return
		}
		for _, entry := range entries {
			if !m.validMusicEntry(entry) {
				http.Error(w, "Playlist contains an unknown music item", http.StatusBadRequest)
				return
			}
		}
		if err := m.audioPlaylists.PutIfRevision(account, name, entries, r.Header.Get("If-Match")); err != nil {
			if errors.Is(err, audio.ErrPlaylistChanged) {
				http.Error(w, err.Error(), http.StatusPreconditionFailed)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("ETag", audio.PlaylistRevision(audio.Playlist{Name: name, Entries: entries}))
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := m.audioPlaylists.RemoveIfRevision(account, name, r.Header.Get("If-Match")); err != nil {
			if errors.Is(err, audio.ErrPlaylistChanged) {
				http.Error(w, err.Error(), http.StatusPreconditionFailed)
				return
			}
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *multiServer) validMusicEntry(entry audio.QueueEntry) bool {
	library, ok := m.library(entry.LibraryID)
	if !ok || library.Audio == nil || library.Kind != "music" {
		return false
	}
	_, ok = library.Audio.Item(entry.ItemID)
	return ok
}
