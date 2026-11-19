package audio

import (
	"errors"
	"math"
	"os"
	"sync"
	"time"

	"mochil/internal/statefile"
)

const maximumProgressStateBytes int64 = 64 << 20

type Progress struct {
	LibraryID    string  `json:"library_id"`
	ItemID       string  `json:"item_id"`
	Position     float64 `json:"position_seconds"`
	Duration     float64 `json:"duration_seconds"`
	UpdatedAt    int64   `json:"updated_at"`
	Revision     uint64  `json:"revision"`
	DeviceID     string  `json:"device_id,omitempty"`
	Finished     bool    `json:"finished,omitempty"`
	Dismissed    bool    `json:"dismissed,omitempty"`
	PlaybackRate float64 `json:"playback_rate,omitempty"`
}

var ErrProgressConflict = errors.New("audio progress changed on another device")

type progressData struct {
	Version  int                            `json:"version"`
	Accounts map[string]map[string]Progress `json:"accounts"`
}
type ProgressStore struct {
	path    string
	mu      sync.RWMutex
	data    progressData
	version uint64
}

func OpenProgressStore(path string) (*ProgressStore, error) {
	s := &ProgressStore{path: path, data: progressData{1, map[string]map[string]Progress{}}, version: 1}
	if path == "" {
		return s, nil
	}
	err := statefile.DecodeJSON(path, maximumProgressStateBytes, &s.data)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if s.data.Version != 1 {
		return nil, errors.New("unsupported audio progress version")
	}
	if s.data.Accounts == nil {
		s.data.Accounts = map[string]map[string]Progress{}
	}
	return s, nil
}
func (s *ProgressStore) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}
func progressKey(library, item string) string { return library + "\x00" + item }
func (s *ProgressStore) Get(account, library, item string) (Progress, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data.Accounts[account][progressKey(library, item)]
	return v, ok
}
func (s *ProgressStore) List(account string) []Progress {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Progress, 0, len(s.data.Accounts[account]))
	for _, v := range s.data.Accounts[account] {
		out = append(out, v)
	}
	return out
}
func (s *ProgressStore) Snapshot(account string) ([]Progress, uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Progress, 0, len(s.data.Accounts[account]))
	for _, value := range s.data.Accounts[account] {
		out = append(out, value)
	}
	return out, s.version
}
func (s *ProgressStore) Put(account string, p Progress) error {
	return s.put(account, p, nil, false)
}

func (s *ProgressStore) PutIfRevision(account string, p Progress, expected uint64) error {
	return s.put(account, p, &expected, false)
}

// PutIfRevisionExplicit permits a deliberate user seek to move playback
// backwards. Passive checkpoints must remain monotonic across devices.
func (s *ProgressStore) PutIfRevisionExplicit(account string, p Progress, expected uint64) error {
	return s.put(account, p, &expected, true)
}

// PatchStatus changes explicit user-owned status without replacing playback
// position written by another device.
func (s *ProgressStore) PatchStatus(account, library, item string, duration float64, finished, dismissed *bool) (Progress, error) {
	if account == "" || library == "" || item == "" || math.IsNaN(duration) || duration < 0 || duration > 30*24*60*60 || finished == nil && dismissed == nil {
		return Progress{}, errors.New("invalid audio progress status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	createdAccount := false
	if s.data.Accounts[account] == nil {
		s.data.Accounts[account] = map[string]Progress{}
		createdAccount = true
	}
	key := progressKey(library, item)
	current, existed := s.data.Accounts[account][key]
	updated := current
	updated.LibraryID, updated.ItemID = library, item
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
	// Retrying an already-applied status must not invalidate active playback's revision.
	if existed && updated == current {
		return current, nil
	}
	updated.UpdatedAt = time.Now().Unix()
	updated.Revision = current.Revision + 1
	s.data.Accounts[account][key] = updated
	if err := s.save(); err != nil {
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
	s.version++
	return updated, nil
}

func (s *ProgressStore) put(account string, p Progress, expected *uint64, explicitSeek bool) error {
	if account == "" || p.LibraryID == "" || p.ItemID == "" || math.IsNaN(p.Position) || math.IsNaN(p.Duration) || p.Position < 0 || p.Duration < 0 || p.Position > p.Duration+5 || p.Duration > 30*24*60*60 {
		return errors.New("invalid audio progress")
	}
	p.UpdatedAt = time.Now().Unix()
	if p.Duration > 0 && p.Position/p.Duration >= .98 {
		p.Finished = true
	}
	if math.IsNaN(p.PlaybackRate) || p.PlaybackRate < .5 || p.PlaybackRate > 4 {
		p.PlaybackRate = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	createdAccount := false
	if s.data.Accounts[account] == nil {
		s.data.Accounts[account] = map[string]Progress{}
		createdAccount = true
	}
	key := progressKey(p.LibraryID, p.ItemID)
	current, existed := s.data.Accounts[account][key]
	if expected != nil && current.Revision != *expected {
		// Media clocks on two clients can differ by a fraction of a second while
		// still describing the same effective playback point. Match the reading
		// store's idempotent conflict behavior: acknowledge that stale checkpoint
		// without replacing newer attribution, status, timestamps, or revision.
		if existed && equivalentAudioProgress(current, p) {
			return nil
		}
		return ErrProgressConflict
	}
	if existed && !explicitSeek && (current.Finished && !p.Finished || p.Position+.25 < current.Position) {
		return ErrProgressConflict
	}
	// Identical checkpoints are acknowledgements, not new activity. Ignore
	// caller timestamps/revisions, but retain every meaningful state change.
	unchanged := p
	unchanged.UpdatedAt, unchanged.Revision = current.UpdatedAt, current.Revision
	if existed && unchanged == current {
		return nil
	}
	p.Revision = current.Revision + 1
	s.data.Accounts[account][key] = p
	if err := s.save(); err != nil {
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
	s.version++
	return nil
}

func equivalentAudioProgress(current, next Progress) bool {
	if current.Finished != next.Finished {
		return false
	}
	difference := current.Position - next.Position
	if difference < 0 {
		difference = -difference
	}
	return difference <= .25
}
func (s *ProgressStore) Delete(account, library, item string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := progressKey(library, item)
	current, exists := s.data.Accounts[account][key]
	if !exists {
		return nil
	}
	delete(s.data.Accounts[account], key)
	if err := s.save(); err != nil {
		s.data.Accounts[account][key] = current
		return err
	}
	s.version++
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
	if err := s.save(); err != nil {
		s.data.Accounts[account] = previous
		return err
	}
	s.version++
	return nil
}
func (s *ProgressStore) save() error {
	return saveAudioState(s.path, ".audio-progress-*", maximumProgressStateBytes, s.data)
}
