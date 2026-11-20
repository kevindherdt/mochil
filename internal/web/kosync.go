package web

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"mochil/internal/statefile"
)

const koSyncMediaType = "application/vnd.koreader.v1+json"
const maximumKoSyncStateBytes int64 = 64 << 20

var koSyncDocumentID = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)

type koSyncProgress struct {
	Document   string  `json:"document"`
	Progress   string  `json:"progress"`
	Percentage float64 `json:"percentage"`
	Device     string  `json:"device"`
	DeviceID   string  `json:"device_id"`
	Timestamp  int64   `json:"timestamp"`
	// SeenAt is when this server received the save, in its own clock. The device stamp above cannot be
	// used to judge how recent a position is: it is whatever that device's clock says.
	SeenAt   int64          `json:"seen_at,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type koSyncUser struct {
	Salt      string                    `json:"salt"`
	KeyHash   string                    `json:"key_hash"`
	Progress  map[string]koSyncProgress `json:"progress"`
	CreatedAt int64                     `json:"created_at"`
}

type koSyncData struct {
	Version int                    `json:"version"`
	Users   map[string]*koSyncUser `json:"users"`
}

type koSyncStore struct {
	path              string
	allowRegistration bool
	mu                sync.RWMutex
	data              koSyncData
	onProgress        func(string, koSyncProgress)
	// newerProgress may substitute a newer position recorded elsewhere, such
	// as the web reader, for what the client asked about.
	newerProgress func(username, document string, stored koSyncProgress, hasStored bool) (koSyncProgress, bool)

	// Wrong keys are counted per network address and user name, apart from the web login's count: a KOReader that
	// keeps retrying an old password then only blocks itself, and guessing at one user cannot lock out the others.
	failuresMu sync.Mutex
	failures   map[string]loginAttempts
	now        func() time.Time
}

const (
	koSyncFailureLimit  = 20
	koSyncFailureWindow = 15 * time.Minute
	koSyncFailureTable  = 4096
)

func (s *koSyncStore) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// koSyncAttempt keeps the wrong-key count of one address and user. With check it only reports whether the limit is
// used up; otherwise it records the attempt (a correct key clears the count).
func (s *koSyncStore) koSyncAttempt(address, username string, failed, check bool) (blocked bool) {
	key := address + "\x00" + username
	now := s.clock()
	s.failuresMu.Lock()
	defer s.failuresMu.Unlock()
	if s.failures == nil {
		s.failures = map[string]loginAttempts{}
	}
	attempt := s.failures[key]
	expired := attempt.started.IsZero() || now.Sub(attempt.started) >= koSyncFailureWindow
	if check {
		return !expired && attempt.count >= koSyncFailureLimit
	}
	if !failed {
		delete(s.failures, key)
		return false
	}
	if expired {
		if len(s.failures) >= koSyncFailureTable {
			for old, value := range s.failures {
				if now.Sub(value.started) >= koSyncFailureWindow {
					delete(s.failures, old)
				}
			}
			if len(s.failures) >= koSyncFailureTable {
				return false
			}
		}
		attempt = loginAttempts{started: now}
	}
	attempt.count++
	s.failures[key] = attempt
	return false
}

func (s *koSyncStore) setProgress(username string, progress koSyncProgress) (koSyncProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.data.Users[username]
	if user == nil {
		return progress, errors.New("KoSync user does not exist")
	}
	// Every save is stamped by the server, so the stamp only ever describes arrival
	// order. A repeated second or a clock stepping backwards must not make the newer
	// save look older than the stored one - that dropped the save while still
	// answering 200, and the device believed it had synced.
	if current, ok := user.Progress[progress.Document]; ok && current.Timestamp >= progress.Timestamp {
		progress.Timestamp = current.Timestamp + 1
	}
	progress.SeenAt = time.Now().Unix()
	previous, existed := user.Progress[progress.Document]
	user.Progress[progress.Document] = progress
	if err := s.saveLocked(); err != nil {
		if existed {
			user.Progress[progress.Document] = previous
		} else {
			delete(user.Progress, progress.Document)
		}
		return previous, err
	}
	return progress, nil
}

func isKoSyncPath(path string) bool {
	return path == "/healthcheck" || path == "/users/create" || path == "/users/auth" || path == "/syncs/progress" || strings.HasPrefix(path, "/syncs/progress/")
}

func newKoSyncStore(path string, allowRegistration bool) (*koSyncStore, error) {
	store := &koSyncStore{path: path, allowRegistration: allowRegistration, data: koSyncData{Version: 1, Users: map[string]*koSyncUser{}}}
	if path != "" {
		err := statefile.DecodeJSON(path, maximumKoSyncStateBytes, &store.data)
		if err == nil {
			if store.data.Version != 1 {
				return nil, fmt.Errorf("unsupported KoSync database version %d", store.data.Version)
			}
			if store.data.Users == nil {
				store.data.Users = map[string]*koSyncUser{}
			}
			for _, user := range store.data.Users {
				if user == nil {
					return nil, errors.New("KoSync database contains a null user")
				}
				if user.Progress == nil {
					user.Progress = map[string]koSyncProgress{}
				}
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return store, nil
}

func koSyncDerivedKey(salt, key []byte) []byte {
	value := sha256.Sum256(append(append([]byte(nil), salt...), key...))
	for range 120000 {
		value = sha256.Sum256(append(value[:], salt...))
	}
	return value[:]
}

func (s *koSyncStore) setUserLocked(username, key string) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	hash := koSyncDerivedKey(salt, []byte(key))
	progress := map[string]koSyncProgress{}
	created := time.Now().Unix()
	if current := s.data.Users[username]; current != nil {
		progress, created = current.Progress, current.CreatedAt
	}
	previous, existed := s.data.Users[username]
	s.data.Users[username] = &koSyncUser{Salt: base64.RawStdEncoding.EncodeToString(salt), KeyHash: base64.RawStdEncoding.EncodeToString(hash), Progress: progress, CreatedAt: created}
	if err := s.saveLocked(); err != nil {
		if existed {
			s.data.Users[username] = previous
		} else {
			delete(s.data.Users, username)
		}
		return err
	}
	return nil
}

func (s *koSyncStore) ensureUser(username, password string) error {
	key := md5.Sum([]byte(password))
	encoded := hex.EncodeToString(key[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	if user := s.data.Users[username]; user != nil && authenticateKoSyncUser(user, encoded) {
		return nil
	}
	return s.setUserLocked(username, encoded)
}

func authenticateKoSyncUser(user *koSyncUser, key string) bool {
	salt, saltErr := base64.RawStdEncoding.DecodeString(user.Salt)
	want, hashErr := base64.RawStdEncoding.DecodeString(user.KeyHash)
	if saltErr != nil || hashErr != nil {
		return false
	}
	got := koSyncDerivedKey(salt, []byte(key))
	return len(got) == len(want) && subtle.ConstantTimeCompare(got, want) == 1
}

func (s *koSyncStore) authenticate(r *http.Request) (string, bool) {
	username, key := strings.TrimSpace(r.Header.Get("x-auth-user")), strings.TrimSpace(r.Header.Get("x-auth-key"))
	if username == "" || key == "" {
		return "", false
	}
	address := requestAddress(r)
	if s.koSyncAttempt(address, username, false, true) {
		return username, false
	}
	s.mu.RLock()
	user := s.data.Users[username]
	s.mu.RUnlock()
	ok := user != nil && authenticateKoSyncUser(user, key)
	s.koSyncAttempt(address, username, !ok, false)
	return username, ok
}

func (s *koSyncStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".kosync-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s.data); err != nil {
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

func koSyncJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", koSyncMediaType)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *koSyncStore) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/healthcheck" && r.Method == http.MethodGet:
		koSyncJSON(w, http.StatusOK, map[string]string{"state": "OK"})
	case r.URL.Path == "/users/create" && r.Method == http.MethodPost:
		s.createUser(w, r)
	case r.URL.Path == "/users/auth" && r.Method == http.MethodGet:
		if _, ok := s.authenticate(r); !ok {
			koSyncJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
			return
		}
		koSyncJSON(w, http.StatusOK, map[string]string{"authorized": "OK"})
	case r.URL.Path == "/syncs/progress" && r.Method == http.MethodPut:
		s.putProgress(w, r)
	case strings.HasPrefix(r.URL.Path, "/syncs/progress/") && r.Method == http.MethodGet:
		s.getProgress(w, r)
	default:
		koSyncJSON(w, http.StatusNotFound, map[string]string{"message": "Not found"})
	}
}

func (s *koSyncStore) createUser(w http.ResponseWriter, r *http.Request) {
	if !s.allowRegistration {
		koSyncJSON(w, http.StatusPaymentRequired, map[string]string{"message": "Registration is disabled; use the Mochil account"})
		return
	}
	var request struct{ Username, Password string }
	if err := decodeKoSyncBody(r, &request); err != nil || strings.TrimSpace(request.Username) == "" || request.Password == "" {
		koSyncJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid username or password"})
		return
	}
	request.Username = strings.TrimSpace(request.Username)
	if len(request.Username) > 128 || len(request.Password) > 256 {
		koSyncJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid username or password"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Users[request.Username] != nil {
		koSyncJSON(w, http.StatusPaymentRequired, map[string]string{"message": "Username is already registered"})
		return
	}
	if err := s.setUserLocked(request.Username, request.Password); err != nil {
		koSyncJSON(w, http.StatusInternalServerError, map[string]string{"message": "Could not create account"})
		return
	}
	koSyncJSON(w, http.StatusCreated, map[string]string{"username": request.Username})
}

func decodeKoSyncBody(r *http.Request, value any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values")
	}
	return nil
}

func (s *koSyncStore) putProgress(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		koSyncJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}
	var progress koSyncProgress
	if err := decodeKoSyncBody(r, &progress); err != nil || !koSyncDocumentID.MatchString(progress.Document) || progress.Progress == "" || math.IsNaN(progress.Percentage) || math.IsInf(progress.Percentage, 0) || progress.Percentage < 0 || progress.Percentage > 1 || len(progress.Progress) > 4096 || len(progress.Device) > 256 || len(progress.DeviceID) > 256 {
		koSyncJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid progress"})
		return
	}
	progress.Document = strings.ToLower(progress.Document)
	progress.Timestamp = time.Now().Unix()
	stored, err := s.setProgress(username, progress)
	if err != nil {
		koSyncJSON(w, http.StatusInternalServerError, map[string]string{"message": "Could not save progress"})
		return
	}
	if s.onProgress != nil {
		s.onProgress(username, stored)
	}
	koSyncJSON(w, http.StatusOK, map[string]any{"document": stored.Document, "timestamp": stored.Timestamp})
}

func (s *koSyncStore) getProgress(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		koSyncJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}
	document := strings.ToLower(strings.TrimPrefix(r.URL.Path, "/syncs/progress/"))
	if !koSyncDocumentID.MatchString(document) {
		koSyncJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid document"})
		return
	}
	s.mu.RLock()
	progress, exists := s.data.Users[username].Progress[document]
	s.mu.RUnlock()
	if s.newerProgress != nil {
		if newer, ok := s.newerProgress(username, document, progress, exists); ok {
			koSyncJSON(w, http.StatusOK, newer)
			return
		}
	}
	if !exists {
		koSyncJSON(w, http.StatusOK, map[string]any{})
		return
	}
	koSyncJSON(w, http.StatusOK, progress)
}

func (s *koSyncStore) progress(username, document string) (koSyncProgress, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user := s.data.Users[username]
	if user == nil {
		return koSyncProgress{}, false
	}
	progress, ok := user.Progress[strings.ToLower(document)]
	return progress, ok
}

// latestProgressAmong returns the newest progress the user has for one of the
// given documents. Offering a "continue on your device" position from another
// book's document would let the reader link the wrong hash to this book.
func (s *koSyncStore) latestProgressAmong(username string, documents []string) (koSyncProgress, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user := s.data.Users[username]
	if user == nil {
		return koSyncProgress{}, false
	}
	var latest koSyncProgress
	found := false
	for _, document := range documents {
		progress, ok := user.Progress[document]
		if ok && (!found || progress.Timestamp > latest.Timestamp) {
			latest, found = progress, true
		}
	}
	return latest, found
}
