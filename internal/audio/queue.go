package audio

import (
	"errors"
	"os"
	"slices"
	"sync"
)

type QueueEntry struct {
	LibraryID string `json:"library_id"`
	ItemID    string `json:"item_id"`
}
type queueData struct {
	Version  int                     `json:"version"`
	Accounts map[string][]QueueEntry `json:"accounts"`
}
type QueueStore struct {
	path string
	mu   sync.RWMutex
	data queueData
}

func OpenQueueStore(path string) (*QueueStore, error) {
	s := &QueueStore{path: path, data: queueData{1, map[string][]QueueEntry{}}}
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) || path == "" {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err = decodeAudioState(path, &s.data); err != nil || s.data.Version != 1 || s.data.Accounts == nil {
		if err == nil {
			err = errors.New("unsupported audio queue version")
		}
		return nil, err
	}
	for account, entries := range s.data.Accounts {
		if err = validateQueue(account, entries); err != nil {
			return nil, errors.New("invalid saved audio queue")
		}
	}
	return s, nil
}
func (s *QueueStore) Get(account string) []QueueEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]QueueEntry(nil), s.data.Accounts[account]...)
}
func (s *QueueStore) Put(account string, entries []QueueEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putLocked(account, entries)
}
func (s *QueueStore) putLocked(account string, entries []QueueEntry) error {
	// Every mutation must obey the same bound as persisted-state decoding.
	if err := validateQueue(account, entries); err != nil {
		return err
	}
	if slices.Equal(s.data.Accounts[account], entries) {
		return nil
	}
	next := queueData{Version: s.data.Version, Accounts: make(map[string][]QueueEntry, len(s.data.Accounts))}
	for key, value := range s.data.Accounts {
		next.Accounts[key] = value
	}
	next.Accounts[account] = append([]QueueEntry(nil), entries...)
	if err := s.save(next); err != nil {
		return err
	}
	s.data = next
	return nil
}
func validateQueue(account string, entries []QueueEntry) error {
	if account == "" || len(entries) > 10000 {
		return errors.New("invalid audio queue")
	}
	for _, entry := range entries {
		if entry.LibraryID == "" || entry.ItemID == "" {
			return errors.New("invalid audio queue entry")
		}
	}
	return nil
}
func (s *QueueStore) Add(account string, entry QueueEntry) error {
	if err := validateQueue(account, []QueueEntry{entry}); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Accounts[account] {
		if existing == entry {
			return nil
		}
	}
	entries := append([]QueueEntry(nil), s.data.Accounts[account]...)
	return s.putLocked(account, append(entries, entry))
}
func (s *QueueStore) Remove(account string, entry QueueEntry) error {
	if err := validateQueue(account, []QueueEntry{entry}); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := append([]QueueEntry(nil), s.data.Accounts[account]...)
	out := entries[:0]
	for _, existing := range entries {
		if existing != entry {
			out = append(out, existing)
		}
	}
	return s.putLocked(account, out)
}
func (s *QueueStore) DeleteAccount(account string) error {
	if account == "" {
		return errors.New("invalid account")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data.Accounts[account]; !exists {
		return nil
	}
	next := queueData{Version: s.data.Version, Accounts: make(map[string][]QueueEntry, len(s.data.Accounts))}
	for key, value := range s.data.Accounts {
		if key != account {
			next.Accounts[key] = value
		}
	}
	if err := s.save(next); err != nil {
		return err
	}
	s.data = next
	return nil
}
func (s *QueueStore) save(data queueData) error {
	return saveAudioState(s.path, ".audio-queue-*", maximumAudioStateBytes, data)
}
