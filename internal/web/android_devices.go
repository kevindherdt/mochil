package web

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mochil/internal/catalog"
	"mochil/internal/webpub"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"mochil/internal/accounts"
)

const androidMaximumDevices = 16

type androidDevice struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TokenHash string    `json:"token_hash"`
	CreatedAt time.Time `json:"created_at"`
}

type androidDeviceFile struct {
	Version  int                        `json:"version"`
	Accounts map[string][]androidDevice `json:"accounts"`
}

type androidDeviceStore struct {
	path    string
	now     func() time.Time
	mu      sync.Mutex
	devices androidDeviceFile
}

func newAndroidDeviceStore(path string) (*androidDeviceStore, error) {
	store := &androidDeviceStore{path: path, now: time.Now, devices: androidDeviceFile{Version: 1, Accounts: map[string][]androidDevice{}}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 1<<20 || json.Unmarshal(data, &store.devices) != nil || store.devices.Version != 1 || store.devices.Accounts == nil {
		return nil, errors.New("invalid Android device state")
	}
	for account, devices := range store.devices.Accounts {
		if len(devices) > androidMaximumDevices {
			store.devices.Accounts[account] = devices[len(devices)-androidMaximumDevices:]
		}
	}
	return store, nil
}

func (s *androidDeviceStore) issue(account, name string) (string, string, error) {
	account, name = strings.TrimSpace(account), strings.TrimSpace(name)
	if account == "" || name == "" || len(name) > 80 {
		return "", "", errors.New("Android device identity is invalid")
	}
	raw, idRaw := make([]byte, 32), make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	if _, err := rand.Read(idRaw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	deviceID := base64.RawURLEncoding.EncodeToString(idRaw)

	s.mu.Lock()
	defer s.mu.Unlock()
	previous := append([]androidDevice(nil), s.devices.Accounts[account]...)
	devices := append(previous, androidDevice{ID: deviceID, Name: name, TokenHash: base64.RawStdEncoding.EncodeToString(digest[:]), CreatedAt: s.now().UTC()})
	if len(devices) > androidMaximumDevices {
		devices = devices[len(devices)-androidMaximumDevices:]
	}
	s.devices.Accounts[account] = devices
	if err := s.saveLocked(); err != nil {
		s.devices.Accounts[account] = previous
		return "", "", err
	}
	return token, deviceID, nil
}

func (s *androidDeviceStore) authenticate(token string) (string, string, bool) {
	digest := sha256.Sum256([]byte(strings.TrimSpace(token)))
	s.mu.Lock()
	defer s.mu.Unlock()
	for account, devices := range s.devices.Accounts {
		for _, device := range devices {
			want, err := base64.RawStdEncoding.DecodeString(device.TokenHash)
			if err == nil && len(want) == len(digest) && subtle.ConstantTimeCompare(want, digest[:]) == 1 {
				return account, device.ID, true
			}
		}
	}
	return "", "", false
}

func (s *androidDeviceStore) revoke(account, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := append([]androidDevice(nil), s.devices.Accounts[account]...)
	remaining := make([]androidDevice, 0, len(previous))
	for _, device := range previous {
		if device.ID != deviceID {
			remaining = append(remaining, device)
		}
	}
	if len(remaining) == len(previous) {
		return nil
	}
	s.devices.Accounts[account] = remaining
	if err := s.saveLocked(); err != nil {
		s.devices.Accounts[account] = previous
		return err
	}
	return nil
}

func (s *androidDeviceStore) revokeAccount(account string) error {
	if account == "" {
		return errors.New("invalid account")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, exists := s.devices.Accounts[account]
	if !exists {
		return nil
	}
	delete(s.devices.Accounts, account)
	if err := s.saveLocked(); err != nil {
		s.devices.Accounts[account] = previous
		return err
	}
	return nil
}

func (s *androidDeviceStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.devices, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".android-devices-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = temporary.Chmod(0o600); err == nil {
		_, err = temporary.Write(data)
	}
	if syncErr := temporary.Sync(); err == nil {
		err = syncErr
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, s.path)
	}
	return err
}

func (m *multiServer) androidLoginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.androidDevices == nil || strings.TrimSpace(m.cfg.PublicURL) == "" {
		http.Error(w, "Android login requires persistent reading state and public_url", http.StatusNotImplemented)
		return
	}
	var input struct {
		DeviceName string `json:"device_name"`
	}
	if err := decodeJSONRequest(w, r, 4<<10, &input, true); err != nil {
		http.Error(w, "Invalid Android login request", http.StatusBadRequest)
		return
	}
	token, deviceID, err := m.androidDevices.issue(m.readingAccount(r), input.DeviceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]string{"device_id": deviceID, "device_token": token, "server_url": strings.TrimRight(m.cfg.PublicURL, "/")})
}

func (m *multiServer) androidAPI(w http.ResponseWriter, r *http.Request) {
	if !m.authorizeAndroid(w, r) {
		return
	}
	if !m.authorizeAdministratorFor(w, r, sharedPath(r.URL.Path)) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	switch {
	case r.URL.Path == "/api/android/v1/device":
		m.androidDeviceAPI(w, r)
	case r.URL.Path == "/api/android/v1/app":
		m.androidAppInfo(w, r)
	case r.URL.Path == "/api/android/v1/tmdb-portrait":
		m.tmdbPortrait(w, r)
	case r.URL.Path == "/api/android/v1/changelog":
		m.apiChangelog(w, r)
	case r.URL.Path == "/api/android/v1/app.apk":
		m.androidAppDownload(w, r)
	case r.URL.Path == "/api/android/v1/libraries":
		m.apiLibraries(w, r)
	case r.URL.Path == "/api/android/v1/home":
		m.androidHome(w, r)
	case r.URL.Path == "/api/android/v1/books":
		m.apiBooks(w, r)
	case r.URL.Path == "/api/android/v1/series":
		m.apiSeries(w, r)
	case r.URL.Path == "/api/android/v1/authors":
		m.apiAuthors(w, r)
	case r.URL.Path == "/api/android/v1/audio":
		m.apiAudio(w, r)
	case r.URL.Path == "/api/android/v1/documents":
		forwardAndroidPath(r, "/api/android/v1/documents", "/api/documents", m.apiDocuments, w)
	case r.URL.Path == "/api/android/v1/videos":
		forwardAndroidPath(r, "/api/android/v1/videos", "/api/videos", m.apiVideos, w)
	case r.URL.Path == "/api/android/v1/video-shows":
		forwardAndroidPath(r, "/api/android/v1/video-shows", "/api/video-shows", m.apiVideoShows, w)
	case r.URL.Path == "/api/android/v1/video-progress":
		forwardAndroidPath(r, "/api/android/v1/video-progress", "/api/video-progress", m.apiVideoProgress, w)
	case r.URL.Path == "/api/android/v1/audio-groups":
		m.androidAudioGroups(w, r)
	case r.URL.Path == "/api/android/v1/audio-navigation":
		forwardAndroidPath(r, "/api/android/v1/audio-navigation", "/api/audio-navigation", m.apiAudioNavigation, w)
	case r.URL.Path == "/api/android/v1/audio-group-tracks":
		m.androidAudioGroupTracks(w, r)
	case r.URL.Path == "/api/android/v1/audio-artist-albums":
		m.androidArtistAlbums(w, r)
	case r.URL.Path == "/api/android/v1/reading":
		m.apiReading(w, r)
	case r.URL.Path == "/api/android/v1/collections" || strings.HasPrefix(r.URL.Path, "/api/android/v1/collections/"):
		forwardAndroidPath(r, "/api/android/v1/collections", "/api/collections", m.apiCollections, w)
	case r.URL.Path == "/api/android/v1/external-library" || strings.HasPrefix(r.URL.Path, "/api/android/v1/external-library/"):
		m.apiExternalLibrary(w, r)
	case r.URL.Path == "/api/android/v1/audio-progress":
		forwardAndroidPath(r, "/api/android/v1/audio-progress", "/api/audio-progress", m.apiAudioProgress, w)
	case r.URL.Path == "/api/android/v1/audio-queue":
		forwardAndroidPath(r, "/api/android/v1/audio-queue", "/api/audio-queue", m.apiAudioQueue, w)
	case r.URL.Path == "/api/android/v1/audio-favorites":
		forwardAndroidPath(r, "/api/android/v1/audio-favorites", "/api/audio-favorites", m.apiAudioFavorites, w)
	case r.URL.Path == "/api/android/v1/audio-playlists":
		forwardAndroidPath(r, "/api/android/v1/audio-playlists", "/api/audio-playlists", m.apiAudioPlaylists, w)
	case r.URL.Path == "/api/android/v1/podcast-shows":
		m.androidPodcastShows(w, r)
	case r.URL.Path == "/api/android/v1/podcast-search":
		m.apiPodcastSearch(w, r)
	case r.URL.Path == "/api/android/v1/podcast-search-artwork":
		forwardAndroidPath(r, "/api/android/v1/podcast-search-artwork", "/api/podcast-search-artwork", m.podcastSearchArtwork, w)
	case r.URL.Path == "/api/android/v1/podcast-feeds":
		forwardAndroidPath(r, "/api/android/v1/podcast-feeds", "/api/podcast-feeds", m.apiPodcastFeeds, w)
	case r.URL.Path == "/api/android/v1/crash-report":
		m.androidCrashReport(w, r)
	case r.URL.Path == "/api/android/v1/diagnostics":
		m.androidDiagnostics(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/covers/"):
		forward := r.Clone(r.Context())
		forwardURL := *r.URL
		forwardURL.Path = "/covers/" + strings.TrimPrefix(r.URL.Path, "/api/android/v1/covers/")
		forward.URL = &forwardURL
		m.cover(w, forward)
	case r.URL.Path == "/api/android/v1/hermes" || strings.HasPrefix(r.URL.Path, "/api/android/v1/hermes/"):
		m.androidHermes(w, r)
	case r.URL.Path == "/api/android/v1/avatar" || strings.HasPrefix(r.URL.Path, "/api/android/v1/avatar/"):
		m.androidAvatar(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/narration/"):
		forwardAndroidPath(r, "/api/android/v1/narration/", "/api/narration/", m.narrationAPI, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/publications/") && strings.Contains(r.URL.Path, "/pages/"):
		m.androidPublicationPage(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/publications/"):
		m.androidPublication(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/readium/"):
		// A read-along book's overlay and its audio, which the app streams instead of storing.
		forwardAndroidPath(r, "/api/android/v1/readium/", "/api/readium/", m.readiumBook, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/reading/"):
		forwardAndroidPath(r, "/api/android/v1/reading/", "/api/reading/", m.apiReadingBook, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/annotations/"):
		forwardAndroidPath(r, "/api/android/v1/annotations/", "/api/annotations/", m.annotationAPI, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/documents/"):
		forwardAndroidPath(r, "/api/android/v1/documents/", "/api/documents/", m.apiDocumentText, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/document-assets/"):
		forwardAndroidPath(r, "/api/android/v1/document-assets/", "/documents/", m.serveDocument, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/vaults/"):
		forwardAndroidPath(r, "/api/android/v1/vaults/", "/api/vaults/", m.apiVaultMutation, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/audio-progress/"):
		forwardAndroidPath(r, "/api/android/v1/audio-progress/", "/api/audio-progress/", m.apiAudioProgress, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/audio-playlists/"):
		forwardAndroidPath(r, "/api/android/v1/audio-playlists/", "/api/audio-playlists/", m.apiAudioPlaylists, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/audio-artwork/"):
		forwardAndroidPath(r, "/api/android/v1/audio-artwork/", "/api/audio-artwork/", m.audioArtworkHTTP, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/audio-stream/"):
		forwardAndroidPath(r, "/api/android/v1/audio-stream/", "/audio/", m.streamAudio, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/audio-hls/"):
		forward := r.Clone(r.Context())
		forward.Header.Set("X-Mochil-Android-Audio", "1")
		forwardAndroidPath(forward, "/api/android/v1/audio-hls/", "/api/audio-hls/", m.audioHLSHTTP, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/video-details/"):
		forwardAndroidPath(r, "/api/android/v1/video-details/", "/api/video-details/", m.apiVideoDetails, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/video-artwork/"):
		forwardAndroidPath(r, "/api/android/v1/video-artwork/", "/api/video-artwork/", m.videoArtworkHTTP, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/video-previews/"):
		forwardAndroidPath(r, "/api/android/v1/video-previews/", "/api/video-previews/", m.videoPreviewHTTP, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/video-stream/"):
		forwardAndroidPath(r, "/api/android/v1/video-stream/", "/videos/", m.streamVideo, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/video-hls/"):
		forward := r.Clone(r.Context())
		forward.Header.Set("X-Mochil-Android-Video", "1")
		forwardAndroidPath(forward, "/api/android/v1/video-hls/", "/api/video-hls/", m.videoHLS, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/video-subtitles/"):
		forwardAndroidPath(r, "/api/android/v1/video-subtitles/", "/api/video-subtitles/", m.videoSubtitle, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/video-metadata-artwork/"):
		forwardAndroidPath(r, "/api/android/v1/video-metadata-artwork/", "/api/video-metadata-artwork/", m.videoMetadataArtworkHTTP, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/video-show-metadata-artwork/"):
		forwardAndroidPath(r, "/api/android/v1/video-show-metadata-artwork/", "/api/video-show-metadata-artwork/", m.videoShowMetadataArtworkHTTP, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/video-progress/"):
		forwardAndroidPath(r, "/api/android/v1/video-progress/", "/api/video-progress/", m.apiVideoProgress, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/podcast-artwork/"):
		forwardAndroidPath(r, "/api/android/v1/podcast-artwork/", "/api/podcast-artwork/", m.podcastArtwork, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/podcast-stream/"):
		forwardAndroidPath(r, "/api/android/v1/podcast-stream/", "/podcast-media/", m.podcastMedia, w)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/podcast-feeds/"):
		forwardAndroidPath(r, "/api/android/v1/podcast-feeds/", "/api/podcast-feeds/", m.apiPodcastFeed, w)
	default:
		http.NotFound(w, r)
	}
}

func (m *multiServer) androidDeviceAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer "))
	account, deviceID, ok := m.androidDevices.authenticate(token)
	if !ok || account != m.readingAccount(r) {
		http.Error(w, "The device token is invalid", http.StatusUnauthorized)
		return
	}
	if err := m.androidDevices.revoke(account, deviceID); err != nil {
		http.Error(w, "Could not disconnect this device", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *multiServer) androidPublication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	remainder := strings.TrimPrefix(r.URL.Path, "/api/android/v1/publications/")
	library, book, ok := m.findBook(remainder)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.URL.Query().Get("text") == "1" && isEPUB(book) && m.writeTextOnlyEPUB(w, r, library, book) {
		return
	}
	forward := r.Clone(r.Context())
	forwardURL := *r.URL
	forwardURL.Path = "/read/" + url.PathEscape(library.ID) + "/" + url.PathEscape(book.ID)
	forward.URL = &forwardURL
	m.readBook(w, forward)
}

// androidPublicationPage serves one OPDS-PSE comic page over the Bearer-gated Android API,
// mirroring web.go's streamPage (session/Basic-only) so a device token can stream manga/comic
// pages without ever needing the session cookie or HTTP Basic auth that gate /stream/{id}.
func (m *multiServer) androidPublicationPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	remainder := strings.TrimPrefix(r.URL.Path, "/api/android/v1/publications/")
	bookPath, pageSegment, ok := strings.Cut(remainder, "/pages/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	library, book, ok := m.findBook(bookPath)
	if !ok {
		http.NotFound(w, r)
		return
	}
	backend, ok := catalog.PageCapability(library.Store)
	if !library.PageStreaming || !ok {
		http.NotFound(w, r)
		return
	}
	index, err := strconv.Atoi(pageSegment)
	if err != nil || index < 0 || index >= book.PageCount {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	page, err := backend.Page(r.Context(), book, index)
	if err != nil {
		http.Error(w, "Page is temporarily unavailable", http.StatusBadGateway)
		return
	}
	defer page.Reader.Close()
	if page.MediaType == "" {
		http.Error(w, "Unsupported page image", http.StatusUnsupportedMediaType)
		return
	}
	w.Header().Set("Content-Type", page.MediaType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if page.Size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(page.Size, 10))
	}
	_, _ = io.Copy(w, page.Reader)
}

func forwardAndroidPath(r *http.Request, from, to string, handler http.HandlerFunc, w http.ResponseWriter) {
	forward := r.Clone(r.Context())
	forwardURL := *r.URL
	forwardURL.Path = to + strings.TrimPrefix(r.URL.Path, from)
	forward.URL = &forwardURL
	handler(w, forward)
}

func (m *multiServer) authorizeAndroid(w http.ResponseWriter, r *http.Request) bool {
	if m.androidDevices == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Android device access is unavailable"})
		return false
	}
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authorization, "Bearer ") {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "A device token is required"})
		return false
	}
	account, _, ok := m.androidDevices.authenticate(strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer ")))
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "The device token is invalid"})
		return false
	}
	identity := accounts.Identity{ID: account, Username: account, Role: accounts.RoleUser}
	if m.accounts != nil {
		var exists bool
		identity, exists = m.accounts.IdentityByID(account)
		if !exists {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "The device account is disabled or unavailable"})
			return false
		}
	}
	*r = *r.WithContext(contextWithIdentity(r.Context(), identity))
	return true
}

// writeTextOnlyEPUB sends a read-along book without its audio, which the app streams instead.
// It reports false when the book has no audio or cannot be read that way, and the whole book is sent.
func (m *multiServer) writeTextOnlyEPUB(w http.ResponseWriter, r *http.Request, library Library, book catalog.Book) bool {
	if library.Store == nil {
		return false
	}
	epub, closeEPUB, err := openEPUBFile(library, book)
	if err != nil {
		return false
	}
	defer closeEPUB()
	if overlay, err := webpub.BuildOverlay(epub.reader, epub.size); err != nil || len(overlay.Chapters) == 0 {
		return false
	}
	w.Header().Set("Content-Type", "application/epub+zip")
	w.Header().Set("Cache-Control", "private, no-store")
	if _, err := webpub.WriteTextOnly(w, epub.reader, epub.size); err != nil {
		panic(http.ErrAbortHandler) // the response has begun; cut the connection so the app sees an incomplete download
	}
	return true
}
