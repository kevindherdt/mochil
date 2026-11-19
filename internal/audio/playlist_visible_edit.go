package audio

import "errors"

// MergeVisiblePlaylistEdit replaces only the entries the client actually saw.
// Unseen entries retain their slots; surplus replacements follow the old slots.
// The caller must commit against the revision of this exact stored snapshot.
func MergeVisiblePlaylistEdit(stored, visible, replacement []QueueEntry) ([]QueueEntry, error) {
	existing := make(map[QueueEntry]int, len(stored))
	for _, entry := range stored {
		existing[entry]++
	}
	seen := make(map[QueueEntry]int, len(visible))
	for _, entry := range visible {
		if seen[entry] >= existing[entry] {
			return nil, errors.New("invalid visible playlist snapshot")
		}
		seen[entry]++
	}
	for _, entry := range replacement {
		if existing[entry] > 0 && seen[entry] == 0 {
			return nil, errors.New("replacement overlaps unseen playlist entries")
		}
	}
	result := make([]QueueEntry, 0, len(stored)+len(replacement))
	next := 0
	for _, entry := range stored {
		if seen[entry] == 0 {
			result = append(result, entry)
		} else {
			seen[entry]--
			if next < len(replacement) {
				result = append(result, replacement[next])
				next++
			}
		}
	}
	return append(result, replacement[next:]...), nil
}
