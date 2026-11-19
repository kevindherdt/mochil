package video

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mochil/internal/statefile"
)

const maximumPreferenceStateBytes int64 = 4 << 20

type LanguagePreferences struct {
	AudioLanguage    string `json:"audio_language,omitempty"`
	SubtitleLanguage string `json:"subtitle_language,omitempty"`
}

type PreferenceStore struct {
	mu       sync.RWMutex
	path     string
	defaults LanguagePreferences
	shows    map[string]LanguagePreferences
}

func OpenPreferenceStore(path string, defaults LanguagePreferences) (*PreferenceStore, error) {
	s := &PreferenceStore{path: path, defaults: normalizePreferences(defaults), shows: map[string]LanguagePreferences{}}
	if path == "" {
		return s, nil
	}
	err := statefile.DecodeJSON(path, maximumPreferenceStateBytes, &s.shows)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if s.shows == nil {
		s.shows = map[string]LanguagePreferences{}
	}
	return s, nil
}

func preferenceKey(library, show string) string { return library + "\x00" + strings.TrimSpace(show) }

func normalizePreferences(p LanguagePreferences) LanguagePreferences {
	p.AudioLanguage = strings.ToLower(strings.TrimSpace(p.AudioLanguage))
	p.SubtitleLanguage = strings.ToLower(strings.TrimSpace(p.SubtitleLanguage))
	return p
}

func (s *PreferenceStore) Effective(library, show string) (LanguagePreferences, LanguagePreferences) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	override := s.shows[preferenceKey(library, show)]
	effective := s.defaults
	if override.AudioLanguage != "" {
		effective.AudioLanguage = override.AudioLanguage
	}
	if override.SubtitleLanguage != "" {
		effective.SubtitleLanguage = override.SubtitleLanguage
	}
	return effective, override
}

func (s *PreferenceStore) Set(library, show string, value LanguagePreferences) error {
	value = normalizePreferences(value)
	s.mu.Lock()
	defer s.mu.Unlock()
	key := preferenceKey(library, show)
	previous, existed := s.shows[key]
	if value.AudioLanguage == "" && value.SubtitleLanguage == "" {
		delete(s.shows, key)
	} else {
		s.shows[key] = value
	}
	if err := s.saveLocked(); err != nil {
		if existed {
			s.shows[key] = previous
		} else {
			delete(s.shows, key)
		}
		return err
	}
	return nil
}

func (s *PreferenceStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.shows, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".video-preferences-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, s.path)
}
