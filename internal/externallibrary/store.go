// Package externallibrary stores synchronized membership and progress for
// titles originating outside scanned Mochil libraries.
package externallibrary

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"mochil/internal/statefile"
)

type ExecutionLocation string

const (
	ExecutionServer ExecutionLocation = "server"
	ExecutionDevice ExecutionLocation = "device"
)

type Title struct {
	ID                string            `json:"id"`
	ProviderID        string            `json:"provider_id"`
	SourceID          string            `json:"source_id"`
	RemoteTitleID     string            `json:"remote_title_id"`
	Name              string            `json:"name"`
	Authors           []string          `json:"authors,omitempty"`
	ArtworkURL        string            `json:"artwork_url,omitempty"`
	ExecutionLocation ExecutionLocation `json:"execution_location"`
	AddedAt           int64             `json:"added_at"`
	UpdatedAt         int64             `json:"updated_at"`
}

type Progress struct {
	TitleID         string  `json:"title_id"`
	RemoteChapterID string  `json:"remote_chapter_id"`
	Locator         string  `json:"locator,omitempty"`
	Fraction        float64 `json:"fraction"`
	Revision        int64   `json:"revision"`
	DeviceID        string  `json:"device_id,omitempty"`
	DeviceName      string  `json:"device_name,omitempty"`
	Finished        bool    `json:"finished,omitempty"`
	UpdatedAt       int64   `json:"updated_at"`
}

type fileData struct {
	Version  int                    `json:"version"`
	Accounts map[string]accountData `json:"accounts"`
}
type accountData struct {
	Titles   map[string]Title               `json:"titles"`
	Progress map[string]map[string]Progress `json:"progress"`
}

type Store struct {
	path string
	now  func() time.Time
	mu   sync.RWMutex
	data fileData
}

var ErrConflict = errors.New("external library progress conflict")

const maximumStateBytes int64 = 64 << 20

func Open(path string) (*Store, error) {
	s := &Store{path: path, now: time.Now, data: fileData{Version: 1, Accounts: map[string]accountData{}}}
	if err := statefile.DecodeJSON(path, maximumStateBytes, &s.data); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if s.data.Version != 1 {
		return nil, fmt.Errorf("unsupported external library version %d", s.data.Version)
	}
	if s.data.Accounts == nil {
		s.data.Accounts = map[string]accountData{}
	}
	for account, values := range s.data.Accounts {
		s.data.Accounts[account] = normalizedAccount(values)
	}
	return s, nil
}

func StableID(providerID, sourceID, remoteTitleID string) string {
	sum := sha256.Sum256([]byte(providerID + "\x00" + sourceID + "\x00" + remoteTitleID))
	return base64.RawURLEncoding.EncodeToString(sum[:18])
}

func (s *Store) List(account string) []Title {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := s.data.Accounts[account]
	result := make([]Title, 0, len(values.Titles))
	for _, title := range values.Titles {
		title.Authors = append([]string(nil), title.Authors...)
		result = append(result, title)
	}
	sort.Slice(result, func(i, j int) bool { return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name) })
	return result
}

func (s *Store) Get(account, titleID string) (Title, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	title, ok := s.data.Accounts[account].Titles[titleID]
	title.Authors = append([]string(nil), title.Authors...)
	return title, ok
}

func (s *Store) Put(account string, title Title) (Title, error) {
	account = strings.TrimSpace(account)
	title.ProviderID, title.SourceID, title.RemoteTitleID = strings.TrimSpace(title.ProviderID), strings.TrimSpace(title.SourceID), strings.TrimSpace(title.RemoteTitleID)
	title.Name, title.ArtworkURL = strings.TrimSpace(title.Name), strings.TrimSpace(title.ArtworkURL)
	if account == "" || !validKey(title.ProviderID, 160) || !validKey(title.SourceID, 240) || !validKey(title.RemoteTitleID, 2048) || title.Name == "" || len(title.Name) > 500 || len(title.ArtworkURL) > 4096 || len(title.Authors) > 32 {
		return Title{}, errors.New("external title is invalid")
	}
	if title.ExecutionLocation != ExecutionServer && title.ExecutionLocation != ExecutionDevice {
		return Title{}, errors.New("execution location must be server or device")
	}
	for index := range title.Authors {
		title.Authors[index] = strings.TrimSpace(title.Authors[index])
		if len(title.Authors[index]) > 300 {
			return Title{}, errors.New("external title author is too long")
		}
	}
	title.ID = StableID(title.ProviderID, title.SourceID, title.RemoteTitleID)
	now := s.now().UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	values, rollbackAccount := s.ensureAccountLocked(account)
	priorTitle, hadTitle := values.Titles[title.ID]
	if !hadTitle && len(values.Titles) >= 20_000 {
		rollbackAccount()
		return Title{}, errors.New("external library title limit reached")
	}
	if hadTitle {
		title.AddedAt = priorTitle.AddedAt
	} else {
		title.AddedAt = now
	}
	title.UpdatedAt = now
	values.Titles[title.ID] = title
	if err := s.saveLocked(); err != nil {
		if hadTitle {
			values.Titles[title.ID] = priorTitle
		} else {
			delete(values.Titles, title.ID)
		}
		rollbackAccount()
		return Title{}, err
	}
	return title, nil
}

func (s *Store) Remove(account, titleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	values := normalizedAccount(s.data.Accounts[account])
	priorTitle, ok := values.Titles[titleID]
	if !ok {
		return os.ErrNotExist
	}
	priorProgress, hadProgress := values.Progress[titleID]
	delete(values.Titles, titleID)
	delete(values.Progress, titleID)
	s.data.Accounts[account] = values
	if err := s.saveLocked(); err != nil {
		values.Titles[titleID] = priorTitle
		if hadProgress {
			values.Progress[titleID] = priorProgress
		} else {
			delete(values.Progress, titleID)
		}
		return err
	}
	return nil
}

func (s *Store) DeleteAccount(account string) error {
	if !validKey(account, 128) {
		return errors.New("external account is invalid")
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

func (s *Store) Progress(account, titleID, chapterID string) (Progress, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data.Accounts[account].Progress[titleID][chapterID]
	return value, ok
}

func (s *Store) ListProgress(account, titleID string) []Progress {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chapters := s.data.Accounts[account].Progress[titleID]
	result := make([]Progress, 0, len(chapters))
	for _, value := range chapters {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].UpdatedAt != result[j].UpdatedAt {
			return result[i].UpdatedAt > result[j].UpdatedAt
		}
		return result[i].RemoteChapterID < result[j].RemoteChapterID
	})
	return result
}

func (s *Store) PutProgress(account, titleID string, value Progress, baseRevision int64) (Progress, error) {
	value.RemoteChapterID, value.Locator, value.DeviceID, value.DeviceName = strings.TrimSpace(value.RemoteChapterID), strings.TrimSpace(value.Locator), strings.TrimSpace(value.DeviceID), strings.TrimSpace(value.DeviceName)
	if !validKey(titleID, 64) || !validKey(value.RemoteChapterID, 2048) || len(value.Locator) > 8192 || len(value.DeviceID) > 160 || len(value.DeviceName) > 128 || math.IsNaN(value.Fraction) || math.IsInf(value.Fraction, 0) || value.Fraction < 0 || value.Fraction > 1 {
		return Progress{}, errors.New("external progress is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	values, rollbackAccount := s.ensureAccountLocked(account)
	if _, ok := values.Titles[titleID]; !ok {
		rollbackAccount()
		return Progress{}, os.ErrNotExist
	}
	chapters := values.Progress[titleID]
	createdChapters := chapters == nil
	if createdChapters {
		chapters = map[string]Progress{}
		values.Progress[titleID] = chapters
	}
	current := chapters[value.RemoteChapterID]
	if current.Revision == 0 && len(chapters) >= 10_000 {
		if createdChapters {
			delete(values.Progress, titleID)
		}
		rollbackAccount()
		return Progress{}, errors.New("external title chapter limit reached")
	}
	if current.Revision != baseRevision {
		if equivalentProgress(current, value) {
			if createdChapters {
				delete(values.Progress, titleID)
			}
			rollbackAccount()
			return current, nil
		}
		if createdChapters {
			delete(values.Progress, titleID)
		}
		rollbackAccount()
		return current, ErrConflict
	}
	priorEntry, hadEntry := chapters[value.RemoteChapterID]
	value.TitleID, value.Revision, value.UpdatedAt = titleID, current.Revision+1, s.now().UnixMilli()
	chapters[value.RemoteChapterID] = value
	if err := s.saveLocked(); err != nil {
		if hadEntry {
			chapters[value.RemoteChapterID] = priorEntry
		} else {
			delete(chapters, value.RemoteChapterID)
		}
		if createdChapters {
			delete(values.Progress, titleID)
		}
		rollbackAccount()
		return Progress{}, err
	}
	return value, nil
}

// equivalentProgress acknowledges a stale write that describes the same
// effective EPUB passage. Readium progression changes with viewport and font
// settings, so text context is a stronger identity than the raw fraction.
func equivalentProgress(left, right Progress) bool {
	if left.RemoteChapterID != right.RemoteChapterID {
		return false
	}
	if left.Locator != "" && left.Locator == right.Locator {
		return true
	}
	if left.Locator == "" || right.Locator == "" {
		return math.Abs(left.Fraction-right.Fraction) <= 0.002
	}
	leftLocator, leftOK := readiumTextLocation(left.Locator)
	rightLocator, rightOK := readiumTextLocation(right.Locator)
	return leftOK && rightOK && leftLocator == rightLocator
}

type textLocation struct {
	Href      string
	Before    string
	Highlight string
	After     string
}

func readiumTextLocation(encoded string) (textLocation, bool) {
	var locator struct {
		Href string `json:"href"`
		Text struct {
			Before    string `json:"before"`
			Highlight string `json:"highlight"`
			After     string `json:"after"`
		} `json:"text"`
	}
	if encoded == "" || json.Unmarshal([]byte(encoded), &locator) != nil {
		return textLocation{}, false
	}
	result := textLocation{Href: locator.Href, Before: locator.Text.Before, Highlight: locator.Text.Highlight, After: locator.Text.After}
	// A text fragment without surrounding context is not unique enough; short
	// sentences and headings commonly repeat within a publication.
	valid := result.Href != "" && result.Highlight != "" && (result.Before != "" || result.After != "")
	return result, valid
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(s.path), ".external-library-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if err = file.Chmod(0o600); err == nil {
		err = statefile.EncodeJSON(file, maximumStateBytes, s.data, true)
	}
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, s.path)
	}
	return err
}

func normalizedAccount(value accountData) accountData {
	if value.Titles == nil {
		value.Titles = map[string]Title{}
	}
	if value.Progress == nil {
		value.Progress = map[string]map[string]Progress{}
	}
	return value
}

// ensureAccountLocked returns the account's data, normalizing/creating it if
// absent. The returned rollback (a no-op if nothing was created) removes
// exactly that fresh, still-empty account so a failed persist leaves the
// store exactly as it was -- without copying any other account's data.
func (s *Store) ensureAccountLocked(account string) (values accountData, rollback func()) {
	existing, had := s.data.Accounts[account]
	if had {
		values = normalizedAccount(existing)
		s.data.Accounts[account] = values
		return values, func() {}
	}
	values = normalizedAccount(accountData{})
	s.data.Accounts[account] = values
	return values, func() { delete(s.data.Accounts, account) }
}

func validKey(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && !strings.ContainsRune(value, '\x00')
}
