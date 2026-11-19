package reading

import "sort"

type ExportEntry struct {
	Position Position   `json:"position"`
	History  []Position `json:"history,omitempty"`
}

type AccountExport struct {
	Version int           `json:"version"`
	Items   []ExportEntry `json:"items"`
}

// ExportAccount returns all current and recent reading positions for one
// account, including dismissed items that ordinary Continue views omit.
func (s *Store) ExportAccount(account string) AccountExport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := AccountExport{Version: 1, Items: make([]ExportEntry, 0, len(s.data.Accounts[account]))}
	for key, position := range s.data.Accounts[account] {
		result.Items = append(result.Items, ExportEntry{Position: clonePosition(position), History: clonePositions(s.data.History[account][key])})
	}
	sort.Slice(result.Items, func(i, j int) bool {
		left, right := result.Items[i].Position, result.Items[j].Position
		if left.LibraryID == right.LibraryID {
			return left.BookID < right.BookID
		}
		return left.LibraryID < right.LibraryID
	})
	return result
}

func (s *Store) History(account, libraryID, bookID string) []Position {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clonePositions(s.data.History[account][readingKey(libraryID, bookID)])
}

func (s *Store) Get(account, libraryID, bookID string) (Position, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	position, ok := s.data.Accounts[account][readingKey(libraryID, bookID)]
	return clonePosition(position), ok
}

func (s *Store) FindKoSyncDocument(account, document string) (Position, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, position := range s.data.Accounts[account] {
		if position.KoSyncDocument == document || containsString(position.KoSyncDocuments, document) {
			return clonePosition(position), true
		}
	}
	return Position{}, false
}

func (s *Store) List(account string, finished bool) []Position {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listPositions(s.data.Accounts[account], finished, 0, "")
}

// ListLimit returns only the newest matching positions. The working set stays
// bounded by limit, so small navigation surfaces do not clone an entire large
// account history merely to show its first rows. A non-positive limit means
// unbounded and retains List's export-oriented behavior.
func (s *Store) ListLimit(account string, finished bool, limit int) []Position {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listPositions(s.data.Accounts[account], finished, limit, "")
}

// ListLibraryLimit returns newest positions for one shelf without cloning or
// sorting unrelated account history.
// Started reports whether the reader got somewhere. Opening a book on page one to look at it
// is not reading, and the continue shelf is only for books actually in progress.
func (p Position) Started() bool { return p.Fraction > 0 }

// Parked reports whether the reader explicitly put the book aside: on hold, or dropped.
func (p Position) Parked() bool { return p.Status == StatusOnHold || p.Status == StatusDropped }

// ListContinue returns what the reader is in the middle of: unfinished, not dismissed, not
// deliberately parked, and actually started — newest reading position first, so the shelf
// reads as "most recently read".
func (s *Store) ListContinue(account string, libraryID string, limit int) []Position {
	s.mu.RLock()
	defer s.mu.RUnlock()
	source := s.data.Accounts[account]
	candidates := make([]Position, 0, len(source))
	for _, position := range source {
		if position.Finished || position.Dismissed || position.Parked() || !position.Started() {
			continue
		}
		if libraryID != "" && position.LibraryID != libraryID {
			continue
		}
		candidates = append(candidates, position)
	}
	sort.SliceStable(candidates, func(i, j int) bool { return positionBefore(candidates[i], candidates[j]) })
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}
	for index := range candidates {
		candidates[index] = clonePosition(candidates[index])
	}
	return candidates
}

func (s *Store) ListLibraryLimit(account string, finished bool, libraryID string, limit int) []Position {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listPositions(s.data.Accounts[account], finished, limit, libraryID)
}

func (s *Store) Has(account string, finished bool) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, position := range s.data.Accounts[account] {
		if position.Finished == finished && !position.Dismissed {
			return true
		}
	}
	return false
}

func listPositions(source map[string]Position, finished bool, limit int, libraryID string) []Position {
	// Collect first, then sort once. The old insert-in-place loop copied the
	// matched slice per match, so an unbounded list (limit 0, which the KOReader
	// statistics and matching paths ask for) was quadratic in a large history
	// while holding the store's read lock.
	positions := make([]Position, 0, len(source))
	for _, position := range source {
		if position.Finished != finished || position.Dismissed || libraryID != "" && position.LibraryID != libraryID {
			continue
		}
		positions = append(positions, position)
	}
	sort.Slice(positions, func(left, right int) bool { return positionBefore(positions[left], positions[right]) })
	if limit > 0 && len(positions) > limit {
		positions = positions[:limit]
	}
	for index := range positions {
		positions[index] = clonePosition(positions[index])
	}
	return positions
}

func positionBefore(left, right Position) bool {
	if left.UpdatedAt != right.UpdatedAt {
		return left.UpdatedAt > right.UpdatedAt
	}
	if left.LibraryID != right.LibraryID {
		return left.LibraryID < right.LibraryID
	}
	return left.BookID < right.BookID
}
