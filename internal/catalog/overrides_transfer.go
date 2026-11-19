package catalog

import (
	"errors"
	"strings"
)

// BookID is the identity a publication gets from its path relative to its library folder; a book moved to the same
// relative path in another library keeps it.
func BookID(relativePath string) string { return bookID(relativePath) }

// overrideCell is one library/book slot of the override store, captured so a transfer can be taken back without
// disturbing corrections written since.
type overrideCell struct {
	library string
	book    string
	value   MetadataOverride
	had     bool
}

// Transfer carries the corrections made to books in one library over to the same books in another (ids maps the old
// book ID to the new one). With move the corrections leave the source library; without, they are copied. The returned
// action puts both libraries back as they were.
func (s *OverrideStore) Transfer(from, to string, ids map[string]string, move bool) (func() error, error) {
	if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" || from == to {
		return nil, errors.New("a transfer needs two different libraries")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Only the cells this transfer writes are remembered. Restoring a whole-store
	// snapshot would also revert every correction made between the transfer and its
	// undo, which is silent data loss.
	prior := make([]overrideCell, 0, len(ids)*2)
	capture := func(library, book string) {
		value, had := s.data.Libraries[library][book]
		prior = append(prior, overrideCell{library: library, book: book, value: value, had: had})
	}
	apply := func(cells []overrideCell) {
		for _, cell := range cells {
			books := s.data.Libraries[cell.library]
			if cell.had {
				if books == nil {
					books = map[string]MetadataOverride{}
					s.data.Libraries[cell.library] = books
				}
				books[cell.book] = cell.value
				continue
			}
			if books == nil {
				continue
			}
			delete(books, cell.book)
			if len(books) == 0 {
				delete(s.data.Libraries, cell.library)
			}
		}
	}
	moved := 0
	for oldID, newID := range ids {
		value, ok := s.data.Libraries[from][oldID]
		if !ok {
			continue
		}
		capture(to, newID)
		if move {
			capture(from, oldID)
		}
		if s.data.Libraries[to] == nil {
			s.data.Libraries[to] = map[string]MetadataOverride{}
		}
		s.data.Libraries[to][newID] = value
		if move {
			delete(s.data.Libraries[from], oldID)
		}
		moved++
	}
	if moved == 0 {
		return func() error { return nil }, nil
	}
	if err := s.saveLocked(); err != nil {
		apply(prior)
		return nil, err
	}
	s.generation++
	return func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		// Remember what the transfer left, so a failed undo can leave memory
		// agreeing with the file that still holds it.
		current := make([]overrideCell, 0, len(prior))
		for _, cell := range prior {
			value, had := s.data.Libraries[cell.library][cell.book]
			current = append(current, overrideCell{library: cell.library, book: cell.book, value: value, had: had})
		}
		apply(prior)
		if err := s.saveLocked(); err != nil {
			apply(current)
			return err
		}
		s.generation++
		return nil
	}, nil
}
