package web

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mochil/internal/statefile"
)

// transferJournalMaximumBytes bounds the journal file. A transfer holds at most
// transferMaximumBooks paths, so even a full journal is far smaller than this;
// the limit only guards against a damaged file.
const transferJournalMaximumBytes int64 = 8 << 20

const transferJournalVersion = 1

// The steps of a transfer, in the order they are applied. Each one is recorded
// in the journal only after it succeeded, so an entry that is still open names
// exactly the work that is missing.
const (
	transferStepFiles    = "files"
	transferStepMetadata = "metadata"
	transferStepColumns  = "columns"
	transferStepReading  = "reading"
)

// transferStepsInOrder is the order in which every transfer applies its steps.
var transferStepsInOrder = []string{transferStepFiles, transferStepMetadata, transferStepColumns, transferStepReading}

// transferJournalEntry is one transfer that has started but has not finished
// every step. It is written before any file or store is touched, so a crash
// between steps can be completed later instead of silently orphaning the books'
// reading places, metadata overrides and custom column values.
type transferJournalEntry struct {
	ID        string            `json:"id"`
	From      string            `json:"from"`
	To        string            `json:"to"`
	Mode      string            `json:"mode"`
	Books     map[string]string `json:"books"` // old book ID -> new book ID
	Paths     []string          `json:"paths"` // paths relative to both library folders
	CreatedAt int64             `json:"created_at"`
	Done      []string          `json:"done,omitempty"`
	// Skipped names the steps reconcile must not replay, with the reason. A
	// step is listed here when replaying it could overwrite later edits.
	Skipped  map[string]string `json:"skipped,omitempty"`
	Failed   string            `json:"failed,omitempty"`
	Failure  string            `json:"failure,omitempty"`
	Attempts int               `json:"attempts,omitempty"`
}

func (entry transferJournalEntry) done(step string) bool {
	for _, name := range entry.Done {
		if name == step {
			return true
		}
	}
	return false
}

type transferJournalData struct {
	Version   int                    `json:"version"`
	Transfers []transferJournalEntry `json:"transfers"`
}

// transferJournal is the durable list of open transfers, stored beside the
// other state files. An empty path keeps it in memory, which is only useful for
// tests and for a server without a state directory.
type transferJournal struct {
	mu      sync.Mutex
	path    string
	entries []transferJournalEntry
}

// openTransferJournal reads the journal, if there is one. A missing file is an
// empty journal, not an error.
func openTransferJournal(path string) (*transferJournal, error) {
	journal := &transferJournal{path: path}
	if path == "" {
		return journal, nil
	}
	var data transferJournalData
	err := statefile.DecodeJSON(path, transferJournalMaximumBytes, &data)
	if errors.Is(err, os.ErrNotExist) {
		return journal, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read library transfer journal: %w", err)
	}
	if data.Version != transferJournalVersion {
		return nil, fmt.Errorf("unsupported library transfer journal version %d", data.Version)
	}
	for _, entry := range data.Transfers {
		if entry.ID == "" || entry.From == "" || entry.To == "" || (entry.Mode != "move" && entry.Mode != "copy") {
			return nil, errors.New("library transfer journal holds an incomplete transfer")
		}
	}
	journal.entries = data.Transfers
	return journal, nil
}

// saveLocked writes the journal atomically: a temporary file in the same
// directory is flushed and then renamed over the real one, so a crash leaves
// either the old list or the new one, never a half-written file.
func (journal *transferJournal) saveLocked() error {
	if journal.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(journal.path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(journal.path), ".library-transfers-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	encoded := transferJournalData{Version: transferJournalVersion, Transfers: journal.entries}
	if err := statefile.EncodeJSON(temporary, transferJournalMaximumBytes, encoded, true); err != nil {
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
	return os.Rename(name, journal.path)
}

// open returns every entry that is still waiting to be completed,
// oldest first. The result is a copy.
func (journal *transferJournal) open() []transferJournalEntry {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	return cloneTransferEntries(journal.entries)
}

// begin persists a new entry. It must be called before the first write of a
// transfer, so a failure after that point can always be replayed.
func (journal *transferJournal) begin(entry transferJournalEntry) (transferJournalEntry, error) {
	if entry.ID == "" {
		entry.ID = newTransferID()
	}
	if entry.CreatedAt == 0 {
		entry.CreatedAt = time.Now().Unix()
	}
	journal.mu.Lock()
	defer journal.mu.Unlock()
	prior := cloneTransferEntries(journal.entries)
	journal.entries = append(journal.entries, entry)
	if err := journal.saveLocked(); err != nil {
		journal.entries = prior
		return transferJournalEntry{}, err
	}
	return entry, nil
}

// update applies change to one entry and persists the result, restoring the
// previous list if the file cannot be written.
func (journal *transferJournal) update(id string, change func(*transferJournalEntry) bool) error {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	index := -1
	for i := range journal.entries {
		if journal.entries[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return nil
	}
	prior := cloneTransferEntries(journal.entries)
	if !change(&journal.entries[index]) {
		return nil
	}
	if err := journal.saveLocked(); err != nil {
		journal.entries = prior
		return err
	}
	return nil
}

// markDone records that one step finished, so reconcile never repeats it.
func (journal *transferJournal) markDone(id, step string) error {
	return journal.update(id, func(entry *transferJournalEntry) bool {
		if entry.done(step) {
			return false
		}
		entry.Done = append(entry.Done, step)
		entry.Failed, entry.Failure = "", ""
		return true
	})
}

// skip records that reconcile must not replay a step, and why.
func (journal *transferJournal) skip(id, step, reason string) error {
	return journal.update(id, func(entry *transferJournalEntry) bool {
		if entry.Skipped == nil {
			entry.Skipped = map[string]string{}
		}
		entry.Skipped[step] = reason
		return true
	})
}

// fail records the step that failed and its error, leaving the entry open.
func (journal *transferJournal) fail(id, step string, failure error) error {
	return journal.update(id, func(entry *transferJournalEntry) bool {
		entry.Failed, entry.Failure = step, failure.Error()
		entry.Attempts++
		return true
	})
}

// finish removes an entry once every step is done.
func (journal *transferJournal) finish(id string) error {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	index := -1
	for i := range journal.entries {
		if journal.entries[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return nil
	}
	prior := journal.entries
	dropped := make([]transferJournalEntry, 0, len(journal.entries)-1)
	dropped = append(dropped, journal.entries[:index]...)
	dropped = append(dropped, journal.entries[index+1:]...)
	journal.entries = dropped
	if err := journal.saveLocked(); err != nil {
		journal.entries = prior
		return err
	}
	return nil
}

func cloneTransferEntries(source []transferJournalEntry) []transferJournalEntry {
	out := make([]transferJournalEntry, 0, len(source))
	for _, entry := range source {
		copied := entry
		copied.Books = make(map[string]string, len(entry.Books))
		for oldID, newID := range entry.Books {
			copied.Books[oldID] = newID
		}
		copied.Paths = append([]string(nil), entry.Paths...)
		copied.Done = append([]string(nil), entry.Done...)
		if entry.Skipped != nil {
			copied.Skipped = make(map[string]string, len(entry.Skipped))
			for step, reason := range entry.Skipped {
				copied.Skipped[step] = reason
			}
		}
		out = append(out, copied)
	}
	return out
}

// newTransferID names one journal entry. The value only has to be unique on
// this server, so random bytes are enough.
func newTransferID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("transfer-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}
