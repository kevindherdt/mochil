package web

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"mochil/internal/enrichment"
	"mochil/internal/reading"
)

type hardcoverJob struct {
	account string
	event   reading.Event
	library string
	book    string
	// progress is set for a job that sends how far the reader is, instead of started or finished.
	progress   float64
	isProgress bool
	// isRating is set for a job that sends the reader's rating of the book, in stars (0 takes it off).
	isRating bool
	stars    float64
}

// startHardcoverSync sends "started" and "finished" to the Hardcover shelf, one at a time in the background.
// It only runs when the module is on and a token is set, and never delays saving a reading position: the
// store's callback just queues the event, and the worker looks the book up.
func (m *multiServer) startHardcoverSync(linksPath string) error {
	links, err := openHardcoverLinks(linksPath)
	if err != nil {
		return err
	}
	m.hardcoverLinks = links
	skippedPath, matchesPath, ignoredPath := "", "", ""
	if linksPath != "" {
		skippedPath = filepath.Join(filepath.Dir(linksPath), "hardcover-skipped.json")
		matchesPath = filepath.Join(filepath.Dir(linksPath), "hardcover-matches.json")
		ignoredPath = filepath.Join(filepath.Dir(linksPath), "hardcover-ignored.json")
	}
	if err := m.hardcoverIgnored.open(ignoredPath); err != nil {
		return err
	}
	if err := m.hardcoverSkipped.open(skippedPath); err != nil {
		return err
	}
	if err := m.hardcoverMatches.open(matchesPath); err != nil {
		return err
	}
	jobs := make(chan hardcoverJob, 64)
	m.hardcoverStop = make(chan struct{})
	slog.Info("Hardcover shelf sync is on")
	// "Reading" repeats with every page turn, so each book is announced once per run (and again after a failure).
	var mu sync.Mutex
	announced := map[string]bool{}
	key := func(job hardcoverJob) string { return job.account + "\x00" + job.library + "\x00" + job.book }
	m.hardcoverEnqueue = func(job hardcoverJob) {
		select {
		case jobs <- job:
		default:
		}
	}
	progressOn := m.moduleOn("hardcover_progress")
	sent := map[string]progressSent{}
	// Where the reader got to since the last progress that was sent, so the final pages of a session are not lost
	// to the throttle: the maintenance tick below sends them once the throttle allows.
	latest := map[string]hardcoverJob{}
	retries := newHardcoverRetries()
	m.reading.Observe(func(account string, event reading.Event, position reading.Position) {
		job := hardcoverJob{account: account, event: event, library: position.LibraryID, book: position.BookID}
		if event == reading.Finished {
			mu.Lock()
			delete(latest, key(job)) // the status job says it all
			mu.Unlock()
		}
		if progressOn && event == reading.Reading {
			now := time.Now()
			mu.Lock()
			previous, seen := sent[key(job)]
			due := progressDue(previous, seen, position.Fraction, now)
			progress := hardcoverJob{account: account, event: event, library: job.library, book: job.book, progress: position.Fraction, isProgress: true}
			if due {
				sent[key(job)] = progressSent{at: now, fraction: position.Fraction}
				delete(latest, key(job))
			} else {
				latest[key(job)] = progress
			}
			mu.Unlock()
			if due {
				retries.settled(progress) // a newer position replaces one still waiting for a retry
				// After the status job below, so a book is on the shelf before its page count is sent.
				defer func() {
					select {
					case jobs <- progress:
					default:
					}
				}()
			}
		}
		if event == reading.Reading {
			mu.Lock()
			seen := announced[key(job)]
			announced[key(job)] = true
			mu.Unlock()
			if seen {
				return
			}
		}
		select {
		case jobs <- job:
		default: // a long backlog means Hardcover is down; dropping beats growing without bound
			mu.Lock()
			delete(announced, key(job))
			mu.Unlock()
		}
	})
	go func() {
		for {
			var job hardcoverJob
			select {
			case <-m.hardcoverStop:
				return
			case job = <-jobs:
			}
			contained("hardcover sync", func() {
				send := m.sendToHardcover
				switch {
				case job.isProgress:
					send = m.sendProgressToHardcover
				case job.isRating:
					send = m.sendRatingToHardcover
				}
				requested, err := send(job)
				if err != nil && !errors.Is(err, enrichment.ErrHardcoverNoMatch) {
					mu.Lock()
					delete(announced, key(job)) // try again with the next page turn
					mu.Unlock()
					retries.failed(job, time.Now())
				} else {
					retries.settled(job)
				}
				if requested {
					time.Sleep(hardcoverTimings().requestGap) // Hardcover allows about sixty requests a minute
				}
			})
		}
	}()
	go contained("hardcover maintenance", func() {
		for {
			select {
			case <-m.hardcoverStop:
				return
			case <-time.After(hardcoverTimings().maintenanceEvery):
			}
			contained("hardcover maintenance tick", func() {
				now := time.Now()
				var ready []hardcoverJob
				mu.Lock()
				for k, job := range latest {
					if progressDue(sent[k], true, job.progress, now) {
						sent[k] = progressSent{at: now, fraction: job.progress}
						delete(latest, k)
						ready = append(ready, job)
					}
				}
				mu.Unlock()
				ready = append(ready, retries.due(now)...)
				for _, job := range ready {
					select {
					case jobs <- job:
					default:
					}
				}
			})
		}
	})
	return nil
}

// stopHardcoverSync retires the worker: both goroutines return, and anything that arrives afterwards is dropped
// instead of queued.
//
// Production never calls it — the worker lives as long as the process. Tests must, because a sync left running
// keeps writing links, skips and matches into the test's own temp directory after the test has returned, and then
// t.TempDir's cleanup fails with "directory not empty". That is not a slow test: it is a worker outliving its owner,
// and the release workflow went red on it.
func (m *multiServer) stopHardcoverSync() {
	m.hardcoverStopOnce.Do(func() {
		if m.hardcoverStop != nil {
			close(m.hardcoverStop)
		}
		m.hardcoverEnqueue = nil
	})
}

// sendToHardcover reports whether it made a request, and how that went.
func (m *multiServer) sendToHardcover(job hardcoverJob) (bool, error) {
	if !m.hardcoverAccount(job.account) {
		return false, nil
	}
	lib, book, ok := m.findBook(job.library + "/" + job.book)
	if !ok || !publicationKind(lib.Kind) || m.hardcoverIgnored.has(lib.ID, book.ID) {
		return false, nil
	}
	// A book the sync put on the shelf and that is gone now was taken off by the reader: leave it off.
	previous, had := m.hardcoverMatches.get(lib.ID, book.ID)
	allowAdd := !onShelfBefore(previous, had)
	status := enrichment.HardcoverReading
	if job.event == reading.Finished {
		status = enrichment.HardcoverRead
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var (
		outcome enrichment.ShelfOutcome
		err     error
		how     = "matched"
	)
	// A Hardcover book the reader linked by hand (or chose in the Metadata suggestions) is known for certain.
	if linked, ok := m.hardcoverLink(lib.ID, book.ID); ok {
		how = "linked"
		outcome.Result, err = m.hardcover.SetStatusByIDAllowing(ctx, linked, status, allowAdd)
		outcome.Book = m.hardcoverBookInfo(ctx, lib.ID, book.ID, linked)
	} else {
		outcome, err = m.hardcover.SetStatusAllowing(ctx, book.Title, book.Authors, status, allowAdd)
	}
	switch {
	case err == nil && outcome.Result == enrichment.ShelfAbsent:
		slog.Info("Hardcover sync: the book was removed from the shelf, so it is left off", "title", book.Title)
		m.hardcoverIgnored.add(ignoredBook{Library: lib.ID, Book: book.ID, Title: book.Title, Authors: append([]string{}, book.Authors...), Reason: ignoredRemoved, At: time.Now().Unix()})
	case err == nil:
		slog.Info("Hardcover sync", "title", book.Title, "wanted", status, "result", string(outcome.Result), "hardcover", outcome.Book.Title)
		m.hardcoverSkipped.remove(lib.ID, book.ID)
		m.hardcoverMatches.put(hardcoverMatch{
			Library: lib.ID, Book: book.ID, Title: book.Title, Authors: append([]string{}, book.Authors...),
			HardcoverID: outcome.Book.ProviderID, HardcoverTitle: outcome.Book.Title, HardcoverAuthors: outcome.Book.Authors, URL: outcome.Book.SourceURL,
			How: how, Result: string(outcome.Result), Doubt: doubtOf(outcome), At: time.Now().Unix(),
		})
	case errors.Is(err, enrichment.ErrHardcoverAmbiguous):
		// Nothing was written: several Hardcover books share this title and author, and the top-ranked one may well be a
		// different book. Left unflagged so the next sync asks again, and the editor shows the books to choose from.
		slog.Info("Hardcover sync: several books match, left for a person to pick", "title", book.Title, "authors", book.Authors, "hardcover", outcome.Book.Title)
	case errors.Is(err, enrichment.ErrHardcoverNoMatch):
		slog.Info("Hardcover sync: no matching book, skipped", "title", book.Title, "authors", book.Authors)
		m.hardcoverSkipped.add(skippedBook{Library: lib.ID, Book: book.ID, Title: book.Title, Authors: append([]string{}, book.Authors...), At: time.Now().Unix()})
	default:
		slog.Warn("Hardcover sync failed", "title", book.Title, "error", err)
	}
	return true, err
}

// hardcoverAccount says whose reading goes to Hardcover. The token belongs to one Hardcover account that an
// administrator connected, so only an administrator's reading (or the single login of a one-user server) is sent.
func (m *multiServer) hardcoverAccount(account string) bool {
	if m.accountStore() == nil {
		return true
	}
	identity, ok := m.accountStore().IdentityByID(account)
	return ok && identity.Administrator()
}

func publicationKind(kind string) bool {
	switch kind {
	case "books", "comics", "manga", "mixed":
		return true
	}
	return false
}

// hardcoverLink returns the Hardcover book id the reader linked by hand, or saved by picking a Hardcover
// suggestion in Metadata.
func (m *multiServer) hardcoverLink(libraryID, bookID string) (string, bool) {
	if m.hardcoverLinks != nil {
		if id, ok := m.hardcoverLinks.get(libraryID, bookID); ok {
			return id, true
		}
	}
	if m.metadataOverrides == nil {
		return "", false
	}
	value, ok := m.metadataOverrides.Get(libraryID, bookID)
	if !ok || value.Provider != "Hardcover" || value.ProviderID == "" {
		return "", false
	}
	return value.ProviderID, true
}

// hardcoverBookInfo describes a linked Hardcover book: what was saved when it was linked, or else a fresh lookup.
func (m *multiServer) hardcoverBookInfo(ctx context.Context, libraryID, bookID, hardcoverID string) enrichment.Suggestion {
	if match, ok := m.hardcoverMatches.get(libraryID, bookID); ok && match.HardcoverID == hardcoverID && match.HardcoverTitle != "" {
		return enrichment.Suggestion{ProviderID: hardcoverID, Title: match.HardcoverTitle, Authors: match.HardcoverAuthors, SourceURL: match.URL}
	}
	if book, err := m.hardcover.BookByID(ctx, hardcoverID); err == nil {
		return book
	}
	return enrichment.Suggestion{ProviderID: hardcoverID}
}

// doubtOf words why a match by name deserves a second look. A book the reader linked by hand is never in doubt.
func doubtOf(outcome enrichment.ShelfOutcome) string {
	if outcome.Ambiguous {
		return "Several Hardcover books have this title and author. The top-ranked one was taken."
	}
	return ""
}
