package wiki

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"mochil/internal/statefile"
)

const (
	storeVersion    = 1
	maxPages        = 10000
	maxFolders      = 10000
	maxContentBytes = 2 << 20
	maxHistory      = 50
)

// maxStateBytes is the largest state file the store will read. Writes enforce the
// same limit, so a successful save can never produce a file Open would refuse.
var maxStateBytes int64 = 128 << 20

var (
	ErrNotFound  = errors.New("wiki page not found")
	ErrConflict  = errors.New("wiki page revision conflict")
	ErrInvalid   = errors.New("invalid wiki page")
	ErrRetention = errors.New("wiki page is still within its trash retention period")
)

type Page struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Folder    string     `json:"folder,omitempty"`
	Content   string     `json:"content"`
	Revision  int64      `json:"revision"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type Snapshot struct {
	Revision int64     `json:"revision"`
	Title    string    `json:"title"`
	Folder   string    `json:"folder,omitempty"`
	Content  string    `json:"content"`
	SavedAt  time.Time `json:"saved_at"`
}

type accountData struct {
	Pages   map[string]Page       `json:"pages"`
	History map[string][]Snapshot `json:"history,omitempty"`
	Folders []string              `json:"folders,omitempty"`
}

type storeData struct {
	Version  int                     `json:"version"`
	Accounts map[string]*accountData `json:"accounts"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	data storeData
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, data: storeData{Version: storeVersion, Accounts: map[string]*accountData{}}}
	if path != "" {
		err := statefile.DecodeJSON(path, maxStateBytes, &s.data)
		if err == nil {
			if s.data.Version != storeVersion {
				return nil, fmt.Errorf("unsupported wiki store version %d", s.data.Version)
			}
			if s.data.Accounts == nil {
				s.data.Accounts = map[string]*accountData{}
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("decode wiki store: %w", err)
		}
	}
	for _, account := range s.data.Accounts {
		if account == nil || len(account.Pages) > maxPages || len(account.Folders) > maxFolders {
			return nil, errors.New("wiki store contains invalid account data")
		}
		for id, page := range account.Pages {
			if id == "" || page.ID != id || validate(page.Title, page.Folder, page.Content) != nil || page.Revision < 1 {
				return nil, errors.New("wiki store contains an invalid page")
			}
		}
		for id, history := range account.History {
			if id == "" || len(history) > maxHistory {
				return nil, errors.New("wiki store contains invalid history")
			}
			for _, snapshot := range history {
				if snapshot.Revision < 1 || validate(snapshot.Title, snapshot.Folder, snapshot.Content) != nil {
					return nil, errors.New("wiki store contains an invalid history snapshot")
				}
			}
		}
		for _, folder := range account.Folders {
			if validateFolder(folder) != nil || normalizeFolder(folder) != folder {
				return nil, errors.New("wiki store contains an invalid folder")
			}
		}
		normalize(account)
	}
	return s, nil
}

func normalize(a *accountData) {
	if a.Pages == nil {
		a.Pages = map[string]Page{}
	}
	if a.History == nil {
		a.History = map[string][]Snapshot{}
	}
}

func (s *Store) accountLocked(name string) *accountData {
	a := s.data.Accounts[name]
	if a == nil {
		a = &accountData{}
		normalize(a)
		s.data.Accounts[name] = a
	}
	return a
}

func normalizeFolder(folder string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(strings.ReplaceAll(folder, `\`, "/")), func(r rune) bool { return r == '/' })
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, "/")
}

func validate(title, folder, content string) error {
	cleanFolder := normalizeFolder(folder)
	if strings.TrimSpace(title) == "" || len(title) > 300 || len(content) > maxContentBytes || !utf8.ValidString(title) || !utf8.ValidString(content) || (cleanFolder != "" && validateFolder(cleanFolder) != nil) {
		return ErrInvalid
	}
	return nil
}

func validateFolder(folder string) error {
	cleanFolder := normalizeFolder(folder)
	if cleanFolder == "" || len(cleanFolder) > 600 || !utf8.ValidString(folder) {
		return ErrInvalid
	}
	for _, part := range strings.Split(cleanFolder, "/") {
		if part == "" || part == "." || part == ".." || len(part) > 120 {
			return ErrInvalid
		}
	}
	return nil
}

func (s *Store) CreateFolder(account, folder string) (string, error) {
	folder = normalizeFolder(folder)
	if account == "" || validateFolder(folder) != nil {
		return "", ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.accountLocked(account)
	for _, existing := range a.Folders {
		if strings.EqualFold(existing, folder) {
			return existing, nil
		}
	}
	if len(a.Folders) >= maxFolders {
		return "", ErrInvalid
	}
	a.Folders = append(a.Folders, folder)
	if err := s.saveLocked(); err != nil {
		a.Folders = a.Folders[:len(a.Folders)-1]
		return "", err
	}
	return folder, nil
}

func (s *Store) Create(account, title, content string) (Page, error) {
	return s.CreateInFolder(account, title, "", content)
}

func (s *Store) CreateInFolder(account, title, folder, content string) (Page, error) {
	if account == "" || validate(title, folder, content) != nil {
		return Page{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.accountLocked(account)
	if len(a.Pages) >= maxPages {
		return Page{}, ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return Page{}, err
	}
	now := time.Now().UTC()
	p := Page{ID: id, Title: strings.TrimSpace(title), Folder: normalizeFolder(folder), Content: content, Revision: 1, CreatedAt: now, UpdatedAt: now}
	a.Pages[id] = p
	if err := s.saveLocked(); err != nil {
		delete(a.Pages, id)
		return Page{}, err
	}
	return p, nil
}

func (s *Store) Update(account, id, title, content string, expected int64) (Page, error) {
	p, ok := s.Get(account, id)
	if !ok {
		return Page{}, ErrNotFound
	}
	return s.UpdateInFolder(account, id, title, p.Folder, content, expected)
}

func (s *Store) UpdateInFolder(account, id, title, folder, content string, expected int64) (Page, error) {
	if validate(title, folder, content) != nil || expected < 1 {
		return Page{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.accountLocked(account)
	p, ok := a.Pages[id]
	if !ok {
		return Page{}, ErrNotFound
	}
	if p.DeletedAt != nil {
		return Page{}, ErrNotFound
	}
	if p.Revision != expected {
		return Page{}, ErrConflict
	}
	previous := p
	previousHistory := append([]Snapshot(nil), a.History[id]...)
	s.snapshotLocked(a, p)
	p.Title, p.Folder, p.Content, p.Revision, p.UpdatedAt = strings.TrimSpace(title), normalizeFolder(folder), content, p.Revision+1, time.Now().UTC()
	a.Pages[id] = p
	if err := s.saveLocked(); err != nil {
		a.Pages[id] = previous
		a.History[id] = previousHistory
		return Page{}, err
	}
	return p, nil
}

func (s *Store) Delete(account, id string, expected int64) (Page, error) {
	return s.setDeleted(account, id, expected, true)
}
func (s *Store) Restore(account, id string, expected int64) (Page, error) {
	return s.setDeleted(account, id, expected, false)
}

// Purge permanently removes one page and all of its snapshots only after it
// has remained in trash for the caller's retention period.
func (s *Store) Purge(account, id string, expected int64, retention time.Duration) error {
	if expected < 1 || retention < 0 {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.accountLocked(account)
	p, ok := a.Pages[id]
	if !ok || p.DeletedAt == nil {
		return ErrNotFound
	}
	if p.Revision != expected {
		return ErrConflict
	}
	if time.Since(*p.DeletedAt) < retention {
		return ErrRetention
	}
	history := append([]Snapshot(nil), a.History[id]...)
	delete(a.Pages, id)
	delete(a.History, id)
	if err := s.saveLocked(); err != nil {
		a.Pages[id] = p
		a.History[id] = history
		return err
	}
	return nil
}

func (s *Store) setDeleted(account, id string, expected int64, deleted bool) (Page, error) {
	if expected < 1 {
		return Page{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.accountLocked(account)
	p, ok := a.Pages[id]
	if !ok {
		return Page{}, ErrNotFound
	}
	if p.Revision != expected {
		return Page{}, ErrConflict
	}
	if (p.DeletedAt != nil) == deleted {
		return p, nil
	}
	previous := p
	previousHistory := append([]Snapshot(nil), a.History[id]...)
	s.snapshotLocked(a, p)
	now := time.Now().UTC()
	p.Revision++
	p.UpdatedAt = now
	if deleted {
		p.DeletedAt = &now
	} else {
		p.DeletedAt = nil
	}
	a.Pages[id] = p
	if err := s.saveLocked(); err != nil {
		a.Pages[id] = previous
		a.History[id] = previousHistory
		return Page{}, err
	}
	return p, nil
}

func (s *Store) RestoreVersion(account, id string, revision, expected int64) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.accountLocked(account)
	p, ok := a.Pages[id]
	if !ok {
		return Page{}, ErrNotFound
	}
	if p.Revision != expected {
		return Page{}, ErrConflict
	}
	var chosen *Snapshot
	for i := range a.History[id] {
		if a.History[id][i].Revision == revision {
			chosen = &a.History[id][i]
			break
		}
	}
	if chosen == nil {
		return Page{}, ErrNotFound
	}
	previous := p
	previousHistory := append([]Snapshot(nil), a.History[id]...)
	s.snapshotLocked(a, p)
	p.Title, p.Folder, p.Content, p.Revision, p.UpdatedAt, p.DeletedAt = chosen.Title, chosen.Folder, chosen.Content, p.Revision+1, time.Now().UTC(), nil
	a.Pages[id] = p
	if err := s.saveLocked(); err != nil {
		a.Pages[id] = previous
		a.History[id] = previousHistory
		return Page{}, err
	}
	return p, nil
}

func (s *Store) DeleteAccount(account string) error {
	if account == "" {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, exists := s.data.Accounts[account]
	if !exists {
		return nil
	}
	delete(s.data.Accounts, account)
	if err := s.saveLocked(); err != nil {
		s.data.Accounts[account] = previous
		return err
	}
	return nil
}

func (s *Store) snapshotLocked(a *accountData, p Page) {
	h := append([]Snapshot{{Revision: p.Revision, Title: p.Title, Folder: p.Folder, Content: p.Content, SavedAt: time.Now().UTC()}}, a.History[p.ID]...)
	if len(h) > maxHistory {
		h = h[:maxHistory]
	}
	a.History[p.ID] = h
}
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if int64(len(b)) > maxStateBytes {
		return fmt.Errorf("wiki state would exceed its %d byte limit", maxStateBytes)
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), "wiki-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, s.path)
}
