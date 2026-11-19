package web

import (
	"errors"
	"os"
)

var (
	errNotSmartCollection = errors.New("only a smart collection has a search to change")
	errInvalidRule        = errors.New("a smart collection needs a search, and the search must be valid")
)

// change renames a collection (when name is not empty) and/or replaces the search of a smart collection, in one save.
func (s *collectionStore) change(account, id, name string, rule *collectionRule) (savedCollection, error) {
	if rule != nil {
		normal := rule.normalized()
		if !normal.valid() {
			return savedCollection{}, errInvalidRule
		}
		rule = &normal
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Accounts[account] {
		c := &s.data.Accounts[account][i]
		if c.ID != id {
			continue
		}
		if rule != nil && c.Rule == nil {
			return savedCollection{}, errNotSmartCollection
		}
		previous := cloneCollectionData(s.data)
		if name != "" {
			c.Name = name
		}
		if rule != nil {
			c.Rule = rule
		}
		return *c, s.persistLocked(previous)
	}
	return savedCollection{}, os.ErrNotExist
}
