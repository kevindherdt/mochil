package web

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"mochil/internal/catalog"
	"mochil/internal/reading"
)

type koReaderDeviceStore struct {
	path string
	mu   sync.RWMutex
	data map[string][]string
}

const maxKoReaderPairingsPerAccount = 16

var (
	koReaderFilenamePrefix      = regexp.MustCompile(`(?i)^(?:(?:unknown|\d+(?:\.\d+)?)\s+-\s+)+`)
	koReaderFilenameReleaseTags = regexp.MustCompile(`(?:\s*\[[^\[\]]+\])+\s*$`)
	koReaderVolumePart          = regexp.MustCompile(`(?i)\bvolume\s*0*(\d+)\s*(?:part|pt|\.)\s*0*(\d+)\b`)
)

func koReaderFilenameKey(value string) string {
	name := strings.TrimSuffix(filepath.Base(strings.TrimSpace(value)), filepath.Ext(value))
	name = koReaderFilenamePrefix.ReplaceAllString(name, "")
	name = koReaderFilenameReleaseTags.ReplaceAllString(name, "")
	name = koReaderVolumePart.ReplaceAllString(name, "volume$1part$2")
	var out strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func koReaderFilenameMatches(left, right string) bool {
	left, right = koReaderFilenameKey(left), koReaderFilenameKey(right)
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	// Download managers and metadata tools commonly prepend an author or index.
	// Only accept a substantial complete suffix; the caller rejects ambiguity.
	return len(left) >= 24 && len(right) >= 24 && (strings.HasSuffix(left, right) || strings.HasSuffix(right, left))
}

func koReaderFilenameMatchesBook(filename string, book catalog.Book) bool {
	if koReaderFilenameMatches(filename, book.Path) || koReaderFilenameMatches(filename, book.Title+filepath.Ext(book.Path)) {
		return true
	}
	for _, author := range book.Authors {
		for _, candidate := range []string{
			author + " - " + book.Title,
			book.Title + " - " + author,
		} {
			if koReaderFilenameKey(filename) == koReaderFilenameKey(candidate+filepath.Ext(book.Path)) {
				return true
			}
		}
		if book.Series != "" && book.SeriesIndex != nil && *book.SeriesIndex > 0 && *book.SeriesIndex == float64(int(*book.SeriesIndex)) {
			position := strconv.Itoa(int(*book.SeriesIndex))
			candidate := author + " - " + book.Series + " - Volume " + position + filepath.Ext(book.Path)
			if koReaderFilenameKey(filename) == koReaderFilenameKey(candidate) {
				return true
			}
		}
	}
	return false
}

type koReaderBookMatch struct {
	libraryID string
	bookID    string
}

func preferActiveKoReaderMatch(matches []koReaderBookMatch, active []readingPosition) []koReaderBookMatch {
	if len(matches) < 2 {
		return matches
	}
	activeBooks := make(map[string]struct{}, len(active))
	for _, position := range active {
		activeBooks[position.LibraryID+"\x00"+position.BookID] = struct{}{}
	}
	preferred := make([]koReaderBookMatch, 0, 1)
	for _, match := range matches {
		if _, ok := activeBooks[match.libraryID+"\x00"+match.bookID]; ok {
			preferred = append(preferred, match)
		}
	}
	if len(preferred) == 1 {
		return preferred
	}
	return matches
}

func newKoReaderDeviceStore(path string) (*koReaderDeviceStore, error) {
	s := &koReaderDeviceStore{path: path, data: map[string][]string{}}
	if data, err := os.ReadFile(path); err == nil {
		var stored map[string]json.RawMessage
		if err := json.Unmarshal(data, &stored); err != nil {
			return nil, err
		}
		for account, raw := range stored {
			var hashes []string
			if err := json.Unmarshal(raw, &hashes); err != nil {
				var legacy string
				if legacyErr := json.Unmarshal(raw, &legacy); legacyErr != nil {
					return nil, fmt.Errorf("decode KOReader pairings for %q: %w", account, err)
				}
				hashes = []string{legacy}
			}
			if len(hashes) > maxKoReaderPairingsPerAccount {
				hashes = hashes[len(hashes)-maxKoReaderPairingsPerAccount:]
			}
			s.data[account] = hashes
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *koReaderDeviceStore) issue(account string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := append([]string(nil), s.data[account]...)
	hashes := append(previous, base64.RawStdEncoding.EncodeToString(hash[:]))
	if len(hashes) > maxKoReaderPairingsPerAccount {
		hashes = hashes[len(hashes)-maxKoReaderPairingsPerAccount:]
	}
	s.data[account] = hashes
	err := s.saveLocked()
	if err != nil {
		s.data[account] = previous
	}
	return token, err
}

func (s *koReaderDeviceStore) revokeAccount(account string) error {
	if account == "" {
		return errors.New("invalid account")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, exists := s.data[account]
	if !exists {
		return nil
	}
	delete(s.data, account)
	if err := s.saveLocked(); err != nil {
		s.data[account] = previous
		return err
	}
	return nil
}

func (s *koReaderDeviceStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".koreader-devices-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(data)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, s.path)
	}
	return err
}

func (s *koReaderDeviceStore) authenticate(token string) (string, bool) {
	hash := sha256.Sum256([]byte(token))
	s.mu.RLock()
	defer s.mu.RUnlock()
	for account, pairings := range s.data {
		for _, encoded := range pairings {
			want, err := base64.RawStdEncoding.DecodeString(encoded)
			if err == nil && len(want) == len(hash) && subtle.ConstantTimeCompare(want, hash[:]) == 1 {
				return account, true
			}
		}
	}
	return "", false
}

func (m *multiServer) koReaderPlugin(w http.ResponseWriter, r *http.Request) {
	if m.devices == nil || m.reading == nil || m.kosync == nil || m.cfg.PublicURL == "" {
		http.Error(w, "The Mochil plugin requires built-in sync and public_url", http.StatusNotImplemented)
		return
	}
	if r.URL.Path == "/api/koreader/plugin.zip" && r.Method == http.MethodPost {
		token, err := m.devices.issue(m.readingAccount(r))
		if err != nil {
			http.Error(w, "Could not create plugin token", 500)
			return
		}
		archive, err := buildKoReaderPlugin(strings.TrimRight(m.cfg.PublicURL, "/"), token)
		if err != nil {
			http.Error(w, "Could not package plugin", 500)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="mochil.koplugin.zip"`)
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(archive)
		return
	}
	progressRequest := strings.HasPrefix(r.URL.Path, "/api/koreader/v1/progress/")
	annotationRequest := strings.HasPrefix(r.URL.Path, "/api/koreader/v1/annotations/")
	finishedRequest := strings.HasPrefix(r.URL.Path, "/api/koreader/v1/finished/")
	vaultHTMLRequest := strings.HasPrefix(r.URL.Path, "/api/koreader/v1/vault/") && strings.HasSuffix(r.URL.Path, ".html")
	// The book dialog shows a cover; a device speaks Bearer, so it needs its own route rather than /covers/.
	coverRequest := strings.HasPrefix(r.URL.Path, "/api/koreader/v1/cover/")
	// Parking a book (on hold / dropped / read) is a device action too, not only a browser one.
	statusSetRequest := strings.HasPrefix(r.URL.Path, "/api/koreader/v1/status/")
	// The dialog offers the next volume of a series; the device asks the catalogue rather than guessing.
	seriesRequest := strings.HasPrefix(r.URL.Path, "/api/koreader/v1/series/")
	updateRequest := r.URL.Path == "/api/koreader/v1/plugin/manifest" || r.URL.Path == "/api/koreader/v1/plugin/core.lua"
	if !progressRequest && !annotationRequest && !updateRequest && !finishedRequest && !vaultHTMLRequest && !coverRequest && !statusSetRequest {
		browsing := r.URL.Path == "/api/koreader/v1/browse" || r.URL.Path == "/api/koreader/v1/continue" || r.URL.Path == "/api/koreader/v1/stats" || strings.HasPrefix(r.URL.Path, "/api/koreader/v1/download/") || strings.HasPrefix(r.URL.Path, "/api/koreader/v1/pages/") || strings.HasPrefix(r.URL.Path, "/api/koreader/v1/series/")
		if (r.URL.Path != "/api/koreader/v1/status" && r.URL.Path != "/api/koreader/v1/home" && r.URL.Path != "/api/koreader/v1/wiki.html" && !browsing) || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
	}
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	account, ok := m.devices.authenticate(token)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_device_token"})
		return
	}
	// A token outlives a disabled account otherwise: unlike the Android routes this
	// one never asked the account store, so disabling a reader did not lock its
	// KOReader out of browsing, downloading or writing progress.
	if m.accountStore() != nil {
		if _, exists := m.accountStore().IdentityByID(account); !exists {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "The device account is disabled or unavailable"})
			return
		}
	}
	if updateRequest {
		// Only a paired device may fetch the plugin, and only to replace itself.
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if strings.HasSuffix(r.URL.Path, "/manifest") {
			writeJSON(w, http.StatusOK, koReaderPluginManifest())
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(koReaderCoreLua))
		return
	}
	if r.URL.Path == "/api/koreader/v1/browse" {
		m.serveKoReaderBrowse(w, r, account)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/koreader/v1/download/") {
		m.serveKoReaderDownload(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/koreader/v1/pages/") {
		m.serveKoReaderPage(w, r)
		return
	}
	if r.URL.Path == "/api/koreader/v1/home" {
		m.serveKoReaderHome(w, account)
		return
	}
	if r.URL.Path == "/api/koreader/v1/continue" {
		m.serveKoReaderContinue(w, r, account)
		return
	}
	if r.URL.Path == "/api/koreader/v1/stats" {
		m.serveKoReaderStats(w, r, account)
		return
	}
	if r.URL.Path == "/api/koreader/v1/status" {
		writeJSON(w, http.StatusOK, map[string]any{"connected": true, "account": account, "server": strings.TrimRight(m.cfg.PublicURL, "/")})
		return
	}
	if r.URL.Path == "/api/koreader/v1/wiki.html" {
		if m.wiki == nil {
			http.Error(w, "Notes are unavailable", http.StatusNotImplemented)
			return
		}
		m.koReaderWikiHTML(w, r, account)
		return
	}
	if vaultHTMLRequest {
		libraryID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/koreader/v1/vault/"), ".html")
		m.koReaderVaultHTML(w, r, libraryID)
		return
	}
	if statusSetRequest {
		m.serveKoReaderStatus(w, r, account)
		return
	}
	if seriesRequest {
		m.serveKoReaderSeries(w, r, account)
		return
	}
	if coverRequest {
		// Same handler the browser uses (variants, provider artwork, disk cache); only the
		// authentication in front of it differs.
		inner := r.Clone(r.Context())
		inner.URL.Path = "/covers/" + strings.TrimPrefix(r.URL.Path, "/api/koreader/v1/cover/")
		m.cover(w, inner)
		return
	}
	if finishedRequest {
		m.serveKoReaderFinished(w, r, account)
		return
	}
	if annotationRequest && m.annotations == nil {
		http.Error(w, "Annotation sync is unavailable", http.StatusNotImplemented)
		return
	}
	if annotationRequest && r.URL.Path == "/api/koreader/v1/annotations/export" {
		m.exportAnnotations(w, r, account)
		return
	}
	if annotationRequest && strings.HasSuffix(r.URL.Path, "/notes") {
		digest := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/koreader/v1/annotations/"), "/notes"))
		if !koSyncDocumentID.MatchString(digest) {
			http.Error(w, "Invalid document", 400)
			return
		}
		position, exists := m.readingSync.FindDocument(account, digest)
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "book_not_linked"})
			return
		}
		m.annotationsToNotes(w, r, account, position.LibraryID, position.BookID)
		return
	}
	prefix := "/api/koreader/v1/progress/"
	if annotationRequest {
		prefix = "/api/koreader/v1/annotations/"
	}
	document := strings.ToLower(strings.TrimPrefix(r.URL.Path, prefix))
	if !koSyncDocumentID.MatchString(document) {
		http.Error(w, "Invalid document", 400)
		return
	}
	position, exists := m.readingSync.FindDocument(account, document)
	if annotationRequest {
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "book_not_linked"})
			return
		}
		m.serveAnnotations(w, r, account, position.LibraryID, position.BookID)
		return
	}
	if !exists && m.publications != nil {
		for _, candidate := range m.reading.List(account, false) {
			lib, book, found := m.findBook(candidate.LibraryID + "/" + candidate.BookID)
			if !found {
				continue
			}
			if digest, ready := publicationDigest(m.publications, lib.ID, book); ready && digest == document {
				var err error
				position, err = m.readingSync.Link(account, lib.ID, book.ID, document)
				if err != nil {
					http.Error(w, "Could not link book", http.StatusInternalServerError)
					return
				}
				exists = true
				break
			}
		}
	}
	if r.Method == http.MethodGet {
		if m.cfg.DisableKoSyncWebReaderBridge {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "web_reader_bridge_disabled"})
			return
		}
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "book_not_linked"})
			return
		}
		position = reading.DevicePosition(position)
		writeJSON(w, http.StatusOK, map[string]any{"position": position})
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var request struct {
		Progress    string     `json:"progress"`
		Percentage  float64    `json:"percentage"`
		Device      string     `json:"device"`
		DeviceID    string     `json:"device_id"`
		Timestamp   int64      `json:"timestamp"`
		Filename    string     `json:"filename"`
		Title       string     `json:"title"`
		Authors     string     `json:"authors"`
		Identifiers string     `json:"identifiers"`
		Series      string     `json:"series"`
		SeriesIndex *float64   `json:"series_index"`
		TextAnchor  *webAnchor `json:"text_anchor"`
		Finished    bool       `json:"finished"`
	}
	if decodeJSONRequest(w, r, 32<<10, &request, false) != nil || request.Progress == "" || request.Percentage < 0 || request.Percentage > 1 || request.Timestamp <= 0 || len(request.Filename) > 1024 || len(request.Title) > 1024 || len(request.Authors) > 2048 || len(request.Identifiers) > 2048 || len(request.Series) > 1024 || (request.SeriesIndex != nil && (math.IsNaN(*request.SeriesIndex) || math.IsInf(*request.SeriesIndex, 0))) || !validWebAnchor(request.TextAnchor) {
		http.Error(w, "Invalid progress", 400)
		return
	}
	if !exists {
		evidence := publicationIdentityEvidence{Title: request.Title, Authors: request.Authors, Identifiers: request.Identifiers, Series: request.Series, SeriesIndex: request.SeriesIndex}
		matches := make([]koReaderBookMatch, 0, 1)
		for _, lib := range m.libraries {
			if lib.Store == nil {
				continue
			}
			for _, book := range lib.Store.Books() {
				if (request.Filename != "" && koReaderFilenameMatchesBook(request.Filename, book)) || publicationIdentityMatchesBook(evidence, book) {
					matches = append(matches, koReaderBookMatch{libraryID: lib.ID, bookID: book.ID})
				}
			}
		}
		matches = preferActiveKoReaderMatch(matches, m.reading.List(account, false))
		if len(matches) == 1 {
			var linkErr error
			position, linkErr = m.readingSync.Link(account, matches[0].libraryID, matches[0].bookID, document)
			if linkErr != nil {
				http.Error(w, "Could not link book", http.StatusInternalServerError)
				return
			}
			exists = true
		}
	}
	if m.kosync != nil {
		// The sync store is keyed by user name; an account's id is a random string. Without a sync user the mirror
		// for stock KOReader clients is skipped, the position itself is recorded below either way.
		if syncAccount := m.koSyncUserFor(account); syncAccount != "" {
			if _, err := m.kosync.setProgress(syncAccount, koSyncProgress{Document: document, Progress: request.Progress, Percentage: request.Percentage, Device: request.Device, DeviceID: request.DeviceID, Timestamp: request.Timestamp}); err != nil {
				http.Error(w, "KoSync account is unavailable", http.StatusConflict)
				return
			}
		}
	}
	if exists && request.TextAnchor == nil {
		request.TextAnchor = m.anchorForKoReader(position.LibraryID, position.BookID, request.Progress)
	}
	if exists {
		var err error
		position, err = m.readingSync.Record(account, position.LibraryID, position.BookID, reading.SyncProgress{
			Document: document, Progress: request.Progress, Percentage: request.Percentage,
			Device: request.Device, DeviceID: request.DeviceID, Timestamp: request.Timestamp,
		}, request.TextAnchor)
		if err != nil {
			http.Error(w, "Could not save progress", http.StatusInternalServerError)
			return
		}
	}
	// "Finished" was chosen on the device; the position stays where the reader left it. Without the flag the usual
	// rule applies (98% or more is read), so a book that is no longer marked complete is being read again.
	if exists && request.Finished && !position.Finished {
		if _, err := m.reading.SetFinished(account, position.LibraryID, position.BookID, true); err != nil {
			http.Error(w, "Could not save progress", http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"saved": exists, "linked": exists})
}

// koSyncUserFor names the sync-store user that mirrors an account's KOReader progress, or "" when it has none.
func (m *multiServer) koSyncUserFor(account string) string {
	// A configured KoSync username is one person's account on whatever server it
	// points at, not a shared key: handing it to every signed-in reader let one
	// account read another's progress and write over it. Only the configured
	// owner may use it; everyone else resolves to their own KoSync user.
	if m.cfg.KoSyncUsername != "" && (m.accountStore() == nil || account == m.cfg.Username) {
		return m.cfg.KoSyncUsername
	}
	name := account
	if m.accountStore() != nil {
		identity, ok := m.accountStore().IdentityByID(account)
		if !ok {
			return ""
		}
		name = identity.Username
	}
	m.kosync.mu.RLock()
	defer m.kosync.mu.RUnlock()
	if m.kosync.data.Users[name] == nil {
		return ""
	}
	return name
}
