package audio

import (
	"errors"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
)

type Playlist struct {
	Name    string       `json:"name"`
	Entries []QueueEntry `json:"entries"`
}
type playlistData struct {
	Version  int                   `json:"version"`
	Accounts map[string][]Playlist `json:"accounts"`
}
type PlaylistStore struct {
	path string
	mu   sync.RWMutex
	data playlistData
}

func OpenPlaylistStore(path string) (*PlaylistStore, error) {
	s := &PlaylistStore{path: path, data: playlistData{Version: 1, Accounts: map[string][]Playlist{}}}
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) || path == "" {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err = decodeAudioState(path, &s.data); err != nil || s.data.Version != 1 || s.data.Accounts == nil {
		return nil, errors.New("unsupported audio playlist version")
	}
	for account, lists := range s.data.Accounts {
		if account == "" || len(lists) > 1000 {
			return nil, errors.New("invalid saved playlists")
		}
		for _, list := range lists {
			clean, cleanErr := cleanPlaylistName(list.Name)
			if cleanErr != nil || clean != list.Name || validateQueue(account, list.Entries) != nil {
				return nil, errors.New("invalid saved playlists")
			}
		}
	}
	return s, nil
}
func cleanPlaylistName(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" || len([]rune(name)) > 80 {
		return "", errors.New("playlist name must be between 1 and 80 characters")
	}
	return name, nil
}
func (s *PlaylistStore) List(account string) []Playlist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Playlist(nil), s.data.Accounts[account]...)
	for i := range out {
		out[i].Entries = append([]QueueEntry(nil), out[i].Entries...)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

// Get snapshots one playlist without copying or sorting unrelated playlists.
// Names follow the same case-insensitive identity used by playlist mutations.
func (s *PlaylistStore) Get(account, name string) (Playlist, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, list := range s.data.Accounts[account] {
		if strings.EqualFold(list.Name, name) {
			return Playlist{Name: list.Name, Entries: slices.Clone(list.Entries)}, true
		}
	}
	return Playlist{}, false
}

func (s *PlaylistStore) Put(account, name string, entries []QueueEntry) error {
	return s.put(account, name, entries, "")
}

func (s *PlaylistStore) put(account, name string, entries []QueueEntry, expected string) error {
	name, err := cleanPlaylistName(name)
	if err != nil || account == "" || len(entries) > 10000 {
		if err != nil {
			return err
		}
		return errors.New("invalid playlist")
	}
	if err = validateQueue(account, entries); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lists := append([]Playlist(nil), s.data.Accounts[account]...)
	if !matchesPlaylistRevision(lists, name, expected) {
		return ErrPlaylistChanged
	}
	found := false
	for i := range lists {
		if strings.EqualFold(lists[i].Name, name) {
			if lists[i].Name == name && slices.Equal(lists[i].Entries, entries) {
				return nil
			}
			lists[i] = Playlist{Name: name, Entries: append([]QueueEntry(nil), entries...)}
			found = true
			break
		}
	}
	if !found {
		if len(lists) >= 1000 {
			return errors.New("playlist limit reached")
		}
		lists = append(lists, Playlist{Name: name, Entries: append([]QueueEntry(nil), entries...)})
	}
	next := s.copyData()
	next.Accounts[account] = lists
	if err = s.save(next); err != nil {
		return err
	}
	s.data = next
	return nil
}
func (s *PlaylistStore) Add(account, name string, entry QueueEntry) error {
	name, err := cleanPlaylistName(name)
	if err != nil || validateQueue(account, []QueueEntry{entry}) != nil {
		return errors.New("invalid playlist entry")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lists := append([]Playlist(nil), s.data.Accounts[account]...)
	found := false
	for i := range lists {
		if !strings.EqualFold(lists[i].Name, name) {
			continue
		}
		found = true
		for _, existing := range lists[i].Entries {
			if existing == entry {
				return nil
			}
		}
		if len(lists[i].Entries) >= 10000 {
			return errors.New("playlist item limit reached")
		}
		lists[i].Entries = append(append([]QueueEntry(nil), lists[i].Entries...), entry)
		break
	}
	if !found {
		if len(lists) >= 1000 {
			return errors.New("playlist limit reached")
		}
		lists = append(lists, Playlist{Name: name, Entries: []QueueEntry{entry}})
	}
	next := s.copyData()
	next.Accounts[account] = lists
	if err = s.save(next); err != nil {
		return err
	}
	s.data = next
	return nil
}
func (s *PlaylistStore) Rename(account, name, newName string) error {
	return s.rename(account, name, newName, "")
}
func (s *PlaylistStore) rename(account, name, newName, expected string) error {
	name, err := cleanPlaylistName(name)
	if err != nil {
		return err
	}
	newName, err = cleanPlaylistName(newName)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lists := append([]Playlist(nil), s.data.Accounts[account]...)
	index := -1
	if !matchesPlaylistRevision(lists, name, expected) {
		return ErrPlaylistChanged
	}
	for i := range lists {
		if strings.EqualFold(lists[i].Name, newName) && !strings.EqualFold(lists[i].Name, name) {
			return errors.New("playlist name already exists")
		}
		if strings.EqualFold(lists[i].Name, name) {
			index = i
		}
	}
	if index < 0 {
		return os.ErrNotExist
	}
	if lists[index].Name == newName {
		return nil
	}
	lists[index] = Playlist{Name: newName, Entries: append([]QueueEntry(nil), lists[index].Entries...)}
	next := s.copyData()
	next.Accounts[account] = lists
	if err = s.save(next); err != nil {
		return err
	}
	s.data = next
	return nil
}
func (s *PlaylistStore) Remove(account, name string) error {
	return s.remove(account, name, "")
}
func (s *PlaylistStore) remove(account, name, expected string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	lists := append([]Playlist(nil), s.data.Accounts[account]...)
	if !matchesPlaylistRevision(lists, name, expected) {
		return ErrPlaylistChanged
	}
	out := lists[:0]
	found := false
	for _, list := range lists {
		if strings.EqualFold(list.Name, name) {
			found = true
		} else {
			out = append(out, list)
		}
	}
	if !found {
		return os.ErrNotExist
	}
	next := s.copyData()
	next.Accounts[account] = out
	if err := s.save(next); err != nil {
		return err
	}
	s.data = next
	return nil
}
func (s *PlaylistStore) DeleteAccount(account string) error {
	if account == "" {
		return errors.New("invalid account")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data.Accounts[account]; !exists {
		return nil
	}
	next := s.copyData()
	delete(next.Accounts, account)
	if err := s.save(next); err != nil {
		return err
	}
	s.data = next
	return nil
}
func (s *PlaylistStore) copyData() playlistData {
	next := playlistData{Version: s.data.Version, Accounts: make(map[string][]Playlist, len(s.data.Accounts))}
	for account, lists := range s.data.Accounts {
		next.Accounts[account] = lists
	}
	return next
}
func (s *PlaylistStore) save(data playlistData) error {
	return saveAudioState(s.path, ".audio-playlists-*", maximumAudioStateBytes, data)
}
