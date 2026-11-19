package appconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var libraryID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

type Config struct {
	Listen         string     `json:"listen"`
	PublicURL      string     `json:"public_url"`
	ScanEvery      string     `json:"scan_every"`
	CacheDirectory string     `json:"cache_directory"`
	Libraries      []Library  `json:"libraries"`
	PageCache      PageCache  `json:"page_cache"`
	CoverCache     CoverCache `json:"cover_cache"`
	KoSync         KoSync     `json:"kosync"`
	Playback       Playback   `json:"playback"`
	// Appearance holds the few look settings that belong to the server, retired: the sign-in background is no longer a setting.
	Appearance Appearance `json:"appearance,omitempty"`
	// RetiredExternalSources exists only so strict decoding accepts old files.
	// Decode clears it immediately; legacy credentials must never survive in the
	// active runtime configuration.
	RetiredExternalSources json.RawMessage   `json:"external_sources,omitempty"`
	MetadataProviders      MetadataProviders `json:"metadata_providers,omitempty"`
	ManagedIngest          ManagedIngest     `json:"managed_ingest,omitempty"`
	// Modules overrides which optional modules run. Absent means automatic:
	// a module runs when a library of its kind exists.
	Modules map[string]bool `json:"modules,omitempty"`
	// Narration configures the optional Narration module: a local Pocket TTS
	// runtime that reads books aloud. Nothing starts until a book is narrated.
	Narration Narration `json:"narration,omitempty"`
	// Hermes points the optional Hermes module at a Hermes Agent API server.
	Hermes Hermes `json:"hermes,omitempty"`
	// Avatar says where the optional avatar module finds Live2D model folders.
	Avatar Avatar `json:"avatar,omitempty"`
}

// Avatar is the folder that holds one sub-folder per Live2D model (default /avatars).
type Avatar struct {
	Directory string `json:"directory,omitempty"`
}

// Hermes is the address and key of a Hermes Agent API server (API_SERVER_KEY).
// The key stays on the server; browsers and apps only ever talk to Mochil.
type Hermes struct {
	URL   string `json:"url,omitempty"`
	Key   string `json:"key,omitempty"`
	Model string `json:"model,omitempty"`
	// Name is what the assistant is called in Mochil; Persona is standing guidance sent with every message.
	Name    string `json:"name,omitempty"`
	Persona string `json:"persona,omitempty"`
	// MemoryURL and MemoryBank point at the Hindsight server and the memory bank Hermes keeps for this assistant, so
	// the chat can show what she remembers and save a message on request. Both are optional; see docs/HERMES_PLAN.md.
	MemoryURL  string `json:"memory_url,omitempty"`
	MemoryBank string `json:"memory_bank,omitempty"`
	// NoLibraryPlay turns off the "play a song from my library" ability. It is on unless this is set, so a saved
	// configuration without the field keeps working.
	NoLibraryPlay bool `json:"no_library_play,omitempty"`
	// Features switches single abilities of the chat on or off (see HermesFeatures). A feature that is not listed is on.
	Features map[string]bool `json:"features,omitempty"`
	// InventoryURL is the read-only list helper (tools/hermes-inventory). Blank means the Hermes host on port 8643.
	InventoryURL string `json:"inventory_url,omitempty"`
	// HelperURL and HelperModel name a plain language model (an OpenAI-style address such as an Ollama or LM Studio
	// server) that the one-shot helpers (Polish a note, Ask Hermes about ISBNs) call directly. They never go through the
	// Hermes agent: its tools can wander (a small model once looped through Home Assistant calls for a bare "ping").
	HelperURL   string `json:"helper_url,omitempty"`
	HelperModel string `json:"helper_model,omitempty"`
}

// HelperConfigured says whether the one-shot helpers have a model to call.
func (h Hermes) HelperConfigured() bool {
	return strings.TrimSpace(h.HelperURL) != "" && strings.TrimSpace(h.HelperModel) != ""
}

// HermesFeatures are the abilities inside the Hermes module that can be switched off one by one.
var HermesFeatures = []string{"library_play", "home", "radio", "voice", "quick", "memory", "polish", "isbn", "match", "music"}

// Feature reports whether one ability of the chat is on.
func (h Hermes) Feature(name string) bool {
	if on, listed := h.Features[name]; listed {
		return on
	}
	return !(name == "library_play" && h.NoLibraryPlay)
}

// Narration points at an installed runtime; see docs/NARRATION.md.
type Narration struct {
	Python      string `json:"python,omitempty"`
	Cache       string `json:"cache,omitempty"`
	Threads     int    `json:"threads,omitempty"`
	Offline     bool   `json:"offline,omitempty"`
	IdleSeconds int    `json:"idle_seconds,omitempty"`
	// Token is an optional Hugging Face token for the gated voice-cloning weights.
	Token string `json:"token,omitempty"`
	// VoiceDirectory holds saved voices; it defaults to a folder in the cache directory.
	VoiceDirectory string `json:"voice_directory,omitempty"`
}

// ManagedIngest is disabled unless both roots are configured. Incoming files
// remain source-owned; verified copies are committed only below ManagedDirectory.
type ManagedIngest struct {
	IncomingDirectory string `json:"incoming_directory,omitempty"`
	ManagedDirectory  string `json:"managed_directory,omitempty"`
	MaximumFileBytes  int64  `json:"maximum_file_bytes,omitempty"`
}

type MetadataProviders struct {
	HardcoverToken string `json:"hardcover_token,omitempty"`
	TMDBToken      string `json:"tmdb_token,omitempty"`
}

// Appearance only exists so config files written when the sign-in background was a choice still load
// (unknown fields are rejected). The value is read and discarded: there is one sign-in background.
type Appearance struct {
	LoginBackground any `json:"login_background,omitempty"`
}

type Playback struct {
	AudioLanguage    string `json:"audio_language,omitempty"`
	SubtitleLanguage string `json:"subtitle_language,omitempty"`
	WiFiQuality      string `json:"wifi_quality,omitempty"`
	MobileQuality    string `json:"mobile_quality,omitempty"`
}

type Library struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Kind             string   `json:"kind"`
	PageStreaming    bool     `json:"page_streaming"`
	MetadataWrites   bool     `json:"metadata_writes,omitempty"`
	VaultWrites      bool     `json:"vault_writes,omitempty"`
	PreferredFormats []string `json:"preferred_formats,omitempty"`
	MetadataProvider string   `json:"metadata_provider,omitempty"`
	// ScanEvery overrides the global scan interval for this library: empty uses
	// the global one, "0" means only when scanned by hand.
	ScanEvery string    `json:"scan_every,omitempty"`
	Source    Source    `json:"source"`
	Inference Inference `json:"inference"`
}

type Source struct {
	Type           string `json:"type"`
	Path           string `json:"path"`
	URL            string `json:"url"`
	Repository     string `json:"repository"`
	RepositoryName string `json:"repository_name"`
	Root           string `json:"root"`
	Manifest       string `json:"manifest"`
	TokenFile      string `json:"token_file"`
}

type Inference struct {
	FolderLayout   string `json:"folder_layout"`
	FilenameAuthor bool   `json:"filename_author"`
}

type PageCache struct {
	Directory    string `json:"directory"`
	MaximumBytes int64  `json:"maximum_bytes"`
	ExpireAfter  string `json:"expire_after"`
}

type CoverCache struct {
	MaximumBytes  int64 `json:"maximum_bytes"`
	MaximumWidth  int   `json:"maximum_width"`
	MaximumHeight int   `json:"maximum_height"`
	JPEGQuality   int   `json:"jpeg_quality"`
}

type KoSync struct {
	Enabled           bool   `json:"enabled"`
	WebReaderBridge   *bool  `json:"web_reader_bridge,omitempty"`
	Provider          string `json:"provider,omitempty"`
	ServerURL         string `json:"server_url,omitempty"`
	Username          string `json:"username,omitempty"`
	Password          string `json:"password,omitempty"`
	Database          string `json:"database"`
	AllowRegistration bool   `json:"allow_registration"`
}

func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	return Decode(f, filepath.Dir(path))
}

func Decode(r io.Reader, configDir string) (Config, error) {
	var cfg Config
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Config{}, errors.New("decode config: multiple JSON values")
		}
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	// The retired source engine is not merely disabled: discard its entire
	// legacy payload before validation or runtime assembly can observe it.
	cfg.RetiredExternalSources = nil
	cfg.Appearance = Appearance{}
	applyDefaults(&cfg, configDir)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyDefaults(cfg *Config, configDir string) {
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	if cfg.ScanEvery == "" {
		cfg.ScanEvery = "10m"
	}
	if cfg.CacheDirectory == "" {
		cfg.CacheDirectory = filepath.Join(configDir, "cache")
	}
	if cfg.PageCache.ExpireAfter == "" {
		cfg.PageCache.ExpireAfter = "168h"
	}
	if cfg.PageCache.Directory == "" {
		cfg.PageCache.Directory = filepath.Join(cfg.CacheDirectory, "publications")
	}
	if cfg.PageCache.MaximumBytes == 0 {
		cfg.PageCache.MaximumBytes = 4 << 30
	}
	if cfg.CoverCache.MaximumBytes == 0 {
		cfg.CoverCache.MaximumBytes = 512 << 20
	}
	if cfg.CoverCache.MaximumWidth == 0 {
		cfg.CoverCache.MaximumWidth = 600
	}
	if cfg.CoverCache.MaximumHeight == 0 {
		cfg.CoverCache.MaximumHeight = 900
	}
	if cfg.CoverCache.JPEGQuality == 0 {
		cfg.CoverCache.JPEGQuality = 82
	}
	if cfg.ManagedIngest.IncomingDirectory != "" || cfg.ManagedIngest.ManagedDirectory != "" {
		if !filepath.IsAbs(cfg.ManagedIngest.IncomingDirectory) {
			cfg.ManagedIngest.IncomingDirectory = filepath.Join(configDir, cfg.ManagedIngest.IncomingDirectory)
		}
		// The managed folder is not something a reader should have to think about:
		// it only ever holds Mochil's own verified copies, never anything a person
		// browses to directly, so picking the incoming folder is enough on its own.
		// It lives under CacheDirectory, which despite its name is the precious,
		// backed-up zone (see bootstrap.json: cache_directory is /data/catalogs).
		if cfg.ManagedIngest.ManagedDirectory == "" {
			cfg.ManagedIngest.ManagedDirectory = filepath.Join(cfg.CacheDirectory, "managed-ingest")
		} else if !filepath.IsAbs(cfg.ManagedIngest.ManagedDirectory) {
			cfg.ManagedIngest.ManagedDirectory = filepath.Join(configDir, cfg.ManagedIngest.ManagedDirectory)
		}
		cfg.ManagedIngest.IncomingDirectory = filepath.Clean(cfg.ManagedIngest.IncomingDirectory)
		cfg.ManagedIngest.ManagedDirectory = filepath.Clean(cfg.ManagedIngest.ManagedDirectory)
		if cfg.ManagedIngest.MaximumFileBytes == 0 {
			cfg.ManagedIngest.MaximumFileBytes = 8 << 30
		}
	}
	if cfg.Narration.VoiceDirectory == "" {
		cfg.Narration.VoiceDirectory = filepath.Join(cfg.CacheDirectory, "narration", "voices")
	}
	if cfg.KoSync.Database == "" {
		cfg.KoSync.Database = filepath.Join(cfg.CacheDirectory, "kosync.json")
	}
	if cfg.KoSync.Provider == "" {
		cfg.KoSync.Provider = "built-in"
	}
	if cfg.KoSync.WebReaderBridge == nil {
		enabled := true
		cfg.KoSync.WebReaderBridge = &enabled
	}
	if cfg.Playback.WiFiQuality == "" {
		cfg.Playback.WiFiQuality = "maximum"
	}
	if cfg.Playback.MobileQuality == "" {
		cfg.Playback.MobileQuality = "high"
	}
	for i := range cfg.Libraries {
		library := &cfg.Libraries[i]
		if library.Name == "" {
			library.Name = library.ID
		}
		if library.Kind == "" {
			library.Kind = "books"
		}
		if library.Source.Type == "" {
			library.Source.Type = "filesystem"
		}
		if library.Inference.FolderLayout == "" {
			library.Inference.FolderLayout = "auto"
		}
		if library.PreferredFormats == nil {
			switch library.Kind {
			case "books", "mixed":
				library.PreferredFormats = []string{"epub", "pdf", "mobi", "azw3"}
			case "comics", "manga":
				library.PreferredFormats = []string{"cbz", "pdf", "epub"}
			}
		}
	}
	applyMetadataProviders(cfg)
}

func (cfg Config) ScanDuration() (time.Duration, error) {
	d, err := time.ParseDuration(cfg.ScanEvery)
	if err != nil {
		return 0, fmt.Errorf("invalid scan_every %q: %w", cfg.ScanEvery, err)
	}
	if d < 0 {
		return 0, errors.New("scan_every cannot be negative")
	}
	return d, nil
}

// ScanDuration is how often this library is rescanned; fallback is the global interval.
func (library Library) ScanDuration(fallback time.Duration) (time.Duration, error) {
	if strings.TrimSpace(library.ScanEvery) == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(strings.TrimSpace(library.ScanEvery))
	if err != nil {
		return 0, fmt.Errorf("invalid scan_every %q: %w", library.ScanEvery, err)
	}
	if d < 0 || (d > 0 && d < time.Minute) {
		return 0, errors.New("scan_every must be 0 or at least 1m")
	}
	return d, nil
}

func (cfg Config) Validate() error {
	if err := cfg.validateModules(); err != nil {
		return err
	}
	if _, err := cfg.ScanDuration(); err != nil {
		return err
	}
	if cfg.PageCache.ExpireAfter != "" {
		if _, err := time.ParseDuration(cfg.PageCache.ExpireAfter); err != nil {
			return fmt.Errorf("invalid page_cache.expire_after: %w", err)
		}
	}
	if cfg.PageCache.MaximumBytes != 0 && cfg.PageCache.MaximumBytes < 64<<20 {
		return errors.New("page_cache.maximum_bytes must be at least 67108864")
	}
	if cfg.CoverCache.MaximumBytes != 0 && cfg.CoverCache.MaximumBytes < 16<<20 {
		return errors.New("cover_cache.maximum_bytes must be at least 16777216")
	}
	if cfg.CoverCache.MaximumWidth != 0 && (cfg.CoverCache.MaximumWidth < 128 || cfg.CoverCache.MaximumWidth > 4096) {
		return errors.New("cover_cache.maximum_width must be between 128 and 4096")
	}
	if cfg.CoverCache.MaximumHeight != 0 && (cfg.CoverCache.MaximumHeight < 128 || cfg.CoverCache.MaximumHeight > 4096) {
		return errors.New("cover_cache.maximum_height must be between 128 and 4096")
	}
	if cfg.CoverCache.JPEGQuality != 0 && (cfg.CoverCache.JPEGQuality < 40 || cfg.CoverCache.JPEGQuality > 95) {
		return errors.New("cover_cache.jpeg_quality must be between 40 and 95")
	}
	if strings.ContainsRune(cfg.ManagedIngest.IncomingDirectory, '\x00') || strings.ContainsRune(cfg.ManagedIngest.ManagedDirectory, '\x00') {
		return errors.New("managed_ingest directories contain an invalid character")
	}
	if (cfg.ManagedIngest.IncomingDirectory == "") != (cfg.ManagedIngest.ManagedDirectory == "") {
		return errors.New("managed_ingest requires both incoming_directory and managed_directory")
	}
	if cfg.ManagedIngest.IncomingDirectory == "" && cfg.ManagedIngest.MaximumFileBytes != 0 {
		return errors.New("managed_ingest.maximum_file_bytes requires configured directories")
	}
	if cfg.ManagedIngest.MaximumFileBytes != 0 && (cfg.ManagedIngest.MaximumFileBytes < 1<<20 || cfg.ManagedIngest.MaximumFileBytes > 1<<40) {
		return errors.New("managed_ingest.maximum_file_bytes must be between 1048576 and 1099511627776")
	}
	if cfg.ManagedIngest.IncomingDirectory != "" && pathsOverlap(cfg.ManagedIngest.IncomingDirectory, cfg.ManagedIngest.ManagedDirectory) {
		return errors.New("managed_ingest incoming and managed directories must not overlap")
	}
	if cfg.KoSync.Provider != "" && cfg.KoSync.Provider != "built-in" && cfg.KoSync.Provider != "external" && cfg.KoSync.Provider != "disabled" {
		return errors.New("kosync.provider must be built-in, external, or disabled")
	}
	if len(cfg.MetadataProviders.HardcoverToken) > 4096 || len(cfg.MetadataProviders.TMDBToken) > 4096 {
		return errors.New("metadata provider credentials must not exceed 4096 characters")
	}
	if !validPlaybackQuality(cfg.Playback.WiFiQuality) || !validPlaybackQuality(cfg.Playback.MobileQuality) {
		return errors.New("playback quality must be maximum, high, medium, or low")
	}
	if cfg.KoSync.Provider == "external" {
		server, err := url.Parse(cfg.KoSync.ServerURL)
		if err != nil || (server.Scheme != "http" && server.Scheme != "https") || server.Host == "" || server.RawQuery != "" || server.Fragment != "" {
			return errors.New("kosync.server_url must be an absolute HTTP or HTTPS URL without a query or fragment")
		}
		if (cfg.KoSync.Username == "") != (cfg.KoSync.Password == "") {
			return errors.New("kosync.username and kosync.password must be configured together")
		}
	}
	seen := make(map[string]bool, len(cfg.Libraries))
	for i, library := range cfg.Libraries {
		prefix := fmt.Sprintf("libraries[%d]", i)
		switch library.MetadataProvider {
		case "":
		case "openlibrary":
			if !isPublicationKind(library.Kind) {
				return fmt.Errorf("%s Open Library metadata requires a publication library", prefix)
			}
		case "hardcover":
			if !isPublicationKind(library.Kind) {
				return fmt.Errorf("%s Hardcover metadata requires a publication library", prefix)
			}
			if strings.TrimSpace(cfg.MetadataProviders.HardcoverToken) == "" {
				return fmt.Errorf("%s Hardcover metadata requires metadata_providers.hardcover_token", prefix)
			}
		case "tmdb":
			if library.Kind != "movies" && library.Kind != "tv" && library.Kind != "anime" {
				return fmt.Errorf("%s TMDB metadata requires a video library", prefix)
			}
			if strings.TrimSpace(cfg.MetadataProviders.TMDBToken) == "" {
				return fmt.Errorf("%s TMDB metadata requires metadata_providers.tmdb_token", prefix)
			}
		default:
			return fmt.Errorf("%s.metadata_provider is unsupported", prefix)
		}
		if _, err := library.ScanDuration(0); err != nil {
			return fmt.Errorf("%s.%w", prefix, err)
		}
		seenFormats := map[string]bool{}
		for _, format := range library.PreferredFormats {
			format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(format, ".")))
			switch format {
			case "epub", "pdf", "cbz", "mobi", "azw3":
			default:
				return fmt.Errorf("%s.preferred_formats contains unsupported format %q", prefix, format)
			}
			if seenFormats[format] {
				return fmt.Errorf("%s.preferred_formats contains duplicate format %q", prefix, format)
			}
			seenFormats[format] = true
		}
		if library.MetadataWrites && library.Source.Type != "filesystem" && library.Source.Type != "calibre" {
			return fmt.Errorf("%s.metadata_writes is supported only for filesystem and Calibre libraries", prefix)
		}
		if library.MetadataWrites && (library.Kind == "movies" || library.Kind == "tv" || library.Kind == "anime" || library.Kind == "audiobooks" || library.Kind == "podcasts" || library.Kind == "music" || library.Kind == "documents" || library.Kind == "vault") {
			return fmt.Errorf("%s.metadata_writes is supported only for publication libraries", prefix)
		}
		if library.VaultWrites && (library.Kind != "vault" || library.Source.Type != "filesystem") {
			return fmt.Errorf("%s.vault_writes is supported only for local Obsidian vault libraries", prefix)
		}
		if !libraryID.MatchString(library.ID) {
			return fmt.Errorf("%s.id must match %s", prefix, libraryID.String())
		}
		if seen[library.ID] {
			return fmt.Errorf("duplicate library id %q", library.ID)
		}
		seen[library.ID] = true
		switch library.Kind {
		case "books", "comics", "manga", "mixed", "movies", "tv", "anime", "audiobooks", "podcasts", "music", "documents", "vault":
		default:
			return fmt.Errorf("%s.kind must be books, comics, manga, mixed, movies, tv, anime, audiobooks, podcasts, music, documents, or vault", prefix)
		}
		if (library.Kind == "documents" || library.Kind == "vault") && library.Source.Type != "filesystem" {
			return fmt.Errorf("%s document and vault libraries require a filesystem source", prefix)
		}
		switch library.Inference.FolderLayout {
		case "auto", "series", "metadata-only":
		default:
			return fmt.Errorf("%s.inference.folder_layout must be auto, series, or metadata-only", prefix)
		}
		switch library.Source.Type {
		case "filesystem", "calibre":
			if strings.TrimSpace(library.Source.Path) == "" {
				return fmt.Errorf("%s.source.path is required", prefix)
			}
		default:
			return fmt.Errorf("%s.source.type must be filesystem or calibre", prefix)
		}
	}
	return nil
}

func validPlaybackQuality(value string) bool {
	switch value {
	case "", "maximum", "high", "medium", "low":
		return true
	default:
		return false
	}
}

func isPublicationKind(kind string) bool {
	switch kind {
	case "books", "comics", "manga", "mixed":
		return true
	default:
		return false
	}
}

func pathsOverlap(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	leftToRight, leftErr := filepath.Rel(left, right)
	rightToLeft, rightErr := filepath.Rel(right, left)
	inside := func(relative string, err error) bool {
		return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
	}
	return inside(leftToRight, leftErr) || inside(rightToLeft, rightErr)
}
