package video

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mochil/internal/statefile"
)

const maximumProgressStateBytes int64 = 64 << 20

type Progress struct {
	LibraryID string  `json:"library_id"`
	ItemID    string  `json:"item_id"`
	Position  float64 `json:"position_seconds"`
	Duration  float64 `json:"duration_seconds"`
	UpdatedAt int64   `json:"updated_at"`
	Revision  uint64  `json:"revision"`
	DeviceID  string  `json:"device_id,omitempty"`
	Finished  bool    `json:"finished,omitempty"`
	Dismissed bool    `json:"dismissed,omitempty"`
}

var ErrProgressConflict = errors.New("video progress changed on another device")

type progressData struct {
	Version  int                            `json:"version"`
	Accounts map[string]map[string]Progress `json:"accounts"`
}

type ProgressStore struct {
	path string
	mu   sync.RWMutex
	data progressData
}

func OpenProgressStore(path string) (*ProgressStore, error) {
	store := &ProgressStore{path: path, data: progressData{Version: 1, Accounts: map[string]map[string]Progress{}}}
	if path == "" {
		return store, nil
	}
	err := statefile.DecodeJSON(path, maximumProgressStateBytes, &store.data)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	if store.data.Version != 1 {
		return nil, errors.New("unsupported video progress version")
	}
	if store.data.Accounts == nil {
		store.data.Accounts = map[string]map[string]Progress{}
	}
	return store, nil
}

func progressKey(libraryID, itemID string) string { return libraryID + "\x00" + itemID }

func (s *ProgressStore) Get(account, libraryID, itemID string) (Progress, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	progress, ok := s.data.Accounts[account][progressKey(libraryID, itemID)]
	return progress, ok
}

func (s *ProgressStore) List(account string) []Progress {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Progress, 0, len(s.data.Accounts[account]))
	for _, progress := range s.data.Accounts[account] {
		out = append(out, progress)
	}
	return out
}

func (s *ProgressStore) Put(account string, progress Progress) error {
	return s.put(account, progress, nil, false)
}

func (s *ProgressStore) PutIfRevision(account string, progress Progress, expected uint64) error {
	return s.put(account, progress, &expected, false)
}

// PutIfRevisionExplicit permits a deliberate user seek to move playback
// backwards. Passive checkpoints must remain monotonic across devices.
func (s *ProgressStore) PutIfRevisionExplicit(account string, progress Progress, expected uint64) error {
	return s.put(account, progress, &expected, true)
}

// PatchStatus changes explicit user-owned status without replacing playback
// position written by another device.
func (s *ProgressStore) PatchStatus(account, libraryID, itemID string, duration float64, finished, dismissed *bool) (Progress, error) {
	if account == "" || libraryID == "" || itemID == "" || math.IsNaN(duration) || duration < 0 || duration > 7*24*60*60 || finished == nil && dismissed == nil {
		return Progress{}, errors.New("invalid video progress status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	createdAccount := false
	if s.data.Accounts[account] == nil {
		s.data.Accounts[account] = map[string]Progress{}
		createdAccount = true
	}
	key := progressKey(libraryID, itemID)
	current, existed := s.data.Accounts[account][key]
	updated := current
	updated.LibraryID, updated.ItemID = libraryID, itemID
	if updated.Duration <= 0 {
		updated.Duration = duration
	}
	if finished != nil {
		updated.Finished = *finished
		if *finished {
			updated.Position = updated.Duration
		} else if updated.Position >= updated.Duration {
			updated.Position = 0
		}
	}
	if dismissed != nil {
		updated.Dismissed = *dismissed
	}
	// Repeated status actions are acknowledgements, not new playback revisions.
	if existed && updated == current {
		return current, nil
	}
	updated.UpdatedAt = time.Now().Unix()
	updated.Revision = current.Revision + 1
	s.data.Accounts[account][key] = updated
	if err := s.saveLocked(); err != nil {
		if existed {
			s.data.Accounts[account][key] = current
		} else {
			delete(s.data.Accounts[account], key)
		}
		if createdAccount {
			delete(s.data.Accounts, account)
		}
		return Progress{}, err
	}
	return updated, nil
}

func (s *ProgressStore) put(account string, progress Progress, expected *uint64, explicitSeek bool) error {
	if account == "" || progress.LibraryID == "" || progress.ItemID == "" || math.IsNaN(progress.Position) || math.IsNaN(progress.Duration) || progress.Position < 0 || progress.Duration < 0 || progress.Position > progress.Duration+5 || progress.Duration > 7*24*60*60 {
		return errors.New("invalid video progress")
	}
	progress.UpdatedAt = time.Now().Unix()
	if progress.Duration > 0 && progress.Position/progress.Duration >= .95 {
		progress.Finished = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	createdAccount := false
	if s.data.Accounts[account] == nil {
		s.data.Accounts[account] = map[string]Progress{}
		createdAccount = true
	}
	key := progressKey(progress.LibraryID, progress.ItemID)
	current, existed := s.data.Accounts[account][key]
	if expected != nil && current.Revision != *expected {
		// Two playback clocks can report the same effective video position a
		// fraction of a second apart. Acknowledge the stale checkpoint without
		// replacing newer device attribution, status, timestamp, or revision.
		// Materially different positions and completion states still conflict.
		if existed && equivalentVideoProgress(current, progress) {
			return nil
		}
		return ErrProgressConflict
	}
	if existed && !explicitSeek && (current.Finished && !progress.Finished || progress.Position+.25 < current.Position) {
		return ErrProgressConflict
	}
	// Identical checkpoints are acknowledgements, not new activity. Ignore
	// caller timestamps/revisions, but retain every meaningful state change.
	unchanged := progress
	unchanged.UpdatedAt, unchanged.Revision = current.UpdatedAt, current.Revision
	if existed && unchanged == current {
		return nil
	}
	progress.Revision = current.Revision + 1
	s.data.Accounts[account][key] = progress
	if err := s.saveLocked(); err != nil {
		if existed {
			s.data.Accounts[account][key] = current
		} else {
			delete(s.data.Accounts[account], key)
			if createdAccount {
				delete(s.data.Accounts, account)
			}
		}
		return err
	}
	return nil
}

func equivalentVideoProgress(current, next Progress) bool {
	if current.Finished != next.Finished {
		return false
	}
	difference := current.Position - next.Position
	if difference < 0 {
		difference = -difference
	}
	return difference <= .25
}

func (s *ProgressStore) Delete(account, libraryID, itemID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := progressKey(libraryID, itemID)
	current, exists := s.data.Accounts[account][key]
	if !exists {
		return nil
	}
	delete(s.data.Accounts[account], key)
	if err := s.saveLocked(); err != nil {
		s.data.Accounts[account][key] = current
		return err
	}
	return nil
}

func (s *ProgressStore) DeleteAccount(account string) error {
	if account == "" {
		return errors.New("invalid account")
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

func (s *ProgressStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".video-progress-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = statefile.EncodeJSON(temporary, maximumProgressStateBytes, s.data, false); err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, s.path)
}
