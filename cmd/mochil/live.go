package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"mochil/internal/narration"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mochil/internal/audio"
	"mochil/internal/ingest"
	"mochil/internal/jobs"
	"mochil/internal/reading"
	"mochil/internal/safely"
	"mochil/internal/video"
	"mochil/internal/web"
)

// liveApplication owns the process-level runtime boundary. A candidate is
// fully validated and scanned while the current handler keeps serving, then a
// single atomic pointer swap publishes every new route and library together.
type liveApplication struct {
	ctx             context.Context
	logger          *slog.Logger
	configPath      string
	legacy          web.Config
	legacyScanEvery time.Duration

	mu       sync.Mutex
	current  runtimeConfig
	handler  atomic.Value // http.Handler
	scans    scanScheduler
	scanJobs *jobs.Pool
	sessions *web.SessionContinuity
	// stores is the account and device state every handler generation shares.
	stores *web.StoreSet
	// reconcile finishes library transfers that were recorded but not completed.
	// It rides on the catalogue rescan so a repaired transfer is scanned and
	// published by the same tick.
	reconcile func()
}

func (a *liveApplication) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, _ := a.handler.Load().(http.Handler)
	if handler == nil {
		http.Error(w, "Mochil is still starting", http.StatusServiceUnavailable)
		return
	}
	web.Observe(web.Recover(web.Compress(handler))).ServeHTTP(w, r)
}

// liveStores is the account and device state the published handler already owns.
// A rebuild continues those stores; re-opening the files would give the new
// handler a snapshot that does not see an account disabled through a request the
// old handler was still finishing.
func (a *liveApplication) liveStores() *web.StoreSet {
	if handler, ok := a.handler.Load().(interface{ Stores() *web.StoreSet }); ok {
		if stores := handler.Stores(); stores != nil {
			return stores
		}
	}
	if a.stores == nil {
		a.stores = web.NewStoreSet()
	}
	return a.stores
}

// liveReadingStore is the reading store the published handler already owns. A
// rebuild continues that store instead of opening the same state file a second
// time: two stores over one path each keep their own snapshot, and the second
// one's next write erases what the first acknowledged after it was opened.
func (a *liveApplication) liveReadingStore() *reading.Store {
	if handler, ok := a.handler.Load().(interface{ ReadingStore() *reading.Store }); ok {
		return handler.ReadingStore()
	}
	return nil
}

func closeHandler(handler http.Handler) {
	if closer, ok := handler.(interface{ Close() }); ok {
		closer.Close()
	}
}
func (a *liveApplication) close() {
	a.scans.close()
	if a.scanJobs != nil {
		a.scanJobs.Close()
	}
	handler, _ := a.handler.Load().(http.Handler)
	// Reading progress is saved in the background now, so a window of it can
	// still be in memory when the process is asked to stop. Write it before the
	// stores it belongs to are closed, or a restart reopens a stale file.
	if flusher, ok := handler.(interface{ FlushReadingState() error }); ok {
		if err := flusher.FlushReadingState(); err != nil {
			a.logger.Error("reading progress could not be saved before shutdown", "error", err)
		}
	}
	closeHandler(handler)
	closeLibraries(a.current.libraries)
}

func (a *liveApplication) build(runtime runtimeConfig) (http.Handler, error) {
	if a.sessions == nil {
		a.sessions = web.NewSessionContinuity()
	}
	// "Remember me" sign-ins survive redeploys: hashed tokens in a private file beside the other state.
	a.sessions.SetStorePath(filepath.Join(runtime.cacheDirectory, "sessions.json"))
	// Optional modules are only constructed when they are on, so a books-only
	// install runs no ffmpeg pools, artwork caches or analysis workers.
	var (
		subtitles     video.SubtitleExtractor
		hls           *video.HLSStore
		videoArtwork  *video.ArtworkStore
		videoPreviews *video.PreviewStore
		audioArtwork  *audio.ArtworkStore
		audioHLS      *audio.HLSStore
		audioAnalysis *audio.AnalysisQueue
	)
	if runtime.modules["video"] {
		subtitles, _ = video.NewFFmpegSubtitleExtractor(env("MOCHIL_FFMPEG", "ffmpeg"), 45*time.Second)
		hls, _ = video.NewHLSStore(filepath.Join(runtime.cacheDirectory, "video-hls"), env("MOCHIL_FFMPEG", "ffmpeg"), int64(envInt("MOCHIL_HLS_CACHE_MB", 4096))<<20, 2, envInt("MOCHIL_HLS_MAX_HEIGHT", 1080), env("MOCHIL_VAAPI_DEVICE", "/dev/dri/renderD128"))
		videoArtwork, _ = video.NewArtworkStore(filepath.Join(runtime.cacheDirectory, "video-artwork"), env("MOCHIL_FFMPEG", "ffmpeg"), int64(envInt("MOCHIL_VIDEO_ARTWORK_CACHE_MB", 1024))<<20, 2)
		videoPreviews, _ = video.NewPreviewStore(filepath.Join(runtime.cacheDirectory, "video-previews"), env("MOCHIL_FFMPEG", "ffmpeg"), int64(envInt("MOCHIL_VIDEO_PREVIEW_CACHE_MB", 512))<<20, 2)
	}
	if runtime.modules["audio"] {
		audioArtwork, _ = audio.NewArtworkStore(filepath.Join(runtime.cacheDirectory, "audio-artwork"), env("MOCHIL_FFMPEG", "ffmpeg"), int64(envInt("MOCHIL_AUDIO_ARTWORK_CACHE_MB", 512))<<20, 2)
		audioHLS, _ = audio.NewHLSStore(filepath.Join(runtime.cacheDirectory, "audio-hls"), env("MOCHIL_FFMPEG", "ffmpeg"), int64(envInt("MOCHIL_AUDIO_HLS_CACHE_MB", 2048))<<20, 2)
		audioAnalyzer, _ := audio.NewFFAnalyzer(env("MOCHIL_FFMPEG", "ffmpeg"))
		audioAnalysis = audio.NewAnalysisQueue(audioAnalyzer, 1)
	}
	var managedIngest *ingest.Executor
	if runtime.managedIngest.IncomingDirectory != "" {
		var ingestErr error
		managedIngest, ingestErr = ingest.NewExecutor(
			runtime.managedIngest.IncomingDirectory,
			runtime.managedIngest.ManagedDirectory,
			ingest.NewJournalStore(filepath.Join(runtime.cacheDirectory, "managed-ingest.json")),
			runtime.managedIngest.MaximumFileBytes,
		)
		if ingestErr != nil {
			if audioAnalysis != nil {
				audioAnalysis.Close()
			}
			return nil, fmt.Errorf("managed ingest: %w", ingestErr)
		}
	}
	// A flag-only run has no config file: leaving these set made the settings API
	// believe it was managed and answer 500 with a raw OS error instead of 501.
	var configJSON func() ([]byte, error)
	var saveConfigJSON func([]byte) error
	if a.configPath != "" {
		configJSON = func() ([]byte, error) { return os.ReadFile(a.configPath) }
		saveConfigJSON = func(data []byte) error { return saveConfig(a.configPath, data) }
	}
	handler, err := web.NewMulti(web.MultiConfig{
		PublicURL: runtime.publicURL,
		// MOCHIL_ALLOW_OPEN=1 serves the library with no sign-in, for installs that are deliberately
		// behind their own proxy. Without it a fresh install asks for its first account.
		AllowOpen:                    env("MOCHIL_ALLOW_OPEN", "") == "1" || env("MOCHIL_ALLOW_OPEN", "") == "true",
		Modules:                      runtime.modules,
		Narration:                    narration.Config{Python: runtime.narration.Python, Cache: runtime.narration.Cache, Threads: runtime.narration.Threads, Offline: runtime.narration.Offline, IdleSecs: runtime.narration.IdleSeconds, Token: runtime.narration.Token},
		NarrationVoices:              runtime.narration.VoiceDirectory,
		NarrationDefaultVoiceState:   filepath.Join(runtime.cacheDirectory, "narration-default-voice.json"),
		Hermes:                       runtime.hermes,
		Avatar:                       runtime.avatar,
		AvatarState:                  filepath.Join(runtime.cacheDirectory, "avatar-model.json"),
		Username:                     runtime.username,
		Password:                     runtime.password,
		Sessions:                     a.sessions,
		ConfigJSON:                   configJSON,
		SaveConfig:                   saveConfigJSON,
		ApplyConfig:                  a.apply,
		ScanLibrary:                  a.requestLibraryScan,
		Restart:                      func() { _ = terminateForRestart() },
		CoverCache:                   filepath.Join(runtime.cacheDirectory, "covers"),
		CoverCacheMaximumBytes:       runtime.coverCacheMaximumBytes,
		CoverMaximumWidth:            runtime.coverMaximumWidth,
		CoverMaximumHeight:           runtime.coverMaximumHeight,
		CoverJPEGQuality:             runtime.coverJPEGQuality,
		BrowseRoots:                  browseRoots(),
		PublicationCache:             runtime.publicationCache,
		PublicationCacheBytes:        runtime.publicationCacheBytes,
		PublicationCacheExpiry:       runtime.publicationCacheExpiry,
		KoSyncEnabled:                runtime.kosyncEnabled,
		DisableKoSyncWebReaderBridge: !runtime.kosyncWebReaderBridge,
		KoSyncDatabase:               runtime.kosyncDatabase,
		KoSyncRegistration:           runtime.kosyncRegistration,
		KoSyncProvider:               runtime.kosyncProvider,
		KoSyncServerURL:              runtime.kosyncServerURL,
		KoSyncUsername:               runtime.kosyncUsername,
		KoSyncPassword:               runtime.kosyncPassword,
		ReadingState:                 filepath.Join(runtime.cacheDirectory, "reading-state.json"),
		ReadingStore:                 a.liveReadingStore(),
		Stores:                       a.liveStores(),
		VideoState:                   filepath.Join(runtime.cacheDirectory, "video-progress.json"),
		AudioState:                   filepath.Join(runtime.cacheDirectory, "audio-progress.json"),
		AudioQueueState:              filepath.Join(runtime.cacheDirectory, "audio-queue.json"),
		AudioFavoritesState:          filepath.Join(runtime.cacheDirectory, "audio-favorites.json"),
		AudioPlaylistsState:          filepath.Join(runtime.cacheDirectory, "audio-playlists.json"),
		AndroidDiagnosticsState:      filepath.Join(runtime.cacheDirectory, "android-diagnostics.jsonl"),
		PodcastState:                 filepath.Join(runtime.cacheDirectory, "podcast-subscriptions.json"),
		PodcastDownloads:             filepath.Join(runtime.cacheDirectory, "podcast-downloads"),
		PodcastDownloadBytes:         int64(envInt("MOCHIL_PODCAST_DOWNLOAD_CACHE_MB", 20480)) << 20,
		VideoPreferences:             filepath.Join(runtime.cacheDirectory, "video-preferences.json"),
		VideoMetadataState:           filepath.Join(runtime.cacheDirectory, "video-metadata.json"),
		VideoMetadataArtworkCache:    filepath.Join(runtime.cacheDirectory, "video-metadata-artwork"),
		VideoMetadataArtworkBytes:    512 << 20,
		DefaultVideoPreferences:      video.LanguagePreferences{AudioLanguage: runtime.videoAudioLanguage, SubtitleLanguage: runtime.videoSubtitleLanguage},
		TMDBToken:                    runtime.tmdbToken,
		HardcoverToken:               runtime.hardcoverToken,
		ManagedIngest:                managedIngest,
		TransferJournalState:         filepath.Join(runtime.cacheDirectory, "library-transfers.json"),
		SubtitleExtractor:            subtitles,
		HLS:                          hls,
		VideoArtwork:                 videoArtwork,
		VideoPreviews:                videoPreviews,
		AudioArtwork:                 audioArtwork,
		AudioHLS:                     audioHLS,
		AudioAnalysis:                audioAnalysis,
		RuntimeJobs:                  a.scanJobs,
		CollectionsState:             filepath.Join(runtime.cacheDirectory, "collections.json"),
		MetadataReviewState:          filepath.Join(runtime.cacheDirectory, "metadata-reviews.json"),
		MetadataBackupDirectory:      filepath.Join(runtime.cacheDirectory, "metadata-backups"),
		CalibreDBCommand:             env("MOCHIL_CALIBREDB", "calibredb"),
		AndroidAPK:                   env("MOCHIL_ANDROID_APK", ""),
	}, runtime.libraries, a.logger)
	if err != nil {
		if hls != nil {
			hls.Close()
		}
		if videoArtwork != nil {
			videoArtwork.Close()
		}
		if videoPreviews != nil {
			videoPreviews.Close()
		}
		if audioArtwork != nil {
			audioArtwork.Close()
		}
		if audioHLS != nil {
			audioHLS.Close()
		}
		if audioAnalysis != nil {
			audioAnalysis.Close()
		}
		return nil, err
	}
	if reconciler, ok := handler.(interface{ ReconcileTransfers() }); ok {
		a.reconcile = reconciler.ReconcileTransfers
	}
	return handler, nil
}

func (a *liveApplication) apply(data []byte) (web.SettingsApplyResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.configPath == "" {
		return web.SettingsApplyResult{}, errors.New("live settings require a managed config file")
	}
	pending := a.configPath + ".live-pending"
	defer os.Remove(pending)
	if err := saveConfig(pending, data); err != nil {
		return web.SettingsApplyResult{}, err
	}
	candidate, err := configure(pending, a.legacy, a.legacyScanEvery)
	if err != nil {
		return web.SettingsApplyResult{}, err
	}
	toScan, reused := reuseUnchangedLibraries(&candidate, a.current)
	pendingAudio := make([]web.Library, 0)
	for _, library := range toScan {
		if library.Audio != nil {
			pendingAudio = append(pendingAudio, library)
			continue
		}
		if !scanLibraryContext(context.Background(), a.logger, library) {
			closeLibraries(toScan)
			return web.SettingsApplyResult{}, fmt.Errorf("library %q could not be activated; the previous configuration remains active", library.ID)
		}
	}
	// The next server opens the reading state file again before the old one is
	// closed, so anything still inside the background writer's window has to
	// reach the disk first or the new server would start from a stale copy.
	if previous, ok := a.handler.Load().(interface{ FlushReadingState() error }); ok {
		if err := previous.FlushReadingState(); err != nil {
			// A reload that cannot put acknowledged progress on disk is not worth
			// publishing: report it and keep the working configuration running.
			return web.SettingsApplyResult{}, fmt.Errorf("reading progress could not be saved, so the settings were not applied: %w", err)
		}
	}
	next, err := a.build(candidate)
	if err != nil {
		closeLibraries(toScan)
		return web.SettingsApplyResult{}, err
	}
	if err := saveConfig(a.configPath, data); err != nil {
		closeHandler(next)
		closeLibraries(toScan)
		return web.SettingsApplyResult{}, err
	}
	restartRequired := []string(nil)
	if candidate.listen != a.current.listen {
		restartRequired = append(restartRequired, "listen address")
	}
	previousRuntime := a.current
	a.current = candidate
	previous, _ := a.handler.Load().(http.Handler)
	a.handler.Store(next)
	closeHandler(previous)
	a.resetScans(candidate)
	a.closeReplacedLibraries(previousRuntime, candidate)
	for _, library := range pendingAudio {
		a.scheduleLibraryScan(library, candidate.libraryFingerprints[library.ID])
	}
	return web.SettingsApplyResult{
		Saved:            true,
		Applied:          true,
		Libraries:        len(candidate.libraries),
		LibrariesScanned: len(toScan),
		LibrariesReused:  reused,
		RestartRequired:  restartRequired,
	}, nil
}

func (a *liveApplication) scheduleLibraryScan(library web.Library, fingerprint string) {
	if a.scanJobs == nil {
		a.scanJobs = jobs.New(2, 64)
	}
	key := libraryScanKey(library.ID, fingerprint)
	if err := a.scanJobs.Submit(key, func(ctx context.Context) error {
		if !scanLibraryContext(ctx, a.logger, library) {
			return fmt.Errorf("library %q scan failed", library.ID)
		}
		return nil
	}); err != nil && !errors.Is(err, jobs.ErrDuplicate) {
		a.logger.Error("library scan was not queued", "library", library.ID, "error", err)
	}
}

func (a *liveApplication) requestLibraryScan(libraryID string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, library := range a.current.libraries {
		if library.ID != libraryID {
			continue
		}
		key := libraryScanKey(library.ID, a.current.libraryFingerprints[library.ID])
		if err := a.scanJobs.Submit(key, func(ctx context.Context) error {
			if !scanLibraryContext(ctx, a.logger, library) {
				return fmt.Errorf("library %q scan failed", library.ID)
			}
			return nil
		}); err != nil && !errors.Is(err, jobs.ErrDuplicate) {
			return "", err
		}
		return key, nil
	}
	return "", fmt.Errorf("library %q does not exist", libraryID)
}

func libraryScanKey(libraryID, fingerprint string) string {
	digest := sha256.Sum256([]byte(fingerprint))
	return "library-scan:" + libraryID + ":" + hex.EncodeToString(digest[:8])
}

// reuseUnchangedLibraries retains the published backend and its in-memory
// catalogue snapshot when every library-affecting setting is identical.
func reuseUnchangedLibraries(candidate *runtimeConfig, current runtimeConfig) ([]web.Library, int) {
	currentByID := make(map[string]web.Library, len(current.libraries))
	for _, library := range current.libraries {
		currentByID[library.ID] = library
	}
	toScan := make([]web.Library, 0, len(candidate.libraries))
	reused := 0
	for index := range candidate.libraries {
		library := &candidate.libraries[index]
		active, exists := currentByID[library.ID]
		fingerprint := candidate.libraryFingerprints[library.ID]
		if exists && fingerprint != "" && fingerprint == current.libraryFingerprints[library.ID] {
			library.Store = active.Store
			library.Video = active.Video
			library.Audio = active.Audio
			library.Documents = active.Documents
			reused++
			continue
		}
		toScan = append(toScan, *library)
	}
	return toScan, reused
}

type scanScheduler struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// scanPlan is one rescan loop: every interval, run schedule.
type scanPlan struct {
	interval time.Duration
	schedule func()
}

func (s *scanScheduler) resetPlans(parent context.Context, plans []scanPlan) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
	ctx, cancel := context.WithCancel(parent)
	var running sync.WaitGroup
	for _, plan := range plans {
		if plan.interval <= 0 {
			continue
		}
		running.Add(1)
		go func(plan scanPlan) {
			defer running.Done()
			safely.Keep(ctx, "library rescans", func() { rescanLoop(ctx, plan.interval, plan.schedule) })
		}(plan)
	}
	done := make(chan struct{})
	go func() { running.Wait(); close(done) }()
	s.cancel, s.done = cancel, done
}

func (s *scanScheduler) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *scanScheduler) stopLocked() {
	if s.cancel == nil {
		return
	}
	s.cancel()
	if s.done != nil {
		<-s.done
	}
	s.cancel, s.done = nil, nil
}

func (a *liveApplication) resetScans(runtime runtimeConfig) {
	// Libraries that share an interval are rescanned together; a library may set its own.
	byInterval := map[time.Duration][]web.Library{}
	for _, library := range runtime.libraries {
		every, ok := runtime.libraryScanEvery[library.ID]
		if !ok {
			every = runtime.scanEvery
		}
		byInterval[every] = append(byInterval[every], library)
	}
	plans := make([]scanPlan, 0, len(byInterval))
	reconcile := a.reconcile
	for interval, libraries := range byInterval {
		libraries := libraries
		plans = append(plans, scanPlan{interval, func() {
			// Finish recorded transfers before scanning, so files this tick moves
			// are picked up by the scans it schedules.
			if reconcile != nil {
				reconcile()
			}
			for _, library := range libraries {
				a.scheduleLibraryScan(library, runtime.libraryFingerprints[library.ID])
			}
		}})
	}
	a.scans.resetPlans(a.ctx, plans)
}

func closeLibraries(libraries []web.Library) {
	for _, library := range libraries {
		if closer, ok := library.Store.(interface{ Close() }); ok {
			closer.Close()
		}
	}
}

func (a *liveApplication) closeReplacedLibraries(previous, next runtimeConfig) {
	for _, library := range previous.libraries {
		fingerprint := previous.libraryFingerprints[library.ID]
		if fingerprint != "" && fingerprint == next.libraryFingerprints[library.ID] {
			continue
		}
		if a.scanJobs != nil {
			key := libraryScanKey(library.ID, fingerprint)
			if a.scanJobs.Cancel(key) {
				_ = a.scanJobs.Wait(context.Background(), key)
			}
		}
		closeLibraries([]web.Library{library})
	}
}

// browseRoots are the locations administrators may browse when choosing a
// library folder. MOCHIL_BROWSE_ROOTS (comma separated) overrides the
// conventional media mounts.
func browseRoots() []string {
	configured := env("MOCHIL_BROWSE_ROOTS", "/media,/books")
	roots := make([]string, 0, 4)
	for _, root := range strings.Split(configured, ",") {
		if root = strings.TrimSpace(root); root != "" {
			roots = append(roots, root)
		}
	}
	return roots
}
