package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mochil/internal/accounts"
)

// sessionStore keeps "Remember me" sessions across restarts and redeploys. Only
// the SHA-256 of each session token is written (a copy of the file cannot be
// replayed as a cookie), the file is private to the server user, and it is
// ignored when the sign-in credentials have changed since it was written.
type sessionStore struct {
	path        string
	fingerprint string
	mu          sync.Mutex
}

type persistedSession struct {
	Key      string    `json:"key"`
	Created  time.Time `json:"created"`
	Seen     time.Time `json:"seen"`
	ID       string    `json:"id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
}

type sessionFile struct {
	Version     int                `json:"version"`
	Fingerprint string             `json:"fingerprint"`
	Sessions    []persistedSession `json:"sessions"`
}

func newSessionStore(path, username, password string) *sessionStore {
	sum := sha256.Sum256([]byte("mochil-sessions-v1\x00" + username + "\x00" + password))
	return &sessionStore{path: path, fingerprint: hex.EncodeToString(sum[:])}
}

// load returns the remembered sessions that are still within their lifetime.
func (s *sessionStore) load(now time.Time) map[[32]byte]webSession {
	loaded := map[[32]byte]webSession{}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return loaded
	}
	var file sessionFile
	if json.Unmarshal(data, &file) != nil || file.Version != 1 || file.Fingerprint != s.fingerprint {
		return loaded
	}
	for _, record := range file.Sessions {
		raw, err := hex.DecodeString(record.Key)
		if err != nil || len(raw) != 32 {
			continue
		}
		session := webSession{created: record.Created, seen: record.Seen, remember: true,
			identity: accounts.Identity{ID: record.ID, Username: record.Username, Role: record.Role}}
		if session.expired(now) || len(loaded) >= maxSessions {
			continue
		}
		var key [32]byte
		copy(key[:], raw)
		loaded[key] = session
	}
	return loaded
}

// save writes the remembered sessions and reports a failed write. Most callers can only carry on (sessions are then forgotten
// at the next restart), but revoking an account's sessions must know whether the removal reached the disk.
func (s *sessionStore) save(sessions map[[32]byte]webSession) error {
	file := sessionFile{Version: 1, Fingerprint: s.fingerprint}
	for key, session := range sessions {
		if !session.remember {
			continue
		}
		file.Sessions = append(file.Sessions, persistedSession{Key: hex.EncodeToString(key[:]), Created: session.created, Seen: session.seen,
			ID: session.identity.ID, Username: session.identity.Username, Role: session.identity.Role})
	}
	data, err := json.Marshal(file)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".sessions-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	_, writeErr := temporary.Write(data)
	closeErr := temporary.Close()
	if err := errors.Join(writeErr, closeErr, os.Chmod(name, 0o600)); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, s.path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}
