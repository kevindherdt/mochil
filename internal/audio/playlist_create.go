package audio

import (
	"errors"
	"strings"
)

var ErrPlaylistExists = errors.New("a playlist with this name already exists")

// Create never replaces an existing playlist, including concurrent/retried creates.
func (s *PlaylistStore) Create(account, name string) error {
	name, err := cleanPlaylistName(name)
	if err != nil {
		return err
	}
	if account == "" {
		return errors.New("invalid playlist account")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lists := s.data.Accounts[account]
	for _, list := range lists {
		if strings.EqualFold(list.Name, name) {
			return ErrPlaylistExists
		}
	}
	if len(lists) >= 1000 {
		return errors.New("playlist limit reached")
	}
	next := s.copyData()
	next.Accounts[account] = append(append([]Playlist(nil), lists...), Playlist{Name: name})
	if err := s.save(next); err != nil {
		return err
	}
	s.data = next
	return nil
}
