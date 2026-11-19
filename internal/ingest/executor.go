package ingest

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PlanEntry maps one immutable file below the incoming root to a new path below
// the managed root. Existing managed files are never valid ingest targets.
type PlanEntry struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
}

// Executor copies verified files from a dedicated incoming root into a
// dedicated managed root. It never modifies or removes a source file.
type Executor struct {
	sourceRoot   string
	targetRoot   string
	journal      *JournalStore
	imported     *ImportedStore
	maxFileBytes int64
	now          func() time.Time
	commit       func(JournalEntry) (bool, error)
	mu           sync.Mutex

	discoveryCacheMu   sync.Mutex
	discoveryCache     *discoveryCache
	discoveryVisitHook func() // test-only: counts WalkDir callback invocations
}

var ErrManualReview = errors.New("managed ingest requires manual review")

func NewExecutor(sourceRoot, targetRoot string, journal *JournalStore, maxFileBytes int64) (*Executor, error) {
	if journal == nil || maxFileBytes < 1 {
		return nil, errors.New("managed ingest requires a journal and a positive file limit")
	}
	source, err := existingDirectoryRoot(sourceRoot)
	if err != nil {
		return nil, fmt.Errorf("incoming root: %w", err)
	}
	if err = os.MkdirAll(targetRoot, managedDirectoryMode); err != nil {
		return nil, fmt.Errorf("managed root: %w", err)
	}
	target, err := existingDirectoryRoot(targetRoot)
	if err != nil {
		return nil, fmt.Errorf("managed root: %w", err)
	}
	if withinRoot(source, target) || withinRoot(target, source) {
		return nil, errors.New("incoming and managed roots must be separate and non-overlapping")
	}
	executor := &Executor{sourceRoot: source, targetRoot: target, journal: journal, imported: newImportedStore(journal.path), maxFileBytes: maxFileBytes, now: time.Now}
	executor.seedImported()
	executor.commit = executor.commitEntry
	return executor, nil
}

// Plan validates and journals a proposed copy without creating any media file.
func (e *Executor) Plan(entries []PlanEntry) (Transaction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(entries) == 0 || len(entries) > maximumEntries {
		return Transaction{}, errors.New("managed ingest plan has an invalid number of entries")
	}
	id, err := newTransactionID()
	if err != nil {
		return Transaction{}, err
	}
	created := e.now().Unix()
	transaction := Transaction{ID: id, State: TransactionPlanned, CreatedAt: created, UpdatedAt: created, Entries: make([]JournalEntry, 0, len(entries))}
	for index, entry := range entries {
		if !safeJournalPath(entry.SourcePath) || !safeJournalPath(entry.TargetPath) || strings.HasPrefix(entry.TargetPath, ".ingest/") {
			return Transaction{}, fmt.Errorf("managed ingest entry %d has an unsafe path", index)
		}
		source, info, resolveErr := e.resolveSource(entry.SourcePath)
		if resolveErr != nil {
			return Transaction{}, fmt.Errorf("managed ingest source %d: %w", index, resolveErr)
		}
		if info.Size() > e.maxFileBytes {
			return Transaction{}, fmt.Errorf("managed ingest source %d exceeds the file limit", index)
		}
		_ = source // Resolution above proves confinement; execution revalidates it.
		target := filepath.Join(e.targetRoot, filepath.FromSlash(entry.TargetPath))
		if _, statErr := os.Lstat(target); statErr == nil {
			return Transaction{}, fmt.Errorf("managed ingest target %d already exists", index)
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return Transaction{}, fmt.Errorf("managed ingest target %d: %w", index, statErr)
		}
		transaction.Entries = append(transaction.Entries, JournalEntry{
			SourcePath:    entry.SourcePath,
			TargetPath:    entry.TargetPath,
			TemporaryPath: filepath.ToSlash(filepath.Join(".ingest", id, fmt.Sprintf("%04d.part", index))),
			Size:          info.Size(),
		})
	}
	journal, err := e.journal.Load()
	if err != nil {
		return Transaction{}, err
	}
	journal.Transactions = journalWithRoom(journal.Transactions, maximumTransactions-1)
	if len(journal.Transactions) >= maximumTransactions {
		return Transaction{}, errors.New("managed ingest journal is full")
	}
	journal.Transactions = append(journal.Transactions, transaction)
	if err = e.journal.Save(journal); err != nil {
		return Transaction{}, err
	}
	return transaction, nil
}

func (e *Executor) Transactions() ([]Transaction, error) {
	journal, err := e.journal.Load()
	if err != nil {
		return nil, err
	}
	transactions := make([]Transaction, len(journal.Transactions))
	for index, transaction := range journal.Transactions {
		transactions[index] = transaction
		transactions[index].Entries = append([]JournalEntry(nil), transaction.Entries...)
	}
	return transactions, nil
}

func (e *Executor) Transaction(id string) (Transaction, error) {
	if !transactionID.MatchString(id) {
		return Transaction{}, errors.New("invalid managed ingest transaction ID")
	}
	transactions, err := e.Transactions()
	if err != nil {
		return Transaction{}, err
	}
	for _, transaction := range transactions {
		if transaction.ID == id {
			return transaction, nil
		}
	}
	return Transaction{}, os.ErrNotExist
}

// Discard closes a plan that has not started copying. No media files exist at
// this point, so the durable rollback records intent without touching storage.
func (e *Executor) Discard(id string) (Transaction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	journal, index, transaction, err := e.loadTransaction(id)
	if err != nil {
		return Transaction{}, err
	}
	if transaction.State != TransactionPlanned {
		return Transaction{}, fmt.Errorf("managed ingest transaction %q is not planned", id)
	}
	for _, transition := range []struct {
		state   TransactionState
		message string
	}{
		{TransactionFailed, "discarded before copying"},
		{TransactionRollingBack, "discarded before copying"},
		{TransactionRolledBack, "discarded before copying"},
	} {
		transaction, err = Transition(transaction, transition.state, e.now(), transition.message)
		if err != nil {
			return Transaction{}, err
		}
		if err = e.saveTransaction(&journal, index, transaction); err != nil {
			return Transaction{}, err
		}
	}
	return transaction, nil
}

// Execute copies, verifies, and commits a planned transaction. A target that
// appears after planning aborts the transaction and is never replaced.
func (e *Executor) Execute(id string) (Transaction, error) {
	return e.ExecuteContext(context.Background(), id)
}

func (e *Executor) ExecuteContext(ctx context.Context, id string) (Transaction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	journal, index, transaction, err := e.loadTransaction(id)
	if err != nil {
		return Transaction{}, err
	}
	if transaction.State != TransactionPlanned {
		return Transaction{}, fmt.Errorf("managed ingest transaction %q is not planned", id)
	}
	transaction, err = Transition(transaction, TransactionCopying, e.now(), "")
	if err != nil {
		return Transaction{}, err
	}
	if err = e.saveTransaction(&journal, index, transaction); err != nil {
		return Transaction{}, err
	}
	return e.executeCopying(ctx, journal, index, transaction)
}

func (e *Executor) executeCopying(ctx context.Context, journal Journal, index int, transaction Transaction) (Transaction, error) {
	var err error
	for entryIndex := range transaction.Entries {
		if err = e.copyEntry(ctx, &transaction.Entries[entryIndex]); err != nil {
			return e.failAndRollback(journal, index, transaction, 0, err)
		}
	}
	transaction, err = Transition(transaction, TransactionVerified, e.now(), "")
	if err != nil {
		return e.failAndRollback(journal, index, transaction, 0, err)
	}
	if err = e.saveTransaction(&journal, index, transaction); err != nil {
		return Transaction{}, err
	}
	return e.executeVerified(journal, index, transaction)
}

func (e *Executor) executeVerified(journal Journal, index int, transaction Transaction) (Transaction, error) {
	var err error
	for _, entry := range transaction.Entries {
		if _, statErr := os.Lstat(e.target(entry.TargetPath)); statErr == nil {
			return e.failAndRollback(journal, index, transaction, 0, errors.New("an ingest target appeared after planning; nothing was overwritten"))
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return e.failAndRollback(journal, index, transaction, 0, statErr)
		}
	}
	transaction, err = Transition(transaction, TransactionCommitting, e.now(), "")
	if err != nil {
		return e.failAndRollback(journal, index, transaction, 0, err)
	}
	if err = e.saveTransaction(&journal, index, transaction); err != nil {
		return Transaction{}, err
	}
	committedEntries := 0
	for _, entry := range transaction.Entries {
		linked, commitErr := e.commit(entry)
		if linked {
			committedEntries++
		}
		if commitErr != nil {
			return e.failAndRollback(journal, index, transaction, committedEntries, commitErr)
		}
	}
	transaction, err = Transition(transaction, TransactionCommitted, e.now(), "")
	if err != nil {
		return Transaction{}, err
	}
	if err = e.saveTransaction(&journal, index, transaction); err != nil {
		return Transaction{}, err
	}
	e.markImported(transaction)
	_ = os.Remove(filepath.Join(e.targetRoot, ".ingest", transaction.ID))
	return transaction, nil
}

// Recover executes only recovery actions whose ownership and digest can be
// proven from the journal. Ambiguous state is returned unchanged for manual
// review.
func (e *Executor) Recover(id string) (Transaction, []RecoveryStep, error) {
	return e.RecoverContext(context.Background(), id)
}

func (e *Executor) RecoverContext(ctx context.Context, id string) (Transaction, []RecoveryStep, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	journal, index, transaction, err := e.loadTransaction(id)
	if err != nil {
		return Transaction{}, nil, err
	}
	observed := e.observe(transaction)
	steps := PlanRecovery(transaction, observed)
	for _, step := range steps {
		if step.Action == RecoveryManualReview {
			return transaction, steps, ErrManualReview
		}
	}
	switch transaction.State {
	case TransactionPlanned:
		transaction, err = Transition(transaction, TransactionCopying, e.now(), "")
		if err == nil {
			err = e.saveTransaction(&journal, index, transaction)
		}
		if err != nil {
			return Transaction{}, steps, err
		}
		transaction, err = e.executeCopying(ctx, journal, index, transaction)
	case TransactionCopying:
		for _, entry := range transaction.Entries {
			_ = os.Remove(e.target(entry.TemporaryPath))
		}
		transaction, err = e.executeCopying(ctx, journal, index, transaction)
	case TransactionVerified:
		transaction, err = e.executeVerified(journal, index, transaction)
	case TransactionCommitting:
		for entryIndex, state := range observed {
			if !state.TargetExists || !state.TargetMatches {
				return transaction, steps, ErrManualReview
			}
			entry := transaction.Entries[entryIndex]
			_ = os.Remove(e.target(entry.TemporaryPath))
			if entry.BackupPath != "" {
				_ = os.Remove(e.target(entry.BackupPath))
			}
		}
		transaction, err = Transition(transaction, TransactionCommitted, e.now(), "")
		if err == nil {
			err = e.saveTransaction(&journal, index, transaction)
			e.markImported(transaction)
		}
	case TransactionCommitted:
		for _, entry := range transaction.Entries {
			_ = os.Remove(e.target(entry.TemporaryPath))
			if entry.BackupPath != "" {
				_ = os.Remove(e.target(entry.BackupPath))
			}
		}
	case TransactionFailed, TransactionRollingBack:
		for _, state := range observed {
			if state.TargetExists || state.BackupExists {
				return transaction, steps, ErrManualReview
			}
		}
		for _, entry := range transaction.Entries {
			_ = os.Remove(e.target(entry.TemporaryPath))
		}
		if transaction.State == TransactionFailed {
			transaction, err = Transition(transaction, TransactionRollingBack, e.now(), transaction.Error)
			if err == nil {
				err = e.saveTransaction(&journal, index, transaction)
			}
		}
		if err == nil {
			transaction, err = Transition(transaction, TransactionRolledBack, e.now(), transaction.Error)
		}
		if err == nil {
			err = e.saveTransaction(&journal, index, transaction)
		}
	case TransactionRolledBack:
	}
	return transaction, steps, err
}

func (e *Executor) failAndRollback(journal Journal, index int, transaction Transaction, committedEntries int, cause error) (Transaction, error) {
	if next, err := Transition(transaction, TransactionRollingBack, e.now(), boundedError(cause)); err == nil {
		transaction = next
		_ = e.saveTransaction(&journal, index, transaction)
	}
	manual := false
	for entryIndex, entry := range transaction.Entries {
		_ = os.Remove(e.target(entry.TemporaryPath))
		if entryIndex >= committedEntries {
			continue
		}
		target := e.target(entry.TargetPath)
		if _, err := os.Lstat(target); err == nil {
			matches, digestErr := fileMatches(target, entry.Size, entry.SHA256)
			if digestErr != nil || !matches {
				manual = true
				continue
			}
			if removeErr := os.Remove(target); removeErr != nil {
				manual = true
			}
		}
	}
	message := boundedError(cause)
	if manual {
		message = boundedError(fmt.Errorf("%w; manual review is required", cause))
	}
	if next, transitionErr := Transition(transaction, TransactionRolledBack, e.now(), message); transitionErr == nil {
		transaction = next
		_ = e.saveTransaction(&journal, index, transaction)
	}
	return transaction, cause
}

func (e *Executor) loadTransaction(id string) (Journal, int, Transaction, error) {
	if !transactionID.MatchString(id) {
		return Journal{}, -1, Transaction{}, errors.New("invalid managed ingest transaction ID")
	}
	journal, err := e.journal.Load()
	if err != nil {
		return Journal{}, -1, Transaction{}, err
	}
	for index, transaction := range journal.Transactions {
		if transaction.ID == id {
			return journal, index, transaction, nil
		}
	}
	return Journal{}, -1, Transaction{}, os.ErrNotExist
}

func (e *Executor) saveTransaction(journal *Journal, index int, transaction Transaction) error {
	journal.Transactions[index] = transaction
	return e.journal.Save(*journal)
}

func newTransactionID() (string, error) {
	var value [12]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "ingest-" + hex.EncodeToString(value[:]), nil
}

func boundedError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) > 500 {
		message = message[:500]
	}
	return message
}

func journalWithRoom(transactions []Transaction, maximumExisting int) []Transaction {
	if len(transactions) <= maximumExisting {
		return transactions
	}
	remove := len(transactions) - maximumExisting
	retained := make([]Transaction, 0, maximumExisting)
	for _, transaction := range transactions {
		terminal := transaction.State == TransactionCommitted || transaction.State == TransactionRolledBack
		if remove > 0 && terminal {
			remove--
			continue
		}
		retained = append(retained, transaction)
	}
	return retained
}

// TargetRoot is the managed folder that verified copies are committed below.
func (e *Executor) TargetRoot() string { return e.targetRoot }

// ForTarget returns an executor that copies from the same incoming folder, with the same limits, into another
// existing folder (a library's own). It keeps its own journal, so each destination recovers independently.
func (e *Executor) ForTarget(targetRoot string) (*Executor, error) {
	info, err := os.Stat(targetRoot)
	if err != nil || !info.IsDir() {
		return nil, errors.New("the destination folder does not exist")
	}
	resolved, err := filepath.EvalSymlinks(targetRoot)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(filepath.Clean(resolved)))
	journal := NewJournalStore(strings.TrimSuffix(e.journal.path, ".json") + "-" + hex.EncodeToString(digest[:6]) + ".json")
	child, err := NewExecutor(e.sourceRoot, targetRoot, journal, e.maxFileBytes)
	if err != nil {
		return nil, err
	}
	child.imported = e.imported
	child.seedImported()
	return child, nil
}

func (e *Executor) markImported(transaction Transaction) {
	paths := make([]string, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		paths = append(paths, entry.SourcePath)
	}
	e.imported.Mark(paths...)
}
