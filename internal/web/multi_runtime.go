package web

import (
	"errors"
	"fmt"
	"log/slog"
	"mochil/internal/bookcheck"
	"mochil/internal/bookstats"
	"mochil/internal/customcolumns"
	"mochil/internal/filefacts"
	"mochil/internal/isbn"
	"net/url"
	"path/filepath"
	"strings"

	"mochil/internal/accounts"
	"mochil/internal/annotations"
	"mochil/internal/audio"
	"mochil/internal/catalog"
	"mochil/internal/externallibrary"
	"mochil/internal/health"
	"mochil/internal/metadata"
	"mochil/internal/podcast"
	catalogquery "mochil/internal/query"
	"mochil/internal/reading"
	"mochil/internal/video"
	"mochil/internal/wiki"
)

func (m *multiServer) configurePersistentStores(cfg MultiConfig) error {
	if cfg.ReadingState != "" {
		var err error
		if cfg.ReadingStore != nil {
			// This process already owns a store for this file: reuse it, writer and
			// all, rather than opening a second snapshot of the same state.
			m.reading = cfg.ReadingStore
		} else {
			m.reading, err = reading.Open(cfg.ReadingState)
			if err != nil {
				return fmt.Errorf("open reading-state store: %w", err)
			}
			m.readingOwned = true
		}
		m.readingSync = reading.NewSyncCoordinator(m.reading)
		if cfg.moduleOn("hardcover") && strings.TrimSpace(cfg.HardcoverToken) != "" {
			if err := m.startHardcoverSync(filepath.Join(filepath.Dir(cfg.ReadingState), "hardcover-links.json")); err != nil {
				return fmt.Errorf("open Hardcover link store: %w", err)
			}
		}
		m.annotations, err = annotations.Open(filepath.Join(filepath.Dir(cfg.ReadingState), "annotations.json"))
		if err != nil {
			return fmt.Errorf("open annotation store: %w", err)
		}
		m.devices, err = m.stores.koReaderDevices(func() (*koReaderDeviceStore, error) {
			return newKoReaderDeviceStore(filepath.Join(filepath.Dir(cfg.ReadingState), "koreader-devices.json"))
		})
		if err != nil {
			return fmt.Errorf("open KOReader device store: %w", err)
		}
		if cfg.moduleOn("android") {
			m.androidDevices, err = m.stores.deviceStore(func() (*androidDeviceStore, error) {
				return newAndroidDeviceStore(filepath.Join(filepath.Dir(cfg.ReadingState), "android-devices.json"))
			}, false)
			if err != nil {
				return fmt.Errorf("open Android device store: %w", err)
			}
			m.androidPairings = newAndroidPairingStore()
		}
		m.wiki, err = wiki.Open(filepath.Join(filepath.Dir(cfg.ReadingState), "wiki.json"))
		if err != nil {
			return fmt.Errorf("open wiki store: %w", err)
		}
		m.externalLibrary, err = externallibrary.Open(filepath.Join(filepath.Dir(cfg.ReadingState), "external-library.json"))
		if err != nil {
			return fmt.Errorf("open external library store: %w", err)
		}
	}
	// The account store is always opened. With credentials in the environment it is seeded from
	// them, exactly as before; without them it starts empty and the first-run page creates the
	// administrator, which is what keeps a fresh install from serving an open library.
	var err error
	accountsPath := cfg.AccountsState
	if accountsPath == "" && cfg.ReadingState != "" {
		accountsPath = filepath.Join(filepath.Dir(cfg.ReadingState), "accounts.json")
	}
	if accountsPath != "" {
		switch {
		case cfg.Username != "":
			// Credentials in the environment seed the administrator, exactly as before.
			m.accounts, err = m.stores.accountStore(func() (*accounts.Store, error) {
				return accounts.Open(accountsPath, cfg.Username, cfg.Password)
			})
		case fileExists(accountsPath):
			// An install that already created its account signs in with it; no environment needed.
			m.accounts, err = m.stores.accountStore(func() (*accounts.Store, error) {
				return accounts.OpenFirstRun(accountsPath)
			})
		default:
			// Nothing configured and no account yet: this install is in its first run. Keep the
			// store closed (an empty store would make device and administrator checks behave as if
			// an account existed and had been disabled) and remember where the account will go.
			m.firstRunAccountsPath = accountsPath
		}
		if err != nil {
			return fmt.Errorf("open account store: %w", err)
		}
	}
	jellyfinPath := ""
	if cfg.ReadingState != "" {
		jellyfinPath = filepath.Join(filepath.Dir(cfg.ReadingState), "jellyfin-devices.json")
	}
	m.jellyfinDevices, err = m.stores.deviceStore(func() (*androidDeviceStore, error) {
		return newAndroidDeviceStore(jellyfinPath)
	}, true)
	if err != nil {
		return fmt.Errorf("open Jellyfin device store: %w", err)
	}
	if cfg.VideoState != "" {
		var err error
		m.videoProgress, err = openRecovering("video progress", cfg.VideoState, func() (*video.ProgressStore, error) { return video.OpenProgressStore(cfg.VideoState) })
		if err != nil {
			return fmt.Errorf("open video progress store: %w", err)
		}
	}
	if cfg.AudioState != "" {
		var err error
		m.audioProgress, err = openRecovering("audio progress", cfg.AudioState, func() (*audio.ProgressStore, error) { return audio.OpenProgressStore(cfg.AudioState) })
		if err != nil {
			return fmt.Errorf("open audio progress store: %w", err)
		}
	}
	if cfg.AudioQueueState != "" {
		var err error
		m.audioQueue, err = openRecovering("audio queue", cfg.AudioQueueState, func() (*audio.QueueStore, error) { return audio.OpenQueueStore(cfg.AudioQueueState) })
		if err != nil {
			return fmt.Errorf("open audio queue store: %w", err)
		}
	}
	if cfg.AudioFavoritesState != "" {
		var err error
		m.audioFavorites, err = openRecovering("audio favorites", cfg.AudioFavoritesState, func() (*audio.QueueStore, error) { return audio.OpenQueueStore(cfg.AudioFavoritesState) })
		if err != nil {
			return fmt.Errorf("open audio favorites store: %w", err)
		}
	}
	if cfg.AudioPlaylistsState != "" {
		var err error
		m.audioPlaylists, err = openRecovering("audio playlists", cfg.AudioPlaylistsState, func() (*audio.PlaylistStore, error) { return audio.OpenPlaylistStore(cfg.AudioPlaylistsState) })
		if err != nil {
			return fmt.Errorf("open audio playlist store: %w", err)
		}
	}
	if cfg.PodcastState != "" {
		var err error
		m.podcasts, err = podcast.Open(cfg.PodcastState, nil)
		if err != nil {
			return fmt.Errorf("open podcast store: %w", err)
		}
		m.podcastDownloads = podcast.NewDownloader(cfg.PodcastDownloads, nil, cfg.PodcastDownloadBytes)
	}
	if cfg.VideoPreferences != "" {
		var err error
		m.videoPreferences, err = video.OpenPreferenceStore(cfg.VideoPreferences, cfg.DefaultVideoPreferences)
		if err != nil {
			return fmt.Errorf("open video preferences: %w", err)
		}
	}
	if cfg.VideoMetadataState != "" {
		var err error
		m.videoMetadata, err = video.OpenMetadataStore(cfg.VideoMetadataState)
		if err != nil {
			return fmt.Errorf("open video metadata overrides: %w", err)
		}
	}
	if cfg.CollectionsState != "" {
		var err error
		m.collections, err = newCollectionStore(cfg.CollectionsState)
		if err != nil {
			return fmt.Errorf("open collection store: %w", err)
		}
	}
	if cfg.MetadataReviewState != "" {
		var err error
		m.metadataJobs, err = metadata.OpenJobStore(cfg.MetadataReviewState)
		if err != nil {
			return fmt.Errorf("open metadata review store: %w", err)
		}
		overridePath := cfg.MetadataOverrideState
		if overridePath == "" {
			overridePath = filepath.Join(filepath.Dir(cfg.MetadataReviewState), "metadata-overrides.json")
		}
		m.metadataOverrides, err = catalog.OpenOverrideStore(overridePath)
		if err != nil {
			return fmt.Errorf("open metadata override store: %w", err)
		}
		if m.bookStats, err = openRecovering("book counts", filepath.Join(filepath.Dir(overridePath), "book-stats.json"), func() (*bookstats.Store, error) {
			return bookstats.Open(filepath.Join(filepath.Dir(overridePath), "book-stats.json"))
		}); err != nil {
			return fmt.Errorf("open book counts: %w", err)
		}
		if m.bookQuality, err = openRecovering("quality findings", filepath.Join(filepath.Dir(overridePath), "book-quality.json"), func() (*bookcheck.Store, error) {
			return bookcheck.Open(filepath.Join(filepath.Dir(overridePath), "book-quality.json"))
		}); err != nil {
			return fmt.Errorf("open quality findings: %w", err)
		}
		if m.isbnFound, err = openRecovering("found ISBNs", filepath.Join(filepath.Dir(overridePath), "isbn-found.json"), func() (*filefacts.Store[isbn.Result], error) {
			return filefacts.Open[isbn.Result](filepath.Join(filepath.Dir(overridePath), "isbn-found.json"))
		}); err != nil {
			return fmt.Errorf("open found ISBNs: %w", err)
		}
		if m.matchFound, err = openRecovering("found Hardcover matches", filepath.Join(filepath.Dir(overridePath), "hardcover-found.json"), func() (*filefacts.Store[bulkMatch], error) {
			return filefacts.Open[bulkMatch](filepath.Join(filepath.Dir(overridePath), "hardcover-found.json"))
		}); err != nil {
			return fmt.Errorf("open found Hardcover matches: %w", err)
		}
		if m.customColumns, err = openRecovering("custom columns", filepath.Join(filepath.Dir(overridePath), "custom-columns.json"), func() (*customcolumns.Store, error) {
			return customcolumns.Open(filepath.Join(filepath.Dir(overridePath), "custom-columns.json"))
		}); err != nil {
			return fmt.Errorf("open custom columns: %w", err)
		}
		for index := range m.libraries {
			if m.libraries[index].Store == nil {
				continue
			}
			m.libraries[index].Store = m.metadataOverrides.Wrap(m.libraries[index].ID, m.libraries[index].Store)
		}
	}
	return nil
}

func (m *multiServer) configureKoSync(cfg MultiConfig) error {
	if cfg.KoSyncEnabled && cfg.KoSyncProvider == "external" {
		m.kosyncRemote = newKoSyncRemote(cfg.KoSyncServerURL, cfg.KoSyncUsername, cfg.KoSyncPassword)
	} else if cfg.KoSyncEnabled {
		var err error
		m.kosync, err = openRecovering("KoSync", cfg.KoSyncDatabase, func() (*koSyncStore, error) { return newKoSyncStore(cfg.KoSyncDatabase, cfg.KoSyncRegistration) })
		if err != nil {
			return fmt.Errorf("open KoSync store: %w", err)
		}
		syncUsername, syncPassword := cfg.KoSyncUsername, cfg.KoSyncPassword
		if syncUsername == "" {
			syncUsername = cfg.Username
		}
		if syncPassword == "" {
			syncPassword = cfg.Password
		}
		if syncUsername != "" {
			if err := m.kosync.ensureUser(syncUsername, syncPassword); err != nil {
				return fmt.Errorf("initialize KoSync account: %w", err)
			}
		}
		m.kosync.onProgress = m.recordKoSyncProgress
		m.kosync.newerProgress = m.koReaderProgressForWeb
	}
	return nil
}

func (m *multiServer) configureLibraries(cfg MultiConfig, logger *slog.Logger) error {
	seen := make(map[string]bool, len(m.libraries))
	for libraryIndex := range m.libraries {
		library := &m.libraries[libraryIndex]
		if library.ID == "" || library.Name == "" || (library.Store == nil && library.Video == nil && library.Audio == nil && library.Documents == nil) {
			return errors.New("every library requires an ID, name, and store")
		}
		if seen[library.ID] {
			return fmt.Errorf("duplicate library ID %q", library.ID)
		}
		seen[library.ID] = true
		if library.Store == nil {
			continue
		}
		base := "/opds/libraries/" + url.PathEscape(library.ID)
		handler, err := New(Config{
			PublicURL:       cfg.PublicURL,
			BasePath:        base,
			AcquisitionPath: "/books/" + url.PathEscape(library.ID),
			StreamingPath:   "/stream/" + url.PathEscape(library.ID),
			PageStreaming:   library.PageStreaming,
			LibraryName:     library.Name,
		}, library.Store, logger)
		if err != nil {
			return err
		}
		m.handlers[library.ID] = handler
	}
	queryLibraries := make([]catalogquery.Library, 0, len(m.libraries))
	for _, library := range m.libraries {
		if library.Store == nil {
			continue
		}
		queryLibraries = append(queryLibraries, catalogquery.Library{ID: library.ID, Name: library.Name, Kind: library.Kind, Store: library.Store})
	}
	m.query = catalogquery.New(queryLibraries)
	m.health = health.New(m.healthSnapshot)
	return nil
}
