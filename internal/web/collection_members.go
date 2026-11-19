package web

import (
	"errors"
	"os"
	"time"

	"mochil/internal/catalog"
)

func (s *collectionStore) createWithMembers(account, name, libraryID string, bookIDs []string) (savedCollection, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := cloneCollectionData(s.data)
	members := make([]collectionMember, 0, len(bookIDs))
	seen := make(map[string]bool, len(bookIDs))
	for _, book := range bookIDs {
		if book == "" || seen[book] {
			continue
		}
		seen[book] = true
		members = append(members, collectionMember{Type: "book", LibraryID: libraryID, BookID: book})
	}
	c := savedCollection{ID: collectionID(), Name: name, LibraryID: libraryID, Members: members, CreatedAt: time.Now().UnixMilli()}
	s.data.Accounts[account] = append(s.data.Accounts[account], c)
	if err := s.persistLocked(previous); err != nil {
		return savedCollection{}, 0, err
	}
	return c, len(members), nil
}

func (s *collectionStore) remove(account, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.data.Accounts[account]
	for i := range items {
		if items[i].ID == id {
			previous := cloneCollectionData(s.data)
			s.data.Accounts[account] = append(items[:i], items[i+1:]...)
			return s.persistLocked(previous)
		}
	}
	return os.ErrNotExist
}

var errSmartCollection = errors.New("a smart collection fills itself from its search")

func sameMember(a, b collectionMember) bool {
	return a.Type == b.Type && a.LibraryID == b.LibraryID && a.BookID == b.BookID && a.Series == b.Series
}
func (s *collectionStore) member(account, id string, member collectionMember, add bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Accounts[account] {
		c := &s.data.Accounts[account][i]
		if c.ID != id {
			continue
		}
		for n, v := range c.Members {
			if sameMember(v, member) {
				if add {
					return nil
				}
				previous := cloneCollectionData(s.data)
				c.Members = append(c.Members[:n], c.Members[n+1:]...)
				return s.persistLocked(previous)
			}
		}
		if c.Rule != nil {
			return errSmartCollection
		}
		if c.LibraryID != "" && member.LibraryID != c.LibraryID {
			return errWrongLibrary
		}
		if add {
			previous := cloneCollectionData(s.data)
			c.Members = append(c.Members, member)
			return s.persistLocked(previous)
		}
		return nil
	}
	return os.ErrNotExist
}

type resolvedCollectionBook struct {
	library Library
	book    catalog.Book
}
