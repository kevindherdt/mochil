package web

import (
	"sync"
	"time"

	"mochil/internal/reading"
)

// A request to Hardcover that failed (the network was down, Hardcover was busy) is tried again a few times, later each
// time, and then given up on. Only things that would otherwise be lost are kept: "started reading" repeats with every
// page turn by itself, but "finished" is said once.

type retryEntry struct {
	job      hardcoverJob
	attempts int
	due      time.Time
}

type hardcoverRetries struct {
	mu      sync.Mutex
	waiting map[string]retryEntry
}

func newHardcoverRetries() *hardcoverRetries {
	return &hardcoverRetries{waiting: map[string]retryEntry{}}
}

func retryKey(job hardcoverJob) string {
	kind := "status"
	switch {
	case job.isProgress:
		kind = "progress"
	case job.isRating:
		kind = "rating"
	}
	return job.account + "\x00" + job.library + "\x00" + job.book + "\x00" + kind + "\x00" + string(rune('0'+job.event))
}

// worthRetrying: progress, a rating and "finished" are; the repeating "reading" status is sent again by the next page turn.
func worthRetrying(job hardcoverJob) bool {
	return job.isProgress || job.isRating || job.event == reading.Finished
}

// failed schedules another try, or gives up once the delays are used up.
func (r *hardcoverRetries) failed(job hardcoverJob, now time.Time) {
	if !worthRetrying(job) {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := retryKey(job)
	attempts, delays := r.waiting[key].attempts, hardcoverTimings().retryDelays
	if attempts >= len(delays) {
		delete(r.waiting, key)
		return
	}
	r.waiting[key] = retryEntry{job: job, attempts: attempts + 1, due: now.Add(delays[attempts])}
}

// settled forgets a job that went through, or that a newer one replaces.
func (r *hardcoverRetries) settled(job hardcoverJob) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.waiting, retryKey(job))
}

// due hands back the jobs whose time has come. They stay on the list until they go through, so a second failure
// counts as the next attempt.
func (r *hardcoverRetries) due(now time.Time) []hardcoverJob {
	r.mu.Lock()
	defer r.mu.Unlock()
	var ready []hardcoverJob
	delays := hardcoverTimings().retryDelays
	for key, entry := range r.waiting {
		if !entry.due.After(now) {
			ready = append(ready, entry.job)
			entry.due = now.Add(delays[len(delays)-1])
			r.waiting[key] = entry
		}
	}
	return ready
}
