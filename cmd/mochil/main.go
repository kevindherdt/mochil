package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"mochil/internal/version"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"mochil/internal/appconfig"
	"mochil/internal/audio"
	"mochil/internal/catalog"
	"mochil/internal/document"
	"mochil/internal/jobs"
	"mochil/internal/vaultwriter"
	"mochil/internal/video"
	"mochil/internal/web"
)

type runtimeConfig struct {
	modules                map[string]bool
	narration              appconfig.Narration
	hermes                 appconfig.Hermes
	avatar                 appconfig.Avatar
	listen                 string
	publicURL              string
	username               string
	password               string
	scanEvery              time.Duration
	libraryScanEvery       map[string]time.Duration
	cacheDirectory         string
	coverCacheMaximumBytes int64
	coverMaximumWidth      int
	coverMaximumHeight     int
	coverJPEGQuality       int
	publicationCache       string
	publicationCacheBytes  int64
	publicationCacheExpiry time.Duration
	kosyncEnabled          bool
	kosyncWebReaderBridge  bool
	kosyncDatabase         string
	kosyncRegistration     bool
	kosyncProvider         string
	kosyncServerURL        string
	kosyncUsername         string
	kosyncPassword         string
	videoAudioLanguage     string
	videoSubtitleLanguage  string
	tmdbToken              string
	hardcoverToken         string
	managedIngest          appconfig.ManagedIngest
	libraries              []web.Library
	libraryFingerprints    map[string]string
}

func main() {
	// Cover decoding and scans make short memory spikes; a soft limit makes the
	// runtime return them instead of holding the peak. GOMEMLIMIT still wins.
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(768 << 20)
	}
	// The live heap is small; hand freed spike memory back promptly.
	go func() {
		for range time.Tick(2 * time.Minute) {
			debug.FreeOSMemory()
		}
	}()
	var configPath, configTemplate string
	var legacy web.Config
	var legacyScanEvery time.Duration
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.StringVar(&configPath, "config", env("MOCHIL_CONFIG", ""), "JSON configuration file for one or more libraries")
	flag.StringVar(&configTemplate, "config-template", "", "copy this configuration on first run when -config does not exist")
	flag.StringVar(&legacy.Listen, "listen", env("MOCHIL_LISTEN", ":8080"), "HTTP listen address")
	flag.StringVar(&legacy.Library, "library", env("MOCHIL_LIBRARY", "./books"), "legacy single-library directory")
	flag.StringVar(&legacy.Cache, "cache", env("MOCHIL_CACHE", "./mochil-cache.json"), "legacy single-library metadata cache")
	flag.StringVar(&legacy.PublicURL, "public-url", env("MOCHIL_PUBLIC_URL", ""), "external base URL, such as https://books.example.com")
	flag.StringVar(&legacy.Username, "username", env("MOCHIL_USERNAME", ""), "optional HTTP Basic username")
	flag.StringVar(&legacy.Password, "password", env("MOCHIL_PASSWORD", ""), "optional HTTP Basic password")
	flag.DurationVar(&legacyScanEvery, "scan-every", envDuration("MOCHIL_SCAN_EVERY", 10*time.Minute), "filesystem rescan interval (0 disables)")
	flag.Parse()
	if *showVersion {
		fmt.Println("mochil " + version.String())
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := bootstrapConfig(configPath, configTemplate); err != nil {
		logger.Error("cannot initialize configuration", "error", err)
		os.Exit(2)
	}
	runtime, err := configure(configPath, legacy, legacyScanEvery)
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(2)
	}

	successful := 0
	pendingAudio := make([]web.Library, 0)
	for _, library := range runtime.libraries {
		if library.Audio != nil {
			pendingAudio = append(pendingAudio, library)
			continue
		}
		if scanLibrary(logger, library) {
			successful++
		}
	}
	if len(runtime.libraries) > 0 && successful == 0 && len(pendingAudio) == 0 {
		logger.Error("no library could be scanned; serving the last known state so Settings remains available")
	}
	if len(runtime.libraries) == 0 {
		logger.Warn("no libraries configured; open Settings to add the first library")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	application := &liveApplication{ctx: ctx, logger: logger, configPath: configPath, legacy: legacy, legacyScanEvery: legacyScanEvery, current: runtime, scanJobs: jobs.New(2, 64)}
	handler, err := application.build(runtime)
	if err != nil {
		logger.Error("invalid server configuration", "error", err)
		os.Exit(2)
	}
	application.handler.Store(handler)
	for _, library := range pendingAudio {
		application.scheduleLibraryScan(library, runtime.libraryFingerprints[library.ID])
	}
	application.resetScans(runtime)

	srv := &http.Server{
		Addr:              runtime.listen,
		Handler:           application,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	logger.Info("serving", "address", runtime.listen, "libraries", len(runtime.libraries))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
	// ListenAndServe returns the moment shutdown begins; wait for requests in flight (a saved reading position, an
	// upload) to finish before the stores they write to are closed and the process exits.
	<-drained
	application.close()
}

func bootstrapConfig(path, template string) error {
	if path == "" || template == "" {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := os.ReadFile(template)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return writeFileDurable(path, data, 0o600)
}

// writeFileDurable writes a file through a temporary in the same directory and
// flushes it before publishing, so a crash cannot leave a truncated file.
func writeFileDurable(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func saveConfig(path string, data []byte) error {
	if path == "" {
		return fmt.Errorf("managed settings require -config")
	}
	if _, err := appconfig.Decode(bytes.NewReader(data), filepath.Dir(path)); err != nil {
		return err
	}
	temporary := path + ".tmp"
	// The config is the only copy of the libraries, tokens and sync credentials: an
	// unflushed rename could leave it empty after a crash.
	if err := writeFileDurable(temporary, append(data, '\n'), 0o600); err != nil {
		return err
	}
	current, readErr := os.ReadFile(path)
	if readErr == nil {
		backupTemporary := path + ".previous.tmp"
		if err := writeFileDurable(backupTemporary, current, 0o600); err != nil {
			_ = os.Remove(temporary)
			return fmt.Errorf("back up current configuration: %w", err)
		}
		if err := os.Rename(backupTemporary, path+".previous"); err != nil {
			_ = os.Remove(temporary)
			return fmt.Errorf("publish configuration backup: %w", err)
		}
	} else if !os.IsNotExist(readErr) {
		_ = os.Remove(temporary)
		return fmt.Errorf("read current configuration: %w", readErr)
	}
	return os.Rename(temporary, path)
}

func configure(configPath string, legacy web.Config, legacyScanEvery time.Duration) (runtimeConfig, error) {
	if configPath == "" {
		store := catalog.NewStore(legacy.Library, legacy.Cache)
		return runtimeConfig{
			listen:                 legacy.Listen,
			publicURL:              legacy.PublicURL,
			username:               legacy.Username,
			password:               legacy.Password,
			scanEvery:              legacyScanEvery,
			cacheDirectory:         filepath.Dir(legacy.Cache),
			coverCacheMaximumBytes: 512 << 20,
			coverMaximumWidth:      600,
			coverMaximumHeight:     900,
			coverJPEGQuality:       82,
			publicationCache:       filepath.Join(filepath.Dir(legacy.Cache), "publications"),
			publicationCacheBytes:  4 << 30,
			publicationCacheExpiry: 7 * 24 * time.Hour,
			kosyncEnabled:          legacy.Username != "" && legacy.Password != "",
			kosyncWebReaderBridge:  true,
			kosyncDatabase:         filepath.Join(filepath.Dir(legacy.Cache), "kosync.json"),
			libraries:              []web.Library{{ID: "library", Name: "Library", Kind: "books", SourceType: "filesystem", Store: store}},
		}, nil
	}

	cfg, err := appconfig.Load(configPath)
	if err != nil {
		return runtimeConfig{}, err
	}
	scanEvery, err := cfg.ScanDuration()
	if err != nil {
		return runtimeConfig{}, err
	}
	publicationCacheExpiry, err := time.ParseDuration(cfg.PageCache.ExpireAfter)
	if err != nil {
		return runtimeConfig{}, err
	}
	libraries := make([]web.Library, 0, len(cfg.Libraries))
	libraryFingerprints := make(map[string]string, len(cfg.Libraries))
	libraryScanEvery := make(map[string]time.Duration, len(cfg.Libraries))
	modules := cfg.EnabledModules()
	for _, library := range cfg.Libraries {
		if module := appconfig.ModuleForKind(library.Kind); module != "" && !modules[module] {
			// The library stays in the config; it just isn't scanned or served.
			slog.Info("library skipped because its module is off", "library", library.ID, "module", module)
			continue
		}
		cache := filepath.Join(cfg.CacheDirectory, library.ID+".json")
		var backend catalog.Backend
		var videoStore *video.Store
		var audioStore *audio.Store
		var documentStore *document.Store
		var vaultWriter *vaultwriter.Writer
		switch library.Source.Type {
		case "filesystem":
			if supportsVideoLibrary(library.Kind) {
				videoStore = video.NewStore(library.ID, library.Source.Path, filepath.Join(cfg.CacheDirectory, library.ID+"-video.json"))
				if prober, probeErr := video.NewFFProber(env("MOCHIL_FFPROBE", "ffprobe"), 8*time.Second); probeErr == nil {
					videoStore.SetProber(prober)
				}
			} else if supportsAudioLibrary(library.Kind) && !(library.Kind == "audiobooks" && holdsOnlyNarratedBooks(library.Source.Path)) {
				audioStore = audio.NewStore(library.ID, library.Source.Path, filepath.Join(cfg.CacheDirectory, library.ID+"-audio.json"))
				if prober, probeErr := audio.NewFFProber(env("MOCHIL_FFPROBE", "ffprobe"), 8*time.Second); probeErr == nil {
					audioStore.SetProber(prober)
				}
			} else if library.Kind == "documents" || library.Kind == "vault" {
				if library.Kind == "vault" {
					documentStore = document.NewVaultStore(library.ID, library.Source.Path, filepath.Join(cfg.CacheDirectory, library.ID+"-documents.json"))
				} else {
					documentStore = document.NewStore(library.ID, library.Source.Path, filepath.Join(cfg.CacheDirectory, library.ID+"-documents.json"))
				}
				if library.Kind == "vault" && library.VaultWrites {
					vaultWriter, err = vaultwriter.New(library.Source.Path)
					if err != nil {
						return runtimeConfig{}, fmt.Errorf("library %q writable vault: %w", library.ID, err)
					}
				}
			} else {
				backend = catalog.NewFilesystemStore(library.Source.Path, cache, catalog.FilesystemOptions{
					FolderLayout: library.Inference.FolderLayout, FilenameAuthor: library.Inference.FilenameAuthor,
				})
			}
		case "calibre":
			backend = catalog.NewCalibreStore(library.Source.Path, cache)
		default:
			return runtimeConfig{}, fmt.Errorf("library %q has unsupported backend %q", library.ID, library.Source.Type)
		}
		backend = catalog.PreferFormats(backend, library.PreferredFormats)
		kind := library.Kind
		if backend != nil && kind == "audiobooks" {
			slog.Info("audiobook library holds narrated EPUBs only; serving it as books", "library", library.ID)
			kind = "books"
		}
		libraries = append(libraries, web.Library{
			ID:               library.ID,
			Name:             library.Name,
			Kind:             kind,
			SourceType:       library.Source.Type,
			SourcePath:       library.Source.Path,
			MetadataWrites:   library.MetadataWrites,
			MetadataProvider: library.MetadataProvider,
			Store:            backend,
			Video:            videoStore,
			Audio:            audioStore,
			Documents:        documentStore,
			VaultWriter:      vaultWriter,
			PageStreaming:    library.PageStreaming && supportsPageStreaming(library.Kind),
		})
		libraryFingerprints[library.ID] = libraryFingerprint(cfg.CacheDirectory, library)
		every, err := library.ScanDuration(scanEvery)
		if err != nil {
			return runtimeConfig{}, err
		}
		libraryScanEvery[library.ID] = every
	}
	username, password := os.Getenv("MOCHIL_USERNAME"), os.Getenv("MOCHIL_PASSWORD")
	kosyncUsername, kosyncPassword := os.Getenv("MOCHIL_KOSYNC_USERNAME"), os.Getenv("MOCHIL_KOSYNC_PASSWORD")
	if kosyncUsername == "" {
		kosyncUsername = cfg.KoSync.Username
	}
	if kosyncPassword == "" {
		kosyncPassword = cfg.KoSync.Password
	}
	return runtimeConfig{
		modules:                modules,
		narration:              cfg.Narration,
		hermes:                 cfg.Hermes,
		avatar:                 cfg.Avatar,
		listen:                 env("MOCHIL_LISTEN", cfg.Listen),
		publicURL:              env("MOCHIL_PUBLIC_URL", cfg.PublicURL),
		username:               username,
		password:               password,
		scanEvery:              scanEvery,
		libraryScanEvery:       libraryScanEvery,
		cacheDirectory:         cfg.CacheDirectory,
		coverCacheMaximumBytes: cfg.CoverCache.MaximumBytes,
		coverMaximumWidth:      cfg.CoverCache.MaximumWidth,
		coverMaximumHeight:     cfg.CoverCache.MaximumHeight,
		coverJPEGQuality:       cfg.CoverCache.JPEGQuality,
		publicationCache:       cfg.PageCache.Directory,
		publicationCacheBytes:  cfg.PageCache.MaximumBytes,
		publicationCacheExpiry: publicationCacheExpiry,
		kosyncEnabled:          cfg.KoSync.Provider != "disabled" && (cfg.KoSync.Enabled || cfg.KoSync.Provider == "external" || (username != "" && password != "")),
		kosyncWebReaderBridge:  cfg.KoSync.WebReaderBridge == nil || *cfg.KoSync.WebReaderBridge,
		kosyncDatabase:         cfg.KoSync.Database,
		kosyncRegistration:     cfg.KoSync.AllowRegistration,
		kosyncProvider:         cfg.KoSync.Provider,
		kosyncServerURL:        cfg.KoSync.ServerURL,
		kosyncUsername:         kosyncUsername,
		kosyncPassword:         kosyncPassword,
		videoAudioLanguage:     cfg.Playback.AudioLanguage,
		videoSubtitleLanguage:  cfg.Playback.SubtitleLanguage,
		tmdbToken:              cfg.MetadataProviders.TMDBToken,
		hardcoverToken:         cfg.MetadataProviders.HardcoverToken,
		managedIngest:          cfg.ManagedIngest,
		libraries:              libraries,
		libraryFingerprints:    libraryFingerprints,
	}, nil
}

// libraryFingerprint includes only settings that affect a library backend.
// Names, write permissions, and page-streaming flags are carried by the fresh
// web.Library value. Kind is included because it selects the publication or
// video backend for local folders.
func libraryFingerprint(cacheDirectory string, library appconfig.Library) string {
	data, err := json.Marshal(struct {
		CacheDirectory   string              `json:"cache_directory"`
		Kind             string              `json:"kind"`
		Source           appconfig.Source    `json:"source"`
		Inference        appconfig.Inference `json:"inference"`
		PreferredFormats []string            `json:"preferred_formats,omitempty"`
	}{CacheDirectory: cacheDirectory, Kind: library.Kind, Source: library.Source, Inference: library.Inference, PreferredFormats: library.PreferredFormats})
	if err != nil {
		panic(fmt.Sprintf("encode validated library configuration: %v", err))
	}
	return string(data)
}

func supportsVideoLibrary(kind string) bool {
	switch kind {
	case "movies", "tv", "anime":
		return true
	default:
		return false
	}
}

func supportsAudioLibrary(kind string) bool {
	switch kind {
	case "audiobooks", "podcasts", "music":
		return true
	default:
		return false
	}
}

func supportsPageStreaming(kind string) bool {
	switch kind {
	case "comics", "manga", "mixed":
		return true
	default:
		return false
	}
}

// env reads a setting, understanding both the current MOCHIL_ prefix and the MOCHIL_ one this
// project used before the rename. The new name wins when both are set, so an install can be moved
// over one variable at a time and an existing compose file keeps working untouched.
func env(name, fallback string) string {
	if value := lookupEnv(name); value != "" {
		return value
	}
	return fallback
}

// lookupEnv resolves a name across both prefixes. The literal "OPDSHELF_" below is the one place
// that must keep the old spelling: a project-wide rename sweep rewrote it to MOCHIL_ and made the
// fallback look up the same variable twice.
func lookupEnv(name string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	if legacy, ok := strings.CutPrefix(name, "MOCHIL_"); ok {
		if value := os.Getenv("OPDSHELF_" + legacy); value != "" {
			return value
		}
	}
	return ""
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := lookupEnv(name)
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid %s: %v\n", name, err)
		os.Exit(2)
	}
	return d
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(lookupEnv(name)))
	if err != nil {
		return fallback
	}
	return value
}
