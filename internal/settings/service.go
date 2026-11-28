package settings

import (
	"crypto/sha256"
	"encoding/hex"
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
	// EffectivePublicURL is the address this server is actually reached at, which is what a device is
	// handed when it pairs, downloads a plugin or asks for a sync URL. It can come from the environment
	// and therefore differ from the copy in the config file; reporting the file's copy made the settings
	// page show an address the server was not using.
	EffectivePublicURL string
	// PublicURLFromEnvironment marks the address as owned by the environment. The page says so, because
	// editing public_url in the file would otherwise look like it did something and change nothing.
	PublicURLFromEnvironment bool
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
	// The narration token is a Hugging Face credential for the gated voice weights. It is only ever
	// read by the server, and the settings page has no field for it, so it must not travel to a browser.
	redactNarrationToken(document)
	// Report the address devices are actually handed, not the file's copy: with public_url in the
	// environment the two can differ, and the page is where someone goes to find out what their
	// server is called.
	if effective := strings.TrimRight(strings.TrimSpace(s.EffectivePublicURL), "/"); effective != "" {
		document["public_url"] = effective
		if s.PublicURLFromEnvironment {
			document["public_url_source"] = "environment"
		}
	}
	// Executable manga sources were retired. Do not expose legacy connection
	// details through the browser; a later settings save removes the stale block.
	delete(document, "external_sources")
	return json.Marshal(document)
}

// ErrStale means the settings file moved while a save was in progress, so applying this save would erase whatever the other
// writer just stored. The page has to reload before saving again.
var ErrStale = errors.New("the settings changed elsewhere while this save was in progress; reload and try again")

// sourceFingerprint identifies the settings file a save was built against. Two saves that overlap both read the file to
// preserve the secrets it holds; the later one must notice that the file moved, or it silently writes the earlier save away.
func sourceFingerprint(read func() ([]byte, error)) string {
	data, err := read()
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

func (s Service) Update(data []byte) (ApplyResult, error) {
	if !s.Available() {
		return ApplyResult{}, ErrUnavailable
	}
	if len(data) > MaximumBytes {
		return ApplyResult{}, errors.New("settings exceed 1 MiB")
	}
	before := sourceFingerprint(s.ReadSource)
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
	// Read() adds fields the page displays; they are not settings, and the config decoder is strict,
	// so a page that saves back what it was shown must not write them into the file.
	delete(incoming, "public_url_source")
	delete(incoming, "narration_token_configured")
	preserveNarrationToken(incoming, s.ReadSource)
	// While the environment owns the address, the page is showing that value rather than the file's.
	// Writing it back would quietly rewrite the file to match a variable that is already in charge, and
	// the file would then claim an address nobody chose.
	if s.PublicURLFromEnvironment {
		s.restoreStoredPublicURL(incoming)
	}
	// The preserve* helpers above read the file to keep the KoSync password, provider tokens and narration token. If the
	// file moved since, this save was built on a snapshot that no longer exists and writing it would drop the other save.
	if before != "" && sourceFingerprint(s.ReadSource) != before {
		return ApplyResult{}, ErrStale
	}
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

// restoreStoredPublicURL puts back whatever the file itself says, so a save made while the environment
// supplies the address neither invents a value nor drops one the file had.
func (s Service) restoreStoredPublicURL(incoming map[string]any) {
	data, err := s.ReadSource()
	if err != nil {
		delete(incoming, "public_url")
		return
	}
	document, err := decode(data)
	if err != nil {
		delete(incoming, "public_url")
		return
	}
	if stored, ok := document["public_url"].(string); ok && strings.TrimSpace(stored) != "" {
		incoming["public_url"] = stored
		return
	}
	delete(incoming, "public_url")
}

func redactNarrationToken(document map[string]any) {
	narration, ok := document["narration"].(map[string]any)
	if !ok {
		return
	}
	if value, _ := narration["token"].(string); value != "" {
		narration["narration_token_configured"] = true
	}
	delete(narration, "token")
}

// preserveNarrationToken keeps the saved token when the page saves everything else, the way the KoSync
// password and the provider secrets are kept.
func preserveNarrationToken(incoming map[string]any, readSource func() ([]byte, error)) {
	narration, ok := incoming["narration"].(map[string]any)
	if !ok {
		return
	}
	delete(narration, "narration_token_configured")
	if token, _ := narration["token"].(string); token != "" {
		return
	}
	data, err := readSource()
	if err != nil {
		return
	}
	current, err := decode(data)
	if err != nil {
		return
	}
	saved, ok := current["narration"].(map[string]any)
	if !ok {
		return
	}
	if token, _ := saved["token"].(string); token != "" {
		narration["token"] = token
	}
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
