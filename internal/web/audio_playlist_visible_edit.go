package web

import (
	"errors"
	"mochil/internal/audio"
	"net/http"
)

// Visible edits carry their original visible identities, not a fresh availability
// check: files can disappear or return between the client's GET and its edit.
func (m *multiServer) editVisiblePlaylist(w http.ResponseWriter, r *http.Request, account, name string) {
	expected := r.Header.Get("If-Match")
	if expected == "" {
		http.Error(w, "A playlist revision is required", http.StatusPreconditionRequired)
		return
	}
	var edit struct {
		Visible []audio.QueueEntry `json:"visible_entries"`
		Entries []audio.QueueEntry `json:"entries"`
	}
	if decodeJSONRequest(w, r, 2<<20, &edit, false) != nil || edit.Visible == nil || edit.Entries == nil {
		http.Error(w, "Invalid visible playlist edit", http.StatusBadRequest)
		return
	}
	stored, found := m.audioPlaylists.Get(account, name)
	if !found || audio.PlaylistRevision(stored) != expected {
		http.Error(w, audio.ErrPlaylistChanged.Error(), http.StatusPreconditionFailed)
		return
	}
	existing := make(map[audio.QueueEntry]bool, len(stored.Entries))
	for _, entry := range stored.Entries {
		existing[entry] = true
	}
	for _, entry := range edit.Entries {
		if !existing[entry] && !m.validMusicEntry(entry) {
			http.Error(w, "Playlist contains an unknown music item", http.StatusBadRequest)
			return
		}
	}
	merged, err := audio.MergeVisiblePlaylistEdit(stored.Entries, edit.Visible, edit.Entries)
	if err == nil {
		err = m.audioPlaylists.PutIfRevision(account, stored.Name, merged, expected)
	}
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, audio.ErrPlaylistChanged) {
			status = http.StatusPreconditionFailed
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("ETag", audio.PlaylistRevision(audio.Playlist{Name: stored.Name, Entries: merged}))
	w.WriteHeader(http.StatusNoContent)
}
