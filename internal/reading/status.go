package reading

import (
	"errors"
	"sort"
	"time"
)

// A reader's own label for a book, on top of finished/unfinished. On hold and dropped books are hidden from the
// dashboard the same way a dismissed one is, so every list that already skips dismissed books skips them too.
const (
	StatusOnHold  = "on_hold"
	StatusDropped = "dropped"
)

var ErrUnknownStatus = errors.New("unknown reading status")

// SetStatus applies one of: unread, reading, read, on_hold, dropped.
func (s *Store) SetStatus(account, libraryID, bookID, status string) (Position, error) {
	switch status {
	case "read":
		if _, err := s.SetFinished(account, libraryID, bookID, true); err != nil {
			return Position{}, err
		}
		return s.update(account, libraryID, bookID, func(p *Position) { p.Status, p.Dismissed = "", false })
	case "reading":
		if _, err := s.SetFinished(account, libraryID, bookID, false); err != nil {
			return Position{}, err
		}
		return s.update(account, libraryID, bookID, func(p *Position) {
			p.Status, p.Dismissed = "", false
			if p.Fraction <= 0 {
				p.Fraction = .001 // started, so it counts as being read
			}
		})
	case StatusOnHold, StatusDropped:
		return s.update(account, libraryID, bookID, func(p *Position) {
			p.Status, p.Dismissed = status, true
			p.Finished, p.FinishedAt = false, 0
		})
	case "unread":
		return s.clear(account, libraryID, bookID)
	}
	return Position{}, ErrUnknownStatus
}

func (s *Store) update(account, libraryID, bookID string, change func(*Position)) (Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	positions := s.ensureAccountLocked(account)
	key := readingKey(libraryID, bookID)
	position := positions[key]
	position.LibraryID, position.BookID = libraryID, bookID
	change(&position)
	position.Revision++
	if position.UpdatedAt == 0 {
		position.UpdatedAt = time.Now().UnixMilli()
	}
	positions[key] = position
	s.markDirtyLocked()
	return clonePosition(position), nil
}

// clear forgets the place and every label: the book is unread again.
func (s *Store) clear(account, libraryID, bookID string) (Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	positions := s.data.Accounts[account]
	delete(positions, readingKey(libraryID, bookID))
	s.markDirtyLocked()
	return Position{LibraryID: libraryID, BookID: bookID}, nil
}

// ListHeld returns the books the reader put on hold or dropped, newest first.
func (s *Store) ListHeld(account string, limit int) []Position {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var held []Position
	for _, position := range s.data.Accounts[account] {
		if position.Status != "" {
			held = append(held, clonePosition(position))
		}
	}
	sort.Slice(held, func(i, j int) bool { return positionBefore(held[i], held[j]) })
	if limit > 0 && len(held) > limit {
		held = held[:limit]
	}
	return held
}
