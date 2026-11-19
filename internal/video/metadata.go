package video

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"mochil/internal/enrichment"
	"mochil/internal/statefile"
)

const maximumMetadataStateBytes int64 = 16 << 20

type MetadataOverride struct {
	Title       string `json:"title,omitempty"`
	Year        int    `json:"year,omitempty"`
	Overview    string `json:"overview,omitempty"`
	ArtworkURL  string `json:"artwork_url,omitempty"`
	BackdropURL string `json:"backdrop_url,omitempty"`
	ProviderID  string `json:"provider_id,omitempty"`
	Provider    string `json:"provider,omitempty"`
}

type MetadataStore struct {
	mu   sync.RWMutex
	path string
	data map[string]MetadataOverride
}

func OpenMetadataStore(path string) (*MetadataStore, error) {
	s := &MetadataStore{path: path, data: map[string]MetadataOverride{}}
	if path == "" {
		return s, nil
	}
	err := statefile.DecodeJSON(path, maximumMetadataStateBytes, &s.data)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("decode video metadata overrides: %w", err)
	}
	if s.data == nil {
		s.data = map[string]MetadataOverride{}
	}
	return s, nil
}

func metadataKey(libraryID, itemID string) string { return libraryID + "\x00" + itemID }
func showMetadataKey(libraryID, show string) string {
	return libraryID + "\x00show\x00" + strings.ToLower(strings.TrimSpace(show))
}

func (s *MetadataStore) Get(libraryID, itemID string) (MetadataOverride, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[metadataKey(libraryID, itemID)]
	return value, ok
}

func (s *MetadataStore) GetShow(libraryID, show string) (MetadataOverride, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[showMetadataKey(libraryID, show)]
	return value, ok
}

func (s *MetadataStore) Set(libraryID, itemID string, value MetadataOverride) error {
	return s.set(metadataKey(libraryID, itemID), libraryID, itemID, value)
}

func (s *MetadataStore) SetShow(libraryID, show string, value MetadataOverride) error {
	return s.set(showMetadataKey(libraryID, show), libraryID, show, value)
}

func (s *MetadataStore) Delete(libraryID, itemID string) error {
	return s.delete(metadataKey(libraryID, itemID))
}

func (s *MetadataStore) DeleteShow(libraryID, show string) error {
	return s.delete(showMetadataKey(libraryID, show))
}

func (s *MetadataStore) delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, exists := s.data[key]
	if !exists {
		return nil
	}
	delete(s.data, key)
	if err := s.saveLocked(); err != nil {
		s.data[key] = previous
		return err
	}
	return nil
}

func (s *MetadataStore) set(key, libraryID, identity string, value MetadataOverride) error {
	if strings.TrimSpace(libraryID) == "" || strings.TrimSpace(identity) == "" {
		return errors.New("video metadata requires a library and item")
	}
	value.Title = strings.TrimSpace(value.Title)
	value.Overview = strings.TrimSpace(value.Overview)
	value.ProviderID = strings.TrimSpace(value.ProviderID)
	value.Provider = strings.TrimSpace(value.Provider)
	if len(value.Title) > 500 || len(value.Overview) > 4000 || len(value.ProviderID) > 64 || value.Year < 0 || value.Year > 3000 {
		return errors.New("video metadata override is outside its allowed bounds")
	}
	if value.Provider != "TMDB" || value.ProviderID == "" {
		return errors.New("video metadata override requires TMDB provenance")
	}
	if _, err := strconv.ParseInt(value.ProviderID, 10, 64); err != nil {
		return errors.New("invalid TMDB identifier")
	}
	if value.ArtworkURL != "" && !enrichment.ValidTMDBArtworkURL(value.ArtworkURL) {
		return errors.New("invalid TMDB artwork URL")
	}
	if value.BackdropURL != "" && !enrichment.ValidTMDBBackdropURL(value.BackdropURL) {
		return errors.New("invalid TMDB backdrop URL")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, existed := s.data[key]
	s.data[key] = value
	if err := s.saveLocked(); err != nil {
		if existed {
			s.data[key] = previous
		} else {
			delete(s.data, key)
		}
		return err
	}
	return nil
}

func (s *MetadataStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".video-metadata-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if err := statefile.EncodeJSON(temporary, maximumMetadataStateBytes, s.data, true); err != nil {
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
