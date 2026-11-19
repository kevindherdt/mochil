package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const MaximumBytes = 1 << 20

var ErrUnavailable = errors.New("managed settings require a config file")

type ApplyResult struct {
	Saved            bool     `json:"saved"`
	Applied          bool     `json:"applied"`
	Libraries        int      `json:"libraries"`
	LibrariesScanned int      `json:"libraries_scanned,omitempty"`
	LibrariesReused  int      `json:"libraries_reused,omitempty"`
	RestartRequired  []string `json:"restart_required,omitempty"`
}

type Service struct {
	ReadSource         func() ([]byte, error)
	SaveSource         func([]byte) error
	ApplySource        func([]byte) (ApplyResult, error)
	EffectiveUser      string
	PasswordConfigured bool
}

func (s Service) Available() bool {
	return s.ReadSource != nil && (s.SaveSource != nil || s.ApplySource != nil)
}

func (s Service) Read() ([]byte, error) {
	if !s.Available() {
		return nil, ErrUnavailable
	}
	data, err := s.ReadSource()
	if err != nil {
		return nil, err
	}
	document, err := decode(data)
	if err != nil {
		return nil, fmt.Errorf("read settings: %w", err)
	}
	if sync, ok := document["kosync"].(map[string]any); ok {
		if s.EffectiveUser != "" {
			sync["effective_username"] = s.EffectiveUser
		}
		if password, _ := sync["password"].(string); password != "" || s.PasswordConfigured {
			sync["password_configured"] = true
		}
		delete(sync, "password")
	}
	redactProviderSecrets(document)
	redactHermesKey(document)
	// Executable manga sources were retired. Do not expose legacy connection
	// details through the browser; a later settings save removes the stale block.
	delete(document, "external_sources")
	return json.Marshal(document)
}

func (s Service) Update(data []byte) (ApplyResult, error) {
	if !s.Available() {
		return ApplyResult{}, ErrUnavailable
	}
	if len(data) > MaximumBytes {
		return ApplyResult{}, errors.New("settings exceed 1 MiB")
	}
	incoming, err := decode(data)
	if err != nil {
		return ApplyResult{}, errors.New("invalid settings JSON")
	}
	if sync, ok := incoming["kosync"].(map[string]any); ok {
		delete(sync, "password_configured")
		delete(sync, "effective_username")
		if password, _ := sync["password"].(string); password == "" {
			if currentData, readErr := s.ReadSource(); readErr == nil {
				if current, decodeErr := decode(currentData); decodeErr == nil {
					if currentSync, ok := current["kosync"].(map[string]any); ok {
						if saved, _ := currentSync["password"].(string); saved != "" {
							sync["password"] = saved
						}
					}
				}
			}
		}
	}
	preserveProviderSecrets(incoming, s.ReadSource)
	preserveHermesKey(incoming, s.ReadSource)
	delete(incoming, "external_sources")
	normalized, err := json.MarshalIndent(incoming, "", "  ")
	if err != nil {
		return ApplyResult{}, err
	}
	if s.ApplySource != nil {
		return s.ApplySource(normalized)
	}
	if err := s.SaveSource(normalized); err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Saved: true, RestartRequired: []string{"application"}}, nil
}

func redactProviderSecrets(document map[string]any) {
	providers, ok := document["metadata_providers"].(map[string]any)
	if !ok {
		return
	}
	for _, field := range []string{"hardcover_token", "tmdb_token"} {
		if value, _ := providers[field].(string); value != "" {
			providers[field+"_configured"] = true
		}
		delete(providers, field)
	}
}

func preserveProviderSecrets(incoming map[string]any, readSource func() ([]byte, error)) {
	providers, ok := incoming["metadata_providers"].(map[string]any)
	if !ok {
		return
	}
	for _, field := range []string{"hardcover_token", "tmdb_token"} {
		delete(providers, field+"_configured")
	}
	current := map[string]any{}
	if readSource != nil {
		if data, err := readSource(); err == nil {
			current, _ = decode(data)
		}
	}
	saved, _ := current["metadata_providers"].(map[string]any)
	for _, field := range []string{"hardcover_token", "tmdb_token"} {
		if value, _ := providers[field].(string); strings.TrimSpace(value) == "" {
			if old, _ := saved[field].(string); old != "" {
				providers[field] = old
			} else {
				delete(providers, field)
			}
		}
	}
}

func decode(data []byte) (map[string]any, error) {
	if len(data) > MaximumBytes {
		return nil, errors.New("settings exceed 1 MiB")
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil || document == nil {
		if err == nil {
			err = errors.New("JSON object required")
		}
		return nil, err
	}
	return document, nil
}
