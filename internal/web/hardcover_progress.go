package web

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync"
	"time"

	"mochil/internal/enrichment"
)

const hardcoverProgressStep = 0.01 // how far the reader must have moved since the last progress sent

// hardcoverTiming holds every wait in the Hardcover sync. It sits behind a lock so a test can shorten them while
// goroutines of earlier tests still read them.
type hardcoverTiming struct {
	progressEvery    time.Duration   // how often one book's page count may be sent
	maintenanceEvery time.Duration   // how often retries that are due and held-back positions are looked at
	requestGap       time.Duration   // the pause after each request: Hardcover allows about sixty a minute
	retryDelays      []time.Duration // the wait before each new attempt at a failed update
}

var (
	hardcoverTimingMu sync.RWMutex
	hardcoverTimingIs = hardcoverTiming{progressEvery: 2 * time.Minute, maintenanceEvery: 30 * time.Second, requestGap: time.Second,
		retryDelays: []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 3 * time.Hour}}
)

func hardcoverTimings() hardcoverTiming {
	hardcoverTimingMu.RLock()
	defer hardcoverTimingMu.RUnlock()
	return hardcoverTimingIs
}

type progressSent struct {
	at       time.Time
	fraction float64
}

// progressDue says whether a book's progress is worth sending now: the first time, or after enough time and enough
// reading. Hardcover allows about sixty requests a minute, and a page turn must never cost one.
func progressDue(previous progressSent, seen bool, fraction float64, now time.Time) bool {
	if !seen {
		return true
	}
	return now.Sub(previous.at) >= hardcoverTimings().progressEvery && math.Abs(fraction-previous.fraction) >= hardcoverProgressStep
}

// sendProgressToHardcover updates the page count of a book whose Hardcover match is already known (linked by hand,
// or found when the shelf sync added it). It never searches and never adds a book to the shelf.
func (m *multiServer) sendProgressToHardcover(job hardcoverJob) (bool, error) {
	if !m.hardcoverAccount(job.account) || m.hardcoverIgnored.has(job.library, job.book) {
		return false, nil
	}
	id, ok := m.hardcoverLink(job.library, job.book)
	if !ok {
		match, found := m.hardcoverMatches.get(job.library, job.book)
		if !found || match.HardcoverID == "" {
			return false, nil
		}
		id = match.HardcoverID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := m.hardcover.SetProgress(ctx, id, job.progress, time.Now().Format("2006-01-02"))
	switch {
	case err == nil:
		slog.Info("Hardcover progress", "book", job.book, "fraction", job.progress, "result", string(result))
	case errors.Is(err, enrichment.ErrNotOnShelf), errors.Is(err, enrichment.ErrNoPageCount):
		slog.Info("Hardcover progress skipped", "book", job.book, "reason", err.Error())
		return true, nil
	default:
		slog.Warn("Hardcover progress failed", "book", job.book, "error", err)
	}
	return true, err
}
