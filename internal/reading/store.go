package reading

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"mochil/internal/statefile"
)

const maximumStateBytes int64 = 64 << 20

type SyncProgress struct {
	Document   string
	Progress   string
	Percentage float64
	Device     string
	DeviceID   string
	Timestamp  int64
	// SeenAt is when this server received the position, in server time. Timestamp above is the device's
	// clock and cannot be compared with ours; this one can.
	SeenAt int64
}

type storeData struct {
	Version  int                              `json:"version"`
	Accounts map[string]map[string]Position   `json:"accounts"`
	History  map[string]map[string][]Position `json:"history,omitempty"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	data storeData
	// observer hears about books starting and finishing; see Observe.
	observer func(account string, event Event, position Position)
	// persist coalesces writes in the background; see persist.go.
	persist persistState
}

func Open(path string) (*Store, error) {
	store := &Store{path: path, data: storeData{Version: 1, Accounts: map[string]map[string]Position{}, History: map[string]map[string][]Position{}}}
	store.persist.write = writeStateFile
	if path != "" {
		err := statefile.DecodeJSON(path, maximumStateBytes, &store.data)
		if err == nil {
			if store.data.Version != 1 {
				return nil, fmt.Errorf("unsupported reading-state version %d", store.data.Version)
			}
			if store.data.Accounts == nil {
				store.data.Accounts = map[string]map[string]Position{}
			}
			if store.data.History == nil {
				store.data.History = map[string]map[string][]Position{}
			}
			dropStalePlaces(store.data.Accounts)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return store, nil
}

// staleAnchorGap is how far a text anchor's own place in the book may sit from the
// saved percentage before the anchor is taken to describe somewhere else.
const staleAnchorGap = 0.15

// dropStalePlaces repairs positions saved before a KOReader update replaced the
// percentage but left the old text anchor, CFI and locators behind: the reader
// followed the anchor and opened far from where the percentage said.
func dropStalePlaces(accounts map[string]map[string]Position) {
	for _, positions := range accounts {
		for key, position := range positions {
			if position.WebAnchor != nil && position.WebAnchor.Progress > 0 && math.Abs(position.WebAnchor.Progress-position.Fraction) > staleAnchorGap {
				position.WebAnchor, position.CFI, position.Locators = nil, "", nil
				positions[key] = position
			}
		}
	}
}

func readingKey(libraryID, bookID string) string { return libraryID + "\x00" + bookID }

func (s *Store) appendHistoryLocked(account, key string, position Position) {
	if position.Revision == 0 {
		return
	}
	if s.data.History[account] == nil {
		s.data.History[account] = map[string][]Position{}
	}
	history := s.data.History[account][key]
	if len(history) > 0 {
		latest := history[0]
		if latest.CFI == position.CFI && latest.KoReaderLocator == position.KoReaderLocator && math.Abs(latest.Fraction-position.Fraction) < 1e-9 && latest.UpdatedAt == position.UpdatedAt {
			return
		}
	}
	history = append([]Position{clonePosition(position)}, history...)
	if len(history) > 5 {
		history = history[:5]
	}
	s.data.History[account][key] = history
}

func (s *Store) RestoreHistory(account, libraryID, bookID string, updatedAt int64) (Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := readingKey(libraryID, bookID)
	positions := s.data.Accounts[account]
	current, exists := positions[key]
	if !exists {
		return Position{}, os.ErrNotExist
	}
	for _, snapshot := range s.data.History[account][key] {
		if snapshot.UpdatedAt != updatedAt {
			continue
		}
		s.appendHistoryLocked(account, key, current)
		snapshot = clonePosition(snapshot)
		snapshot.Revision = current.Revision + 1
		snapshot.UpdatedAt = time.Now().UnixMilli()
		snapshot.DeviceID = "history"
		snapshot.DeviceName = "History restore"
		positions[key] = snapshot
		s.appendHistoryLocked(account, key, snapshot)
		s.markDirtyLocked()
		return clonePosition(snapshot), nil
	}
	return Position{}, os.ErrNotExist
}

func (s *Store) SetDismissed(account, libraryID, bookID string, dismissed bool) (Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	positions := s.data.Accounts[account]
	key := readingKey(libraryID, bookID)
	position, exists := positions[key]
	if !exists {
		return Position{}, os.ErrNotExist
	}
	if position.Dismissed == dismissed {
		return clonePosition(position), nil
	}
	position.Dismissed = dismissed
	if !dismissed {
		position.Status = ""
	}
	position.Revision++
	positions[key] = position
	s.markDirtyLocked()
	return clonePosition(position), nil
}

func (s *Store) Put(account string, next Position, baseRevision int64) (Position, error) {
	if account == "" || !ValidUpdate(next) {
		return Position{}, ErrInvalid
	}
	next = clonePosition(next)
	s.mu.Lock()
	defer s.mu.Unlock()
	positions := s.data.Accounts[account]
	key := readingKey(next.LibraryID, next.BookID)
	current := positions[key]
	if current.Revision != baseRevision {
		// A renderer can report a slightly different fraction for the same
		// visible page. A stale client saving that same effective position is
		// idempotent: acknowledge the authoritative value without replacing its
		// newer locators, device attribution, timestamps, or revision.
		if current.Revision > 0 && EquivalentPosition(current, next) {
			return clonePosition(current), nil
		}
		return clonePosition(current), ErrConflict
	}
	// The revision check above is what protects this record; the change is
	// applied to the account and key being touched only.
	positions = s.ensureAccountLocked(account)
	next.Revision = current.Revision + 1
	next.KoSyncSeenAt = current.KoSyncSeenAt
	next.KoSyncDocument = current.KoSyncDocument
	next.KoSyncDocuments = append([]string(nil), current.KoSyncDocuments...)
	if next.KoReaderLocator == "" {
		next.KoReaderLocator = current.KoReaderLocator
	}
	// A client without EPUB CFIs (Android, KOReader, the Readium preview) moves the
	// reader without a new CFI. Keeping the old one would send the web reader back
	// to where the book used to be, so keep it only while the position has not
	// meaningfully moved; the text anchor and fraction place the reader instead.
	if next.CFI == "" && math.Abs(next.Fraction-current.Fraction) <= 0.002 {
		next.CFI = current.CFI
	}
	mergedLocators := make(map[string]string, len(current.Locators)+len(next.Locators))
	for kind, locator := range current.Locators {
		mergedLocators[kind] = locator
	}
	for kind, locator := range next.Locators {
		mergedLocators[kind] = locator
	}
	if len(mergedLocators) > 0 {
		// The incoming map was validated on its own, but the merged map is what
		// gets stored and what history keeps: successive valid writes could
		// otherwise grow one record past the type and byte limits, and eventually
		// past the whole-state limit that every save depends on.
		if !ValidLocators(mergedLocators) {
			return Position{}, ErrInvalid
		}
		next.Locators = mergedLocators
	}
	// Likewise a text anchor describes the place it was taken from.
	if next.WebAnchor == nil && math.Abs(next.Fraction-current.Fraction) <= 0.002 {
		next.WebAnchor = current.WebAnchor
	}
	next.UpdatedAt = time.Now().UnixMilli()
	if next.Fraction >= .98 {
		next.Finished = true
		if current.FinishedAt != 0 {
			next.FinishedAt = current.FinishedAt
		} else {
			next.FinishedAt = next.UpdatedAt
		}
	}
	positions[key] = next
	s.appendHistoryLocked(account, key, next)
	s.markDirtyLocked()
	s.notifyLocked(account, current, next)
	return clonePosition(next), nil
}

func (s *Store) SetFinished(account, libraryID, bookID string, finished bool) (Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	positions := s.ensureAccountLocked(account)
	key := readingKey(libraryID, bookID)
	position := positions[key]
	before := position
	position.LibraryID, position.BookID = libraryID, bookID
	position.Finished = finished
	position.Revision++
	position.UpdatedAt = time.Now().UnixMilli()
	if finished {
		position.FinishedAt = position.UpdatedAt
	} else {
		position.FinishedAt = 0
		if position.Fraction >= .98 {
			position.Fraction = .97
		}
	}
	positions[key] = position
	s.markDirtyLocked()
	s.notifyLocked(account, before, position)
	return clonePosition(position), nil
}

func (s *Store) MarkKoSyncSeen(account, libraryID, bookID string, timestamp int64) (Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	positions := s.ensureAccountLocked(account)
	key := readingKey(libraryID, bookID)
	position := positions[key]
	position.LibraryID, position.BookID = libraryID, bookID
	if timestamp > position.KoSyncSeenAt {
		position.KoSyncSeenAt = timestamp
	}
	position.Revision++
	positions[key] = position
	s.markDirtyLocked()
	return clonePosition(position), nil
}

func (s *Store) AcceptKoSync(account, libraryID, bookID string, timestamp int64, accepted Position) (Position, error) {
	accepted = clonePosition(accepted)
	s.mu.Lock()
	defer s.mu.Unlock()
	positions := s.ensureAccountLocked(account)
	key := readingKey(libraryID, bookID)
	position := clonePosition(positions[key])
	position.LibraryID, position.BookID = libraryID, bookID
	position.CFI, position.Fraction, position.WebAnchor = accepted.CFI, accepted.Fraction, accepted.WebAnchor
	for kind, locator := range accepted.Locators {
		if position.Locators == nil {
			position.Locators = map[string]string{}
		}
		position.Locators[kind] = locator
	}
	if timestamp > position.KoSyncSeenAt {
		position.KoSyncSeenAt = timestamp
	}
	// Keep UpdatedAt unchanged: this is an accepted KOReader location, not a
	// newer web-originated edit that should be echoed back to KOReader.
	position.Revision++
	positions[key] = position
	s.markDirtyLocked()
	return clonePosition(position), nil
}

func (s *Store) LinkKoSync(account, libraryID, bookID, document string) (Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	positions := s.ensureAccountLocked(account)
	key := readingKey(libraryID, bookID)
	position := clonePosition(positions[key])
	position.LibraryID, position.BookID = libraryID, bookID
	position = addKoSyncDocument(position, document)
	position.Revision++
	positions[key] = position
	s.markDirtyLocked()
	return clonePosition(position), nil
}

func (s *Store) RecordKoSync(account, libraryID, bookID string, progress SyncProgress, anchor *Anchor) (Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	positions := s.ensureAccountLocked(account)
	key := readingKey(libraryID, bookID)
	before := positions[key]
	position := clonePosition(before)
	// A push is judged against the last push accepted, never against this server's clock. The two are
	// different clocks: comparing them meant an e-reader running a minute behind had every push
	// discarded, because the moment it reported looked older than the server time stamped on whatever
	// the phone had saved. The reading was lost silently, the phone's position stayed, and the e-reader
	// later pulled that wrong place back. Only a push that would change nothing at all is ignored -- it
	// carries no new information, and accepting it would keep bumping the revision other devices
	// compare against.
	if progress.Timestamp <= before.KoSyncSeenAt &&
		math.Abs(before.Fraction-progress.Percentage) <= 0.002 &&
		before.Finished == (progress.Percentage >= .98) {
		return clonePosition(before), nil
	}
	updatedAt := time.Now().UnixMilli()
	if progress.Timestamp > position.KoSyncSeenAt {
		position.KoSyncSeenAt = progress.Timestamp
	}
	position.LibraryID, position.BookID = libraryID, bookID
	position.Fraction = progress.Percentage
	position.UpdatedAt = updatedAt
	position.DeviceID = progress.DeviceID
	position.DeviceName = progress.Device
	position = addKoSyncDocument(position, progress.Document)
	position.KoReaderLocator = progress.Progress
	// The text anchor, CFI and locators describe wherever the reader was before. When
	// KOReader's position cannot be turned into text (a chapter start, say), keeping them
	// would open the book at that old place while the percentage shows the new one.
	position.WebAnchor, position.CFI, position.Locators = nil, "", nil
	if anchor != nil {
		anchorCopy := *anchor
		position.WebAnchor = &anchorCopy
	}
	position.Dismissed = false
	position.Finished = progress.Percentage >= .98
	if position.Finished {
		position.FinishedAt = updatedAt
	} else {
		position.FinishedAt = 0
	}
	position.Revision++
	positions[key] = position
	s.appendHistoryLocked(account, key, position)
	s.markDirtyLocked()
	s.notifyLocked(account, before, position)
	return clonePosition(position), nil
}

func addKoSyncDocument(position Position, document string) Position {
	if document == "" {
		return position
	}
	if position.KoSyncDocument != "" && position.KoSyncDocument != document && !containsString(position.KoSyncDocuments, position.KoSyncDocument) {
		position.KoSyncDocuments = append(position.KoSyncDocuments, position.KoSyncDocument)
	}
	position.KoSyncDocument = document
	if !containsString(position.KoSyncDocuments, document) {
		position.KoSyncDocuments = append(position.KoSyncDocuments, document)
	}
	if len(position.KoSyncDocuments) > 8 {
		position.KoSyncDocuments = append([]string(nil), position.KoSyncDocuments[len(position.KoSyncDocuments)-8:]...)
	}
	return position
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (s *Store) Delete(account, libraryID, bookID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := readingKey(libraryID, bookID)
	positions := s.data.Accounts[account]
	prior, exists := positions[key]
	if !exists {
		return nil
	}
	delete(positions, key)
	s.markDirtyLocked()
	if err := s.saveLocked(); err != nil {
		positions[key] = prior
		s.markDirtyLocked()
		return err
	}
	return nil
}

func (s *Store) DeleteAccount(account string) error {
	if account == "" {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	priorPositions, hadPositions := s.data.Accounts[account]
	priorHistory, hadHistory := s.data.History[account]
	if !hadPositions && !hadHistory {
		return nil
	}
	delete(s.data.Accounts, account)
	delete(s.data.History, account)
	s.markDirtyLocked()
	if err := s.saveLocked(); err != nil {
		if hadPositions {
			s.data.Accounts[account] = priorPositions
		}
		if hadHistory {
			s.data.History[account] = priorHistory
		}
		s.markDirtyLocked()
		return err
	}
	return nil
}

// Transfer moves everyone's place in books from one library to the same books in another (ids maps old book ID to
// new), so a book moved between libraries is still where each reader left it. The returned action restores the
// previous state.
func (s *Store) Transfer(fromLibrary, toLibrary string, ids map[string]string) (func() error, error) {
	if fromLibrary == "" || toLibrary == "" || fromLibrary == toLibrary {
		return nil, fmt.Errorf("a transfer needs two different libraries")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.snapshotLocked()
	changed := false
	for account, places := range s.data.Accounts {
		for oldID, newID := range ids {
			oldKey := readingKey(fromLibrary, oldID)
			position, ok := places[oldKey]
			if !ok {
				continue
			}
			position.LibraryID, position.BookID = toLibrary, newID
			places[readingKey(toLibrary, newID)] = position
			delete(places, oldKey)
			changed = true
			if history := s.data.History[account]; history != nil {
				if entries, ok := history[oldKey]; ok {
					for i := range entries {
						entries[i].LibraryID, entries[i].BookID = toLibrary, newID
					}
					history[readingKey(toLibrary, newID)] = entries
					delete(history, oldKey)
				}
			}
		}
	}
	if !changed {
		return func() error { return nil }, nil
	}
	s.markDirtyLocked()
	if err := s.saveLocked(); err != nil {
		s.data = previous
		s.markDirtyLocked()
		return nil, err
	}
	// Undoing moves the same books back rather than restoring a snapshot, so progress saved since is kept.
	return func() error {
		inverse := make(map[string]string, len(ids))
		for oldID, newID := range ids {
			inverse[newID] = oldID
		}
		_, err := s.Transfer(toLibrary, fromLibrary, inverse)
		return err
	}, nil
}

// snapshotLocked is a deep copy of the stored positions.
func (s *Store) snapshotLocked() storeData {
	copied := storeData{Version: s.data.Version, Accounts: map[string]map[string]Position{}, History: map[string]map[string][]Position{}}
	for account, places := range s.data.Accounts {
		copied.Accounts[account] = make(map[string]Position, len(places))
		for key, position := range places {
			copied.Accounts[account][key] = position
		}
	}
	for account, history := range s.data.History {
		copied.History[account] = make(map[string][]Position, len(history))
		for key, entries := range history {
			copied.History[account][key] = append([]Position(nil), entries...)
		}
	}
	return copied
}
