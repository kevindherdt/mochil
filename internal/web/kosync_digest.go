package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// koSyncDigests finds books by the content hash KOReader uses by default. It is
// filled in the background, because hashing every book touches thousands of files
// and a sync request must not wait for that.
type koSyncDigests struct {
	mu        sync.Mutex
	byBook    map[string]koSyncDigest // "library/book" -> hash of that revision
	running   bool
	refreshed time.Time
	told      map[string]time.Time
	waiting   map[string]waitingKoSync
	// generation counts published digest tables. A queued update remembers the
	// generation it was last replayed against, which is what tells "this book
	// turned up while the pass was finishing" apart from "this device holds a
	// different copy of the book and no retry will ever help".
	generation uint64
}

// koSyncReplayHook runs after a pass has drained its queue and before the queued
// updates are replayed. It is nil in production; a test uses it to enqueue an
// update in exactly that window.
var koSyncReplayHook func()

// waitingKoSyncKey scopes a pending update to one user: two readers syncing the
// same document must both be replayed, so the document alone is not a key.
func waitingKoSyncKey(username, document string) string { return username + "\x00" + document }

// waitingKoSync is the newest update for a document that matched no book yet.
type waitingKoSync struct {
	username  string
	progress  koSyncProgress
	attempted uint64 // digest generation this update was replayed against, 0 = never
}

const maximumWaitingKoSync = 64

type koSyncDigest struct {
	hash     string
	size     int64
	modified time.Time
}

const koSyncDigestRefresh = time.Minute

// lookup returns the books whose content hash is document. A refresh is started
// when the table is empty or old, so a book added since the last pass is found next time.
func (m *multiServer) koSyncDigestMatches(document string) (matches [][2]string) {
	d := &m.koSyncContent
	d.mu.Lock()
	for key, digest := range d.byBook {
		if digest.hash == document {
			libraryID, bookID := splitBookKey(key)
			matches = append(matches, [2]string{libraryID, bookID})
		}
	}
	stale := !d.running && time.Since(d.refreshed) > koSyncDigestRefresh
	if stale {
		d.running = true
	}
	d.mu.Unlock()
	if stale {
		go contained("kosync digests", m.refreshKoSyncDigests)
	}
	return matches
}

func splitBookKey(key string) (string, string) {
	for index := 0; index < len(key); index++ {
		if key[index] == '/' {
			return key[:index], key[index+1:]
		}
	}
	return key, ""
}

func (m *multiServer) refreshKoSyncDigests() {
	d := &m.koSyncContent
	defer func() {
		d.mu.Lock()
		d.running = false
		d.refreshed = time.Now()
		d.generation++
		generation := d.generation
		waiting := d.waiting
		d.waiting = nil
		d.mu.Unlock()
		if koSyncReplayHook != nil {
			koSyncReplayHook()
		}
		// Updates that arrived before their book was indexed can be placed now.
		for _, update := range waiting {
			m.recordKoSyncProgressAt(update.username, update.progress, generation)
		}
		// An update queued while this pass was finishing was never replayed against
		// the table it is now looking at. Ask for another pass soon instead of
		// leaving it for a minute; an update that WAS replayed and still matched
		// nothing carries this generation and will not ask again until the library
		// actually changes.
		d.mu.Lock()
		for _, update := range d.waiting {
			if update.attempted < generation {
				d.refreshed = time.Time{}
				break
			}
		}
		d.mu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	// A book that is no longer in any library must stop matching its old hash,
	// otherwise a deleted book keeps attracting progress and a moved copy makes a
	// formerly unique hash ambiguous. Only a complete pass may prune: an
	// interrupted scan, an unreachable library or a book that could not be read
	// says nothing about the books it did not reach.
	seen := make(map[string]struct{}, 512)
	complete := true
	prunable := true
	for _, library := range m.libraries {
		if library.Store == nil {
			prunable = false
			continue
		}
		for _, book := range library.Store.Books() {
			if ctx.Err() != nil {
				complete = false
				break
			}
			if !isEPUB(book) && book.Format != "pdf" && book.Format != "cbz" {
				continue
			}
			key := library.ID + "/" + book.ID
			// Reached, so its entry stays valid even if the hash cannot be refreshed.
			seen[key] = struct{}{}
			d.mu.Lock()
			known, ok := d.byBook[key]
			d.mu.Unlock()
			if ok && known.size == book.Size && known.modified.Equal(book.Modified) {
				continue
			}
			acquired, err := library.Store.Acquire(ctx, book)
			if acquired.Reader == nil {
				continue
			}
			readerAt, ok := acquired.Reader.(io.ReaderAt)
			if err == nil && ok {
				hash := koSyncPartialMD5(readerAt)
				d.mu.Lock()
				if d.byBook == nil {
					d.byBook = make(map[string]koSyncDigest)
				}
				d.byBook[key] = koSyncDigest{hash: hash, size: book.Size, modified: book.Modified}
				d.mu.Unlock()
			}
			_ = acquired.Reader.Close()
		}
		if !complete {
			break
		}
	}
	if complete && prunable {
		d.mu.Lock()
		for key := range d.byBook {
			if _, ok := seen[key]; !ok {
				delete(d.byBook, key)
			}
		}
		d.mu.Unlock()
	}
}

// noteUnmatchedKoSync says once an hour why a KOReader update did not reach the
// web reader: the document id belongs to no file the server has, which happens
// when the device holds a different copy of the book than the library does.
func (m *multiServer) noteUnmatchedKoSync(username string, progress koSyncProgress) {
	m.noteUnmatchedKoSyncAt(username, progress, 0)
}

func (m *multiServer) noteUnmatchedKoSyncAt(username string, progress koSyncProgress, attempted uint64) {
	d := &m.koSyncContent
	d.mu.Lock()
	// The book may simply not be indexed yet: keep the latest update and try again when it is.
	if d.waiting == nil {
		d.waiting = make(map[string]waitingKoSync)
	}
	key := waitingKoSyncKey(username, progress.Document)
	if _, known := d.waiting[key]; known || len(d.waiting) < maximumWaitingKoSync {
		d.waiting[key] = waitingKoSync{username: username, progress: progress, attempted: attempted}
	}
	if d.told == nil {
		d.told = make(map[string]time.Time)
	}
	recent := time.Since(d.told[progress.Document]) < time.Hour
	if !recent {
		d.told[progress.Document] = time.Now()
	}
	indexed := len(d.byBook)
	building := d.running
	d.mu.Unlock()
	if recent || m.root == nil || m.root.log == nil {
		return
	}
	m.root.log.Info("KOReader progress not linked to a library book",
		slog.String("document", progress.Document), slog.String("device", progress.Device),
		slog.Int("indexed_books", indexed), slog.Bool("index_building", building),
		slog.String("hint", "the device's copy differs from the server's file; download the book from Mochil or use filename matching"))
}

// serveKoSyncLogged answers a KOReader sync request and records what happened, so
// "it did not sync" can be answered from the log: did the device ask, and what was the reply.
func (m *multiServer) serveKoSyncLogged(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthcheck" || m.root == nil || m.root.log == nil {
		m.kosync.serveHTTP(w, r)
		return
	}
	recorder := &statusWriter{ResponseWriter: w}
	started := time.Now()
	m.kosync.serveHTTP(recorder, r)
	document := strings.TrimPrefix(r.URL.Path, "/syncs/progress")
	document = strings.TrimPrefix(document, "/")
	if len(document) > 8 {
		document = document[:8]
	}
	request := r.Method + " " + r.URL.Path
	if document != "" {
		request = r.Method + " /syncs/progress/{document}"
	}
	m.root.log.Info("KOReader sync",
		slog.String("request", request),
		slog.String("user", r.Header.Get("x-auth-user")), slog.String("document", document),
		slog.Int("status", recorder.status), slog.Duration("took", time.Since(started).Round(time.Millisecond)))
}
