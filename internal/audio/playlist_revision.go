package audio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

var ErrPlaylistChanged = errors.New("playlist changed on another device; reload it before editing")

// PlaylistRevision identifies the complete stored ordering, not a filtered media view.
// Content-based revisions require no migration of existing playlist files.
func PlaylistRevision(list Playlist) string {
	entries := list.Entries
	if entries == nil {
		entries = []QueueEntry{}
	}
	data, _ := json.Marshal(Playlist{Name: strings.Join(strings.Fields(list.Name), " "), Entries: entries})
	sum := sha256.Sum256(data)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

// PutIfRevision is opt-in for upgraded clients. Empty expected preserves legacy PUT.
func (s *PlaylistStore) PutIfRevision(account, name string, entries []QueueEntry, expected string) error {
	return s.put(account, name, entries, expected)
}

func (s *PlaylistStore) RenameIfRevision(account, name, newName, expected string) error {
	return s.rename(account, name, newName, expected)
}

func (s *PlaylistStore) RemoveIfRevision(account, name, expected string) error {
	return s.remove(account, name, expected)
}

// Call only while holding the mutation lock; a missing list fails a condition.
func matchesPlaylistRevision(lists []Playlist, name, expected string) bool {
	if expected == "" {
		return true // Legacy clients have no revision contract.
	}
	for _, list := range lists {
		if strings.EqualFold(list.Name, name) {
			return PlaylistRevision(list) == expected
		}
	}
	return false
}
