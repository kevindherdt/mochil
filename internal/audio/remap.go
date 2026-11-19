package audio

// Remapping follows tracks whose file was renamed: a track's id comes from its path, so the queue, favourites,
// playlists and play progress that name the old id must name the new one or the track is lost from them.
// Each store saves the whole change in one write or changes nothing.

// Remap maps old track ids to new ones. Applying the reverse map undoes it.
type Remap map[string]string

// Reverse returns the map that undoes r.
func (r Remap) Reverse() Remap {
	back := make(Remap, len(r))
	for from, to := range r {
		back[to] = from
	}
	return back
}

func (r Remap) entry(library string, e QueueEntry) (QueueEntry, bool) {
	if e.LibraryID != library {
		return e, false
	}
	if to, ok := r[e.ItemID]; ok {
		e.ItemID = to
		return e, true
	}
	return e, false
}

// Remap rewrites matching entries of every account.
func (s *QueueStore) Remap(library string, ids Remap) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := queueData{Version: s.data.Version, Accounts: make(map[string][]QueueEntry, len(s.data.Accounts))}
	changed := false
	for account, entries := range s.data.Accounts {
		out := append([]QueueEntry(nil), entries...)
		for i, e := range out {
			if moved, ok := ids.entry(library, e); ok {
				out[i], changed = moved, true
			}
		}
		next.Accounts[account] = out
	}
	if !changed {
		return nil
	}
	if err := s.save(next); err != nil {
		return err
	}
	s.data = next
	return nil
}

// Remap rewrites matching entries of every playlist of every account.
func (s *PlaylistStore) Remap(library string, ids Remap) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := playlistData{Version: s.data.Version, Accounts: make(map[string][]Playlist, len(s.data.Accounts))}
	changed := false
	for account, lists := range s.data.Accounts {
		out := make([]Playlist, len(lists))
		for i, list := range lists {
			entries := append([]QueueEntry(nil), list.Entries...)
			for j, e := range entries {
				if moved, ok := ids.entry(library, e); ok {
					entries[j], changed = moved, true
				}
			}
			out[i] = Playlist{Name: list.Name, Entries: entries}
		}
		next.Accounts[account] = out
	}
	if !changed {
		return nil
	}
	if err := s.save(next); err != nil {
		return err
	}
	s.data = next
	return nil
}

// Remap moves each account's progress for a renamed track to the new id. If the new id already has progress (the
// track was played after a rescan), the more recent of the two is kept (on a tie, the one already under the new id).
func (s *ProgressStore) Remap(library string, ids Remap) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := progressData{Version: s.data.Version, Accounts: make(map[string]map[string]Progress, len(s.data.Accounts))}
	changed := false
	for account, entries := range s.data.Accounts {
		out := make(map[string]Progress, len(entries))
		for key, p := range entries {
			out[key] = p
		}
		for from, to := range ids {
			old, ok := out[progressKey(library, from)]
			if !ok {
				continue
			}
			delete(out, progressKey(library, from))
			old.ItemID = to
			if existing, taken := out[progressKey(library, to)]; !taken || old.UpdatedAt > existing.UpdatedAt {
				out[progressKey(library, to)] = old
			}
			changed = true
		}
		next.Accounts[account] = out
	}
	if !changed {
		return nil
	}
	saved := s.data
	s.data = next
	if err := s.save(); err != nil {
		s.data = saved
		return err
	}
	s.version++
	return nil
}
