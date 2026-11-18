package web

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mochil/internal/bookcheck"
	"mochil/internal/bookstats"
	"mochil/internal/customcolumns"
	"mochil/internal/filefacts"
	"mochil/internal/isbn"
	"mochil/internal/narration"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mochil/internal/accounts"
	"mochil/internal/acquisition"
	"mochil/internal/annotations"
	"mochil/internal/appconfig"
	"mochil/internal/audio"
	"mochil/internal/catalog"
	"mochil/internal/document"
	"mochil/internal/enrichment"
	"mochil/internal/externallibrary"
	"mochil/internal/health"
	"mochil/internal/ingest"
	"mochil/internal/jobs"
	"mochil/internal/metadata"
	"mochil/internal/podcast"
	catalogquery "mochil/internal/query"
	"mochil/internal/reading"
	settingssvc "mochil/internal/settings"
	"mochil/internal/vaultwriter"
	"mochil/internal/video"
	"mochil/internal/wiki"
)

type Library struct {
	ID               string
	Name             string
	Kind             string
	SourceType       string
	SourcePath       string
	MetadataWrites   bool
	MetadataProvider string
	Store            catalog.Backend
	Video            *video.Store
	Audio            *audio.Store
	Documents        *document.Store
	VaultWriter      *vaultwriter.Writer
	PageStreaming    bool
}

type MultiConfig struct {
	PublicURL string
	Username  string
	Password  string
	// AllowOpen serves the library without a sign-in. It exists for installs that sit behind their
	// own proxy or on a machine only the owner reaches; a fresh install that leaves it off asks for
	// its first account instead of publishing itself to whoever finds the port.
	AllowOpen  bool
	Sessions   *SessionContinuity
	ConfigJSON func() ([]byte, error)
	// Modules is the effective on/off state of each optional module. Nil means
	// every module is on, which keeps single-library and test setups unchanged.
	Modules                map[string]bool
	SaveConfig             func([]byte) error
	ApplyConfig            func([]byte) (SettingsApplyResult, error)
	ScanLibrary            func(string) (string, error)
	Restart                func()
	CoverCache             string
	CoverCacheMaximumBytes int64
	CoverMaximumWidth      int
	CoverMaximumHeight     int
	CoverJPEGQuality       int
	// BrowseRoots limits the folders administrators can browse when choosing a library location.
	BrowseRoots                  []string
	PublicationCache             string
	PublicationCacheBytes        int64
	PublicationCacheExpiry       time.Duration
	KoSyncEnabled                bool
	DisableKoSyncWebReaderBridge bool
	KoSyncDatabase               string
	KoSyncRegistration           bool
	KoSyncProvider               string
	KoSyncServerURL              string
	KoSyncUsername               string
	KoSyncPassword               string
	ReadingState                 string
	// ReadingStore, when set, is the process's live reading store. A live
	// settings reload hands the running one over instead of opening the same file
	// a second time: two stores over one path each hold their own snapshot, so a
	// save through the old handler disappeared at the successor's next write.
	ReadingStore *reading.Store
	// Stores, when set, is the process's live security state (accounts and device
	// registries). A live reload hands it over so both handler generations mutate
	// one store; see StoreSet.
	Stores                     *StoreSet
	VideoState                 string
	AudioState                 string
	AudioQueueState            string
	AudioFavoritesState        string
	AudioPlaylistsState        string
	PodcastState               string
	PodcastDownloads           string
	PodcastDownloadBytes       int64
	VideoPreferences           string
	VideoMetadataState         string
	VideoMetadataArtworkCache  string
	VideoMetadataArtworkBytes  int64
	DefaultVideoPreferences    video.LanguagePreferences
	TMDBToken                  string
	HardcoverToken             string
	SubtitleExtractor          video.SubtitleExtractor
	HLS                        *video.HLSStore
	VideoArtwork               *video.ArtworkStore
	VideoPreviews              *video.PreviewStore
	AudioArtwork               *audio.ArtworkStore
	AudioHLS                   *audio.HLSStore
	AudioAnalysis              *audio.AnalysisQueue
	RuntimeJobs                *jobs.Pool
	CollectionsState           string
	MetadataReviewState        string
	MetadataOverrideState      string
	MetadataBackupDirectory    string
	CalibreDBCommand           string
	AndroidAPK                 string
	Narration                  narration.Config
	NarrationVoices            string
	NarrationDefaultVoiceState string
	Avatar                     appconfig.Avatar
	AvatarState                string
	Hermes                     appconfig.Hermes
	AndroidDiagnosticsState    string
	ManagedIngest              *ingest.Executor
	AccountsState              string
	// TransferJournalState is the file that records transfers whose bookkeeping
	// steps have not all finished, so they can be completed after a restart.
	TransferJournalState string
}

type SettingsApplyResult = settingssvc.ApplyResult

type multiServer struct {
	cfg                    MultiConfig
	libraries              []Library
	handlers               map[string]http.Handler
	root                   *Server
	covers                 *coverCache
	publications           *acquisition.PublicationCache
	koSyncContent          koSyncDigests
	kosync                 *koSyncStore
	kosyncRemote           *koSyncRemote
	reading                *reading.Store
	readingOwned           bool // false when a live reload handed its store over
	readingSync            *reading.SyncCoordinator
	annotations            *annotations.Store
	videoProgress          *video.ProgressStore
	audioProgress          *audio.ProgressStore
	hermesPlays            *hermesPlayStore
	hermesPlaysOnce        sync.Once
	audioQueue             *audio.QueueStore
	audioFavorites         *audio.QueueStore
	audioPlaylists         *audio.PlaylistStore
	podcasts               *podcast.Store
	podcastDownloads       *podcast.Downloader
	videoPreferences       *video.PreferenceStore
	videoMetadata          *video.MetadataStore
	metadataArtwork        *enrichment.ArtworkCache
	subtitles              video.SubtitleExtractor
	hls                    *video.HLSStore
	videoViewers           videoViewers
	videoArtwork           *video.ArtworkStore
	videoPreviews          *video.PreviewStore
	audioArtwork           *audio.ArtworkStore
	audioHLS               *audio.HLSStore
	audioAnalysis          *audio.AnalysisQueue
	collections            *collectionStore
	externalLibrary        *externallibrary.Store
	metadataJobs           *metadata.JobStore
	metadataOverrides      *catalog.OverrideStore
	customColumns          *customcolumns.Store
	bookCounts             sync.Map   // see health_http.go
	hermesTurn             sync.Mutex // one helper request to Hermes at a time, see hermes_ask.go
	musicEngines           sync.Map   // library id -> *musicrename.Engine
	polishMu               sync.Mutex // guards polishJobs, see hermes_polish.go
	polishJobs             map[string]*polishJob
	bookStats              *bookstats.Store              // words and pages counted for the editor
	bookQuality            *bookcheck.Store              // what the editor's quality check found in each file
	isbnFound              *filefacts.Store[isbn.Result] // the ISBNs the editor found by reading books
	matchFound             *filefacts.Store[bulkMatch]   // what the editor's Hardcover lookups found for each book
	hardcoverLinks         *hardcoverLinkStore
	hardcoverSkipped       hardcoverSkipped
	hardcoverMatches       hardcoverMatches
	hardcoverIgnored       hardcoverIgnored
	editorUndo             editorUndo
	hardcoverEnqueue       func(hardcoverJob) // set only while the Hardcover module runs
	metadataApplyMu        sync.Mutex
	ingestPreviewMu        sync.Mutex
	ingestPreviews         map[string]cachedIngestPreview
	managedIngest          *ingest.Executor
	ingestMu               sync.Mutex
	ingestTargets          map[string]*ingest.Executor
	openLibrary            *enrichment.OpenLibrary
	tmdb                   *enrichment.TMDB
	hardcover              *enrichment.Hardcover
	calibre                metadata.CalibreRunner
	calibreError           string
	devices                *koReaderDeviceStore
	androidDevices         *androidDeviceStore
	dropSessions           func(account string) error // signs an account out of every browser
	forgetBasicAuth        func()                     // drops memoized Basic-auth results after account changes
	jellyfinDevices        *androidDeviceStore
	jellyfinLogins         jellyfinLimiter
	androidPairings        *androidPairingStore
	androidDiagnosticStore *androidDiagnosticStore
	narration              *narration.Engine
	narrationVoices        *narration.Voices
	accounts               *accounts.Store
	stores                 *StoreSet
	// Where the first account will be written on an install that has none yet; empty once it has one.
	firstRunAccountsPath string
	accountMutationMu    sync.Mutex
	coverSlots           chan struct{}
	// subtitleSlots bounds concurrent subtitle extractions: every request starts its
	// own ffmpeg pass over the whole file, and nothing limited how many could run.
	subtitleSlots       chan struct{}
	query               *catalogquery.Service
	wiki                *wiki.Store
	settingsService     settingssvc.Service
	health              *health.Service
	backgroundJobs      *jobs.Pool
	runtimeJobs         *jobs.Pool
	scanLibrary         func(string) (string, error)
	transferJournalOnce sync.Once
	transferJournal     *transferJournal
	transferJournalErr  error
}

type managedHandler struct {
	http.Handler
	close        func()
	reconcile    func()
	flushReading func() error
	readingStore *reading.Store
	stores       *StoreSet
}

// ReadingStore hands the live reading store to a replacement handler, so a
// reload continues the same writer instead of opening a second one.
func (h *managedHandler) ReadingStore() *reading.Store { return h.readingStore }

// Stores hands the live account and device registries to a replacement handler,
// so an account change made through either generation is one change.
func (h *managedHandler) Stores() *StoreSet { return h.stores }

func (h *managedHandler) Close() {
	if h.close != nil {
		h.close()
	}
}

// FlushReadingState puts pending reading progress on disk. A live settings
// reload builds a second server that opens the same state file before the old
// one is closed, so the pending window must be written first.
func (h *managedHandler) FlushReadingState() error {
	if h.flushReading == nil {
		return nil
	}
	return h.flushReading()
}

// ReconcileTransfers lets the process's scan scheduler finish interrupted
// library transfers on the same schedule as the catalogue rescan.
func (h *managedHandler) ReconcileTransfers() {
	if h.reconcile != nil {
		h.reconcile()
	}
}

func NewMulti(cfg MultiConfig, libraries []Library, logger *slog.Logger) (http.Handler, error) {
	if (cfg.Username == "") != (cfg.Password == "") {
		return nil, errors.New("username and password must be configured together")
	}
	stores := cfg.Stores
	if stores == nil {
		stores = NewStoreSet()
	}
	m := &multiServer{
		cfg:             cfg,
		stores:          stores,
		libraries:       append([]Library(nil), libraries...),
		handlers:        make(map[string]http.Handler, len(libraries)),
		root:            &Server{cfg: Config{PublicURL: strings.TrimRight(cfg.PublicURL, "/"), BasePath: "/opds"}, log: logger},
		coverSlots:      make(chan struct{}, 3),
		subtitleSlots:   make(chan struct{}, 2),
		subtitles:       cfg.SubtitleExtractor,
		hls:             cfg.HLS,
		videoArtwork:    cfg.VideoArtwork,
		videoPreviews:   cfg.VideoPreviews,
		audioArtwork:    cfg.AudioArtwork,
		audioHLS:        cfg.AudioHLS,
		audioAnalysis:   cfg.AudioAnalysis,
		backgroundJobs:  jobs.New(2, 64),
		runtimeJobs:     cfg.RuntimeJobs,
		scanLibrary:     cfg.ScanLibrary,
		openLibrary:     enrichment.NewOpenLibrary(nil),
		tmdb:            enrichment.NewTMDB(cfg.TMDBToken, nil),
		hardcover:       enrichment.NewHardcover(cfg.HardcoverToken, nil),
		metadataArtwork: enrichment.NewArtworkCache(cfg.VideoMetadataArtworkCache, cfg.VideoMetadataArtworkBytes, nil),
		managedIngest:   cfg.ManagedIngest,
	}
	if cfg.VideoMetadataState != "" {
		m.tmdb.PersistEpisodes(filepath.Join(filepath.Dir(cfg.VideoMetadataState), "tmdb-episodes.json"))
		m.tmdb.PersistDetails(filepath.Join(filepath.Dir(cfg.VideoMetadataState), "tmdb-details.json"))
	}
	if cfg.Modules != nil && cfg.Modules["narration"] {
		m.narration = narration.New(cfg.Narration)
		if cfg.NarrationVoices != "" {
			m.narrationVoices = narration.NewVoices(cfg.NarrationVoices)
		}
	}
	if cfg.moduleOn("android") {
		m.androidDiagnosticStore = newAndroidDiagnosticStore(cfg.AndroidDiagnosticsState, 2<<20)
	}
	built := false
	defer func() {
		if !built {
			m.close()
		}
	}()
	effectiveUsername := cfg.KoSyncUsername
	passwordConfigured := cfg.KoSyncPassword != ""
	if effectiveUsername == "" && cfg.KoSyncProvider != "external" {
		effectiveUsername = cfg.Username
		passwordConfigured = cfg.Password != ""
	}
	m.settingsService = settingssvc.Service{ReadSource: cfg.ConfigJSON, SaveSource: cfg.SaveConfig, ApplySource: cfg.ApplyConfig, EffectiveUser: effectiveUsername, PasswordConfigured: passwordConfigured}
	if command, err := metadata.FindCalibreCommand(cfg.CalibreDBCommand); err != nil {
		m.calibreError = err.Error()
	} else {
		m.calibre = command
	}
	if cfg.CoverCache != "" {
		m.covers = newCoverCache(cfg.CoverCache, cfg.CoverCacheMaximumBytes, cfg.CoverMaximumWidth, cfg.CoverMaximumHeight, cfg.CoverJPEGQuality, logger)
		m.scheduleCoverMaintenance()
	}
	if cfg.PublicationCache != "" {
		m.publications = acquisition.NewPublicationCache(cfg.PublicationCache, cfg.PublicationCacheBytes, cfg.PublicationCacheExpiry, logger)
		m.schedulePublicationMaintenance()
	}
	if err := m.configurePersistentStores(cfg); err != nil {
		return nil, err
	}
	if err := m.configureKoSync(cfg); err != nil {
		return nil, err
	}
	if err := m.configureLibraries(cfg, logger); err != nil {
		return nil, err
	}
	// Open the transfer journal before serving and finish anything a previous
	// run left behind: the stores are open and the libraries are known now.
	if _, err := m.transfers(); err != nil {
		return nil, err
	}
	m.reconcileTransfers()
	var handler http.Handler = m
	if (cfg.Username != "" || m.accounts != nil || m.needsFirstAccount()) && !cfg.AllowOpen {
		sessions := cfg.Sessions
		if sessions == nil {
			sessions = NewSessionContinuity()
		}
		m.dropSessions = sessions.revokeAccount
		m.forgetBasicAuth = sessions.forgetCredentials
		sessions.setSetup(m.needsFirstAccount, m.createFirstRunAccount)
		handler = sessions.wrap(cfg.Username, cfg.Password, m.authenticateAccount, m.resolveAccountIdentity, handler, func(r *http.Request) bool {
			return r.URL.Path == "/healthz" || r.URL.Path == "/sw.js" || r.URL.Path == "/manifest.webmanifest" || strings.HasPrefix(r.URL.Path, "/assets/") || (m.kosync != nil && isKoSyncPath(r.URL.Path)) || strings.HasPrefix(r.URL.Path, "/api/koreader/v1/") || (strings.HasPrefix(r.URL.Path, "/api/android/v1/") && r.URL.Path != "/api/android/v1/login") || isJellyfinPath(r.URL.Path)
		})
	}
	built = true
	return &managedHandler{Handler: handler, close: m.close, reconcile: m.reconcileTransfers, flushReading: m.flushReading, readingStore: m.reading, stores: m.stores}, nil
}

func (m *multiServer) close() {
	// Reading progress is written in the background, so shutdown has to put the
	// last window on disk itself. Everything below only stops goroutines; this
	// is the one store whose data would otherwise be lost silently.
	if m.reading != nil && m.readingOwned {
		if err := m.reading.Close(); err != nil {
			slog.Error("reading progress could not be saved at shutdown", "error", err)
		}
	}
	if m.narration != nil {
		m.narration.Close()
	}
	if m.podcastDownloads != nil {
		m.podcastDownloads.Close()
	}
	if m.backgroundJobs != nil {
		m.backgroundJobs.Close()
	}
	if m.videoArtwork != nil {
		m.videoArtwork.Close()
	}
	if m.videoPreviews != nil {
		m.videoPreviews.Close()
	}
	if m.audioArtwork != nil {
		m.audioArtwork.Close()
	}
	if m.audioHLS != nil {
		m.audioHLS.Close()
	}
	if m.audioAnalysis != nil {
		m.audioAnalysis.Close()
	}
	if m.hls != nil {
		m.hls.Close()
	}
}

// flushReading writes pending reading progress without stopping the writer; see
// managedHandler.FlushReadingState.
func (m *multiServer) flushReading() error {
	if m.reading == nil {
		return nil
	}
	return m.reading.Flush()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (m *multiServer) serveAcquisition(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, "/books/")
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) > 0 {
		id, err := url.PathUnescape(parts[0])
		if err == nil {
			if _, ok := m.handlers[id]; ok {
				m.serveLibrary(w, r, "/books/", false)
				return
			}
		}
	}
	if len(m.libraries) == 1 {
		serveRewritten(m.handlers[m.libraries[0].ID], w, r, r.URL.Path)
		return
	}
	http.NotFound(w, r)
}

func (m *multiServer) serveStream(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, "/stream/")
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id, err := url.PathUnescape(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	handler, ok := m.handlers[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveRewritten(handler, w, r, "/stream/"+parts[1])
}

func (m *multiServer) serveLibrary(w http.ResponseWriter, r *http.Request, prefix string, catalogRoute bool) {
	remainder := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(remainder, "/", 2)
	id, err := url.PathUnescape(parts[0])
	if err != nil || id == "" {
		http.NotFound(w, r)
		return
	}
	handler, ok := m.handlers[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	suffix := ""
	if len(parts) == 2 {
		suffix = "/" + parts[1]
	}
	if suffix == "/" {
		suffix = ""
	}
	internalPath := "/books" + suffix
	if catalogRoute {
		if suffix == "/opensearch.xml" {
			internalPath = "/opensearch.xml"
		} else {
			internalPath = "/opds" + suffix
		}
	}
	serveRewritten(handler, w, r, internalPath)
}

func (m *multiServer) serveLegacy(w http.ResponseWriter, r *http.Request) {
	library := m.libraries[0]
	handler := m.handlers[library.ID]
	serveRewritten(handler, w, r, r.URL.Path)
}

func serveRewritten(handler http.Handler, w http.ResponseWriter, r *http.Request, path string) {
	clone := r.Clone(r.Context())
	clonedURL := *r.URL
	clonedURL.Path = path
	clonedURL.RawPath = ""
	clone.URL = &clonedURL
	handler.ServeHTTP(w, clone)
}
