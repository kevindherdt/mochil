package web

import "fmt"

// transfers opens the transfer journal once, on first use. An empty state path
// keeps it in memory.
func (m *multiServer) transfers() (*transferJournal, error) {
	m.transferJournalOnce.Do(func() {
		m.transferJournal, m.transferJournalErr = openTransferJournal(m.cfg.TransferJournalState)
	})
	return m.transferJournal, m.transferJournalErr
}

// reconcileTransfers completes every transfer whose journal entry is still
// open: it replays the steps that are not recorded as done, in order, and then
// removes the entry. It runs at startup and with the catalogue rescan, so a
// transfer interrupted by a crash or a failed state write is repaired without a
// person noticing. It is safe to run repeatedly and while requests are served:
// the journal and every store it touches do their own locking, and each step is
// written to be replayed (see the step functions).
func (m *multiServer) reconcileTransfers() {
	journal, err := m.transfers()
	if err != nil {
		m.logTransferProblem("the library transfer journal could not be opened", err)
		return
	}
	if len(journal.open()) == 0 {
		return
	}
	m.metadataApplyMu.Lock()
	defer m.metadataApplyMu.Unlock()
	// Re-read under the transfer lock: a transfer that started between the
	// check above and this lock must not be replayed half-applied.
	for _, entry := range journal.open() {
		from, foundFrom := m.library(entry.From)
		to, foundTo := m.library(entry.To)
		if !foundFrom || !foundTo {
			m.logTransferProblem("a recorded transfer names a library that no longer exists", fmt.Errorf("%s to %s", entry.From, entry.To))
			continue
		}
		step, err := m.replayTransfer(journal, entry, from, to)
		if err != nil {
			// Leave the entry open: it names exactly where the work stopped.
			if recordErr := journal.fail(entry.ID, step, err); recordErr != nil {
				m.logTransferProblem("a failed transfer step could not be recorded", recordErr)
			}
			m.logTransferProblem(fmt.Sprintf("recorded transfer %s stopped at %s", entry.ID, step), err)
			continue
		}
		if err := journal.finish(entry.ID); err != nil {
			m.logTransferProblem("a completed transfer could not be cleared from the journal", err)
			continue
		}
		m.rescanAfterTransfer(entry.From, entry.To)
	}
}

// replayTransfer applies the steps a journal entry has not recorded as done.
// Replayed steps return undo actions that nobody asked for, so they are dropped;
// the point is to leave the books' bookkeeping with their files.
func (m *multiServer) replayTransfer(journal *transferJournal, entry transferJournalEntry, from, to Library) (string, error) {
	for _, step := range m.transferSteps(from, to, entry.Books, entry.Paths, entry.Mode == "move") {
		if entry.done(step.name) {
			continue
		}
		if _, skipped := entry.Skipped[step.name]; skipped {
			// Replaying this step is not safe (see transferSkippedSteps). The
			// transfer finishes with that step left as it was rather than
			// overwriting data that may have changed since.
			continue
		}
		if step.name == transferStepMetadata && m.metadataOverrides == nil {
			// The editor was turned off after the transfer was recorded. There
			// is no store to carry the corrections into, so the entry records
			// the skip instead of pretending the step ran.
			if err := journal.skip(entry.ID, step.name, "the metadata store is not configured"); err != nil {
				return step.name, err
			}
			continue
		}
		if _, err := step.apply(); err != nil {
			return step.name, err
		}
		if err := journal.markDone(entry.ID, step.name); err != nil {
			return step.name, err
		}
	}
	return "", nil
}

// rescanAfterTransfer asks the catalogue scan to pick up the files reconcile
// moved. It runs in the background because the scan callback can be the live
// settings applier, which holds its own lock while a new server is built.
func (m *multiServer) rescanAfterTransfer(libraries ...string) {
	if m.scanLibrary == nil {
		return
	}
	go func() {
		for _, library := range libraries {
			_, _ = m.scanLibrary(library)
		}
	}()
}

func (m *multiServer) logTransferProblem(message string, err error) {
	if m.root != nil && m.root.log != nil {
		m.root.log.Warn("library transfer", "problem", message, "error", err)
	}
}
