package reading

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"mochil/internal/statefile"
)

// persistInterval bounds how often the background writer may replace the state
// file. Reading progress arrives at the pace of page turns from the web reader,
// Android and KOReader, and every save used to encode and fsync the whole
// history while holding the store's write lock: 1.2 ms per Put at 100
// positions, 51.8 ms at 20,000 and 171 ms at 60,000, with every Get in the same
// window waiting the whole time (/api/books does 120 of them per page).
//
// Saves are now applied in memory and coalesced into at most one write per
// window, outside the lock. Because the mutation is visible to readers before
// it reaches the disk, a failed write can no longer be undone in memory: the
// background writer keeps the state dirty and retries every window, Flush
// reports the failure synchronously at shutdown, and LastError exposes it to
// whoever logs the store's health. Nothing is silently dropped.
const persistInterval = 1500 * time.Millisecond

// persistState is the store's write side: the dirty counter, the single writer
// goroutine and the bookkeeping that keeps a shutdown flush from racing a
// scheduled write.
type persistState struct {
	// generation counts in-memory mutations, written is the newest generation
	// whose state is known to be on disk. Both are atomic so the writer can
	// compare them without taking the lock every reader needs.
	generation atomic.Uint64
	written    atomic.Uint64

	// mu serialises file writes. The synchronous save path takes it while
	// holding the store's write lock, so a flush must never take the store lock
	// while holding it (see flush).
	mu sync.Mutex

	// window is how long saves coalesce before the writer runs; tests shorten
	// it so they do not have to sleep through the real one.
	window time.Duration

	// write replaces writeStateFile in tests that need to block, count, or fail
	// a write; nil means the real atomic save.
	write func(path string, state storeData) error

	startOnce sync.Once
	stopOnce  sync.Once
	started   atomic.Bool
	// reporting remembers whether the writer is currently known to be failing, so
	// the transition is logged once instead of on every retry window.
	reporting atomic.Bool
	stopped   atomic.Bool
	quit      chan struct{}
	wake      chan struct{}
	done      chan struct{}

	errMu   sync.Mutex
	lastErr error
}

// markDirtyLocked records a mutation and schedules the next write. The caller
// holds the store's write lock, so generation and the state it describes move
// together. It never blocks: a pending wake-up already covers every change
// since it was signalled.
func (s *Store) markDirtyLocked() {
	s.persist.generation.Add(1)
	if s.path == "" {
		return
	}
	if s.persist.stopped.Load() {
		// Closed while a request was still in flight: nothing is left to
		// schedule a write, so write now rather than drop the change.
		_ = s.saveLocked()
		return
	}
	s.startWriterLocked()
	select {
	case s.persist.wake <- struct{}{}:
	default:
	}
}

// startWriterLocked launches the single background writer on the first
// mutation. Stores that are only read (and the ones tests open with no path)
// never run a goroutine.
func (s *Store) startWriterLocked() {
	s.persist.startOnce.Do(func() {
		s.persist.quit = make(chan struct{})
		s.persist.wake = make(chan struct{}, 1)
		s.persist.done = make(chan struct{})
		s.persist.started.Store(true)
		go s.writeLoop()
	})
}

// writeLoop waits for the first dirty mutation, lets the window pass so rapid
// saves coalesce, then writes whatever is dirty. A failed write leaves the
// state dirty and is retried after the next window: a transient disk problem
// must not discard a position the reader was already shown as saved.
func (s *Store) writeLoop() {
	defer close(s.persist.done)
	for {
		select {
		case <-s.persist.quit:
			return
		case <-s.persist.wake:
		}
		for {
			timer := time.NewTimer(s.persist.windowOrInterval())
			select {
			case <-s.persist.quit:
				timer.Stop()
				return
			case <-timer.C:
			}
			err := s.flush()
			if err == nil {
				if s.persist.reporting.Swap(false) {
					slog.Info("reading progress is being saved again")
				}
				break
			}
			// Without this the disk could stop accepting progress while every
			// request kept being acknowledged and nothing said so.
			if !s.persist.reporting.Swap(true) {
				slog.Error("reading progress could not be saved; it stays in memory and is retried", "error", err)
			}
		}
	}
}

// Flush persists every acknowledged change synchronously. Shutdown calls it
// before the process exits, and the account purge calls it because a delete
// that is only in memory is not a delete. A failed Flush leaves the state
// dirty (the caller may retry) and returns the write error.
func (s *Store) Flush() error {
	if s.path == "" {
		return nil
	}
	return s.flush()
}

// LastError reports the most recent background write failure, or nil once a
// later write succeeded. Reading progress is acknowledged in memory, so this is
// how a caller learns that the disk has fallen behind.
func (s *Store) LastError() error {
	s.persist.errMu.Lock()
	defer s.persist.errMu.Unlock()
	return s.persist.lastErr
}

// Close stops the background writer and flushes what it had not written yet.
func (s *Store) Close() error {
	if s.persist.started.Load() {
		s.persist.stopped.Store(true)
		s.persist.stopOnce.Do(func() { close(s.persist.quit) })
		<-s.persist.done
	}
	return s.Flush()
}

// flush writes the current state outside the store lock. It is safe to call
// concurrently with itself, with the background writer and with the
// synchronous save path: a single write lock serialises them, and a write that
// is already covered by a newer one is skipped instead of repeated.
func (s *Store) flush() error {
	p := &s.persist
	if p.generation.Load() == p.written.Load() {
		return p.storedError()
	}
	// Snapshot first, then take the write lock: the synchronous save holds the
	// store's write lock while taking that lock, so taking it the other way
	// round here would deadlock against a blocked reader.
	state, target := s.stateForWrite()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.written.Load() >= target {
		return p.storedError()
	}
	if err := s.writeStateFile(state); err != nil {
		p.recordError(err)
		return err
	}
	p.written.Store(target)
	p.recordError(nil)
	return nil
}

// stateForWrite copies the state for the encoder. Stored values are replaced,
// never mutated through a shared pointer, Locators map or KoSyncDocuments slice,
// so copying the maps and history slices by value yields a consistent snapshot
// without deep-cloning every position. Copying the Position values themselves
// also isolates the encoder from Transfer rewriting history entries in place.
func (s *Store) stateForWrite() (storeData, uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copied := storeData{
		Version:  s.data.Version,
		Accounts: make(map[string]map[string]Position, len(s.data.Accounts)),
		History:  make(map[string]map[string][]Position, len(s.data.History)),
	}
	for account, places := range s.data.Accounts {
		positions := make(map[string]Position, len(places))
		for key, position := range places {
			positions[key] = position
		}
		copied.Accounts[account] = positions
	}
	for account, history := range s.data.History {
		entries := make(map[string][]Position, len(history))
		for key, list := range history {
			entries[key] = append([]Position(nil), list...)
		}
		copied.History[account] = entries
	}
	// Read the generation under the same lock: every mutation bumps it while
	// holding the write lock, so it describes exactly the copied state.
	return copied, s.persist.generation.Load()
}

func (s *Store) writeStateFile(state storeData) error {
	if s.persist.write != nil {
		return s.persist.write(s.path, state)
	}
	return writeStateFile(s.path, state)
}

// saveLocked writes the state while the caller holds the store's write lock.
// Deleting a position, purging an account and transferring a library use it
// because they must be on disk before they return -- and because a failed
// write can still be rolled back while no other writer can have seen the
// change. Ordinary progress saves go through the coalescing writer instead.
func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	p := &s.persist
	target := p.generation.Load()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.written.Load() >= target {
		return p.storedError()
	}
	if err := s.writeStateFile(s.data); err != nil {
		p.recordError(err)
		return err
	}
	p.written.Store(target)
	p.recordError(nil)
	return nil
}

func (p *persistState) windowOrInterval() time.Duration {
	if p.window > 0 {
		return p.window
	}
	return persistInterval
}

func (p *persistState) recordError(err error) {
	p.errMu.Lock()
	p.lastErr = err
	p.errMu.Unlock()
}

func (p *persistState) storedError() error {
	p.errMu.Lock()
	defer p.errMu.Unlock()
	return p.lastErr
}

// writeStateFile replaces the state file atomically and durably: a temporary
// file in the same directory, flushed with Sync, then renamed over the target.
// A crash leaves either the old state or the new one, never a truncated file.
func writeStateFile(path string, state storeData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".reading-state-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if err := statefile.EncodeJSON(temporary, maximumStateBytes, state, true); err != nil {
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
	return os.Rename(name, path)
}
