package ingest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"mochil/internal/statefile"
)

const (
	journalVersion      = 1
	maximumJournalBytes = 8 << 20
	maximumTransactions = 128
	maximumEntries      = 1000
)

type TransactionState string

const (
	TransactionPlanned     TransactionState = "planned"
	TransactionCopying     TransactionState = "copying"
	TransactionVerified    TransactionState = "verified"
	TransactionCommitting  TransactionState = "committing"
	TransactionCommitted   TransactionState = "committed"
	TransactionRollingBack TransactionState = "rolling_back"
	TransactionRolledBack  TransactionState = "rolled_back"
	TransactionFailed      TransactionState = "failed"
)

type JournalEntry struct {
	SourcePath    string `json:"source_path"`
	TargetPath    string `json:"target_path"`
	TemporaryPath string `json:"temporary_path"`
	BackupPath    string `json:"backup_path,omitempty"`
	Size          int64  `json:"size"`
	SHA256        string `json:"sha256,omitempty"`
}

type Transaction struct {
	ID        string           `json:"id"`
	State     TransactionState `json:"state"`
	CreatedAt int64            `json:"created_at"`
	UpdatedAt int64            `json:"updated_at"`
	Entries   []JournalEntry   `json:"entries"`
	Error     string           `json:"error,omitempty"`
}

type Journal struct {
	Version      int           `json:"version"`
	Transactions []Transaction `json:"transactions"`
}

type JournalStore struct {
	path string
	mu   sync.Mutex
}

func NewJournalStore(path string) *JournalStore { return &JournalStore{path: path} }

func (s *JournalStore) Load() (Journal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return loadJournal(s.path)
}

func (s *JournalStore) Save(journal Journal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateJournal(journal); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".ingest-journal-*.tmp")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = temporary.Chmod(0o600); err == nil {
		err = statefile.EncodeJSON(temporary, maximumJournalBytes, journal, true)
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
	if err := os.Rename(name, s.path); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(s.path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func loadJournal(path string) (Journal, error) {
	var journal Journal
	if err := statefile.DecodeJSON(path, maximumJournalBytes, &journal); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Journal{Version: journalVersion}, nil
		}
		return Journal{}, err
	}
	if err := validateJournal(journal); err != nil {
		return Journal{}, err
	}
	return journal, nil
}

var transactionID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)
var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)

func validateJournal(journal Journal) error {
	if journal.Version != journalVersion {
		return fmt.Errorf("unsupported ingest journal version %d", journal.Version)
	}
	if len(journal.Transactions) > maximumTransactions {
		return errors.New("ingest journal contains too many transactions")
	}
	seen := make(map[string]struct{}, len(journal.Transactions))
	for _, transaction := range journal.Transactions {
		if !transactionID.MatchString(transaction.ID) {
			return errors.New("ingest transaction has an invalid ID")
		}
		if _, exists := seen[transaction.ID]; exists {
			return errors.New("ingest journal contains duplicate transaction IDs")
		}
		seen[transaction.ID] = struct{}{}
		if !validTransactionState(transaction.State) || transaction.CreatedAt <= 0 || transaction.UpdatedAt < transaction.CreatedAt || len(transaction.Entries) == 0 || len(transaction.Entries) > maximumEntries || len(transaction.Error) > 500 {
			return fmt.Errorf("ingest transaction %q is invalid", transaction.ID)
		}
		sources := make(map[string]struct{}, len(transaction.Entries))
		managedPaths := make(map[string]struct{}, len(transaction.Entries)*3)
		for _, entry := range transaction.Entries {
			if !safeJournalPath(entry.SourcePath) || !safeJournalPath(entry.TargetPath) || !safeJournalPath(entry.TemporaryPath) || entry.BackupPath != "" && !safeJournalPath(entry.BackupPath) || entry.Size < 0 || entry.SHA256 != "" && !digest.MatchString(entry.SHA256) {
				return fmt.Errorf("ingest transaction %q contains an invalid entry", transaction.ID)
			}
			if entry.TargetPath == entry.TemporaryPath || entry.TargetPath == entry.BackupPath || entry.TemporaryPath == entry.BackupPath && entry.BackupPath != "" {
				return fmt.Errorf("ingest transaction %q reuses a path", transaction.ID)
			}
			if _, exists := sources[entry.SourcePath]; exists {
				return fmt.Errorf("ingest transaction %q contains a duplicate source", transaction.ID)
			}
			sources[entry.SourcePath] = struct{}{}
			for _, value := range []string{entry.TargetPath, entry.TemporaryPath, entry.BackupPath} {
				if value == "" {
					continue
				}
				if _, exists := managedPaths[value]; exists {
					return fmt.Errorf("ingest transaction %q contains a path collision", transaction.ID)
				}
				managedPaths[value] = struct{}{}
			}
		}
	}
	return nil
}

// ValidManagedPath reports whether a computed target is a path the executor would accept: relative, no backslashes, no
// escapes. Callers that show a target to a person before anything is written need this, because a name that plan will
// refuse must not be presented as the name a book is going to get.
func ValidManagedPath(value string) bool { return safeJournalPath(value) }

func safeJournalPath(value string) bool {
	if value == "" || len(value) > 2048 || filepath.IsAbs(value) || strings.ContainsRune(value, '\x00') || strings.ContainsRune(value, '\\') {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && clean == filepath.ToSlash(value)
}

func validTransactionState(state TransactionState) bool {
	switch state {
	case TransactionPlanned, TransactionCopying, TransactionVerified, TransactionCommitting, TransactionCommitted, TransactionRollingBack, TransactionRolledBack, TransactionFailed:
		return true
	default:
		return false
	}
}

func Transition(transaction Transaction, next TransactionState, now time.Time, message string) (Transaction, error) {
	allowed := map[TransactionState]map[TransactionState]bool{
		TransactionPlanned:     {TransactionCopying: true, TransactionFailed: true},
		TransactionCopying:     {TransactionVerified: true, TransactionRollingBack: true, TransactionFailed: true},
		TransactionVerified:    {TransactionCommitting: true, TransactionRollingBack: true, TransactionFailed: true},
		TransactionCommitting:  {TransactionCommitted: true, TransactionRollingBack: true, TransactionFailed: true},
		TransactionRollingBack: {TransactionRolledBack: true, TransactionFailed: true},
		TransactionFailed:      {TransactionRollingBack: true},
	}
	if !allowed[transaction.State][next] || now.Unix() < transaction.UpdatedAt || len(message) > 500 {
		return Transaction{}, fmt.Errorf("invalid ingest transition from %q to %q", transaction.State, next)
	}
	transaction.State, transaction.UpdatedAt = next, now.Unix()
	transaction.Error = message
	return transaction, nil
}
