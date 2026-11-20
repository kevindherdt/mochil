package annotations

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"mochil/internal/statefile"
)

const maximumStateBytes int64 = 256 << 20

var ErrInvalid = errors.New("invalid annotation update")

type publication struct {
	Revision int64                 `json:"revision"`
	Items    map[string]Annotation `json:"items"`
}

type storeData struct {
	Version  int                               `json:"version"`
	Accounts map[string]map[string]publication `json:"accounts"`
}

type Store struct {
	path string
	mu   sync.RWMutex
	data storeData
}

func Open(path string) (*Store, error) {
	store := &Store{path: path, data: storeData{Version: 1, Accounts: map[string]map[string]publication{}}}
	if path == "" {
		return store, nil
	}
	if err := statefile.DecodeJSON(path, maximumStateBytes, &store.data); err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	if store.data.Version != 1 {
		return nil, fmt.Errorf("unsupported annotation-state version %d", store.data.Version)
	}
	if store.data.Accounts == nil {
		store.data.Accounts = map[string]map[string]publication{}
	}
	return store, nil
}

func publicationKey(libraryID, bookID string) string { return libraryID + "\x00" + bookID }

func (s *Store) List(account, libraryID, bookID string, since int64) Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	publication := s.data.Accounts[account][publicationKey(libraryID, bookID)]
	items := make([]Annotation, 0, len(publication.Items))
	for _, item := range publication.Items {
		if item.Revision > since {
			items = append(items, cloneAnnotation(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Revision == items[j].Revision {
			return items[i].ID < items[j].ID
		}
		return items[i].Revision < items[j].Revision
	})
	return Result{Revision: publication.Revision, Applied: items}
}

// ExportAccount returns a deterministic, bounded JSON export containing only
// one account's annotation data. Tombstones are retained so an import or audit
// can reconstruct the complete synchronization state.
func (s *Store) ExportAccount(account string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	export := Export{Version: 1, Publications: []ExportPublication{}}
	for key, publication := range s.data.Accounts[account] {
		parts := strings.SplitN(key, "\x00", 2)
		if len(parts) != 2 {
			return nil, errors.New("annotation state contains an invalid publication key")
		}
		entry := ExportPublication{
			LibraryID: parts[0],
			BookID:    parts[1],
			Revision:  publication.Revision,
			Items:     make([]Annotation, 0, len(publication.Items)),
		}
		for _, item := range publication.Items {
			entry.Items = append(entry.Items, cloneAnnotation(item))
		}
		sort.Slice(entry.Items, func(i, j int) bool {
			if entry.Items[i].Revision == entry.Items[j].Revision {
				return entry.Items[i].ID < entry.Items[j].ID
			}
			return entry.Items[i].Revision < entry.Items[j].Revision
		})
		export.Publications = append(export.Publications, entry)
	}
	sort.Slice(export.Publications, func(i, j int) bool {
		left, right := export.Publications[i], export.Publications[j]
		if left.LibraryID == right.LibraryID {
			return left.BookID < right.BookID
		}
		return left.LibraryID < right.LibraryID
	})
	var output bytes.Buffer
	if err := statefile.EncodeJSON(&output, maximumStateBytes, export, true); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// DeleteAccount permanently removes all annotation state for one account. It
// is intentionally separate from per-item tombstones: this is the privacy
// control for deleting the server copy, not a synchronization edit.
func (s *Store) DeleteAccount(account string) (int, error) {
	if account == "" {
		return 0, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	books, exists := s.data.Accounts[account]
	if !exists {
		return 0, nil
	}
	count := 0
	for _, publication := range books {
		count += len(publication.Items)
	}
	delete(s.data.Accounts, account)
	if err := s.saveLocked(); err != nil {
		s.data.Accounts[account] = books
		return 0, err
	}
	return count, nil
}

func (s *Store) Apply(account, libraryID, bookID string, changes []Change) (Result, error) {
	if account == "" || libraryID == "" || bookID == "" || len(changes) == 0 || len(changes) > MaximumItemsPerBatch {
		return Result{}, ErrInvalid
	}
	for _, change := range changes {
		if change.BaseRevision < 0 || !Valid(change.Annotation) {
			return Result{}, ErrInvalid
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	books := s.data.Accounts[account]
	rollbackAccount := func() {}
	if books == nil {
		books = map[string]publication{}
		s.data.Accounts[account] = books
		rollbackAccount = func() { delete(s.data.Accounts, account) }
	}
	key := publicationKey(libraryID, bookID)
	// Work on a fresh copy of just this one publication's items (bounded by
	// MaximumItemsPerBook, not by the size of the whole store), so the
	// original map stored at books[key] is never mutated in place and a
	// failed persist or a mid-batch limit can simply be dropped.
	priorPublication, hadPublication := books[key]
	rollbackPublication := func() {
		if hadPublication {
			books[key] = priorPublication
		} else {
			delete(books, key)
		}
		rollbackAccount()
	}
	currentPublication := publication{Revision: priorPublication.Revision, Items: make(map[string]Annotation, len(priorPublication.Items)+1)}
	for id, annotation := range priorPublication.Items {
		currentPublication.Items[id] = annotation
	}
	result := Result{}
	for _, change := range changes {
		current, exists := currentPublication.Items[change.Annotation.ID]
		if !exists && len(currentPublication.Items) >= MaximumItemsPerBook {
			rollbackPublication()
			return Result{}, ErrInvalid
		}
		if current.Revision != change.BaseRevision {
			if exists && SameContent(current, change.Annotation) {
				result.Applied = append(result.Applied, cloneAnnotation(current))
				continue
			}
			result.Conflicts = append(result.Conflicts, Conflict{Submitted: cloneAnnotation(change.Annotation), Current: cloneAnnotation(current)})
			continue
		}
		annotation := cloneAnnotation(change.Annotation)
		currentPublication.Revision++
		annotation.Revision = currentPublication.Revision
		annotation.UpdatedAt = time.Now().UnixMilli()
		if annotation.CreatedAt == 0 {
			if current.CreatedAt != 0 {
				annotation.CreatedAt = current.CreatedAt
			} else {
				annotation.CreatedAt = annotation.UpdatedAt
			}
		}
		currentPublication.Items[annotation.ID] = annotation
		result.Applied = append(result.Applied, cloneAnnotation(annotation))
	}
	books[key] = currentPublication
	result.Revision = currentPublication.Revision
	if err := s.saveLocked(); err != nil {
		rollbackPublication()
		return Result{}, err
	}
	return result, nil
}

// Transfer carries every account's highlights, notes and bookmarks from one
// library's book id to another's, the same way a reading place follows a book
// between libraries. It takes the source entries with them, so a replay after a
// crash finds nothing left to carry and the step is safe to repeat.
//
// A publication already stored under the target key is kept aside and put back
// by the undo, never discarded: the target may hold work on a file that
// happened to have the same id.
func (s *Store) Transfer(fromLibrary, toLibrary string, ids map[string]string) (func() error, error) {
	if fromLibrary == "" || toLibrary == "" || fromLibrary == toLibrary {
		return nil, errors.New("an annotation transfer needs two different libraries")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.snapshotLocked()
	type displaced struct {
		account string
		key     string
		value   publication
	}
	var restored []displaced
	changed := false
	for account, books := range s.data.Accounts {
		for oldID, newID := range ids {
			source := publicationKey(fromLibrary, oldID)
			carried, ok := books[source]
			if !ok {
				continue
			}
			target := publicationKey(toLibrary, newID)
			if existing, found := books[target]; found {
				restored = append(restored, displaced{account: account, key: target, value: existing})
			}
			books[target] = carried
			delete(books, source)
			changed = true
		}
	}
	if !changed {
		return func() error { return nil }, nil
	}
	if err := s.saveLocked(); err != nil {
		s.data = previous
		return nil, err
	}
	// Undoing moves the publications back rather than restoring a snapshot, so
	// highlights written after the move are carried back instead of dropped.
	return func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		before := s.snapshotLocked()
		for _, books := range s.data.Accounts {
			for oldID, newID := range ids {
				source := publicationKey(fromLibrary, oldID)
				target := publicationKey(toLibrary, newID)
				if carried, ok := books[target]; ok {
					books[source] = carried
					delete(books, target)
				}
			}
		}
		for _, entry := range restored {
			if books := s.data.Accounts[entry.account]; books != nil {
				books[entry.key] = entry.value
			}
		}
		if err := s.saveLocked(); err != nil {
			s.data = before
			return err
		}
		return nil
	}, nil
}

// snapshotLocked is a deep copy of the stored publications, for restore after a
// failed write.
func (s *Store) snapshotLocked() storeData {
	copied := storeData{Version: s.data.Version, Accounts: map[string]map[string]publication{}}
	for account, books := range s.data.Accounts {
		copiedBooks := make(map[string]publication, len(books))
		for key, stored := range books {
			items := make(map[string]Annotation, len(stored.Items))
			for id, annotation := range stored.Items {
				items[id] = cloneAnnotation(annotation)
			}
			copiedBooks[key] = publication{Revision: stored.Revision, Items: items}
		}
		copied.Accounts[account] = copiedBooks
	}
	return copied
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".annotation-state-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = temporary.Chmod(0o600); err == nil {
		err = statefile.EncodeJSON(temporary, maximumStateBytes, s.data, false)
	}
	if err == nil {
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

func cloneAnnotation(annotation Annotation) Annotation {
	if annotation.Anchor != nil {
		anchor := *annotation.Anchor
		annotation.Anchor = &anchor
	}
	if annotation.Locators != nil {
		annotation.Locators = make(map[string]string, len(annotation.Locators))
		for kind, locator := range annotation.Locators {
			annotation.Locators[kind] = locator
		}
	}
	return annotation
}
