package web

import (
	"net/http"
	"strings"
	"time"

	"mochil/internal/catalog"
)

// hardcoverFacts is what the Hardcover sync knows about the books, read once for a whole editor list.
type hardcoverFacts struct {
	on      bool
	matches map[string]hardcoverMatch
	skipped map[string]bool
	ignored map[string]ignoredBook
}

func (m *multiServer) hardcoverFacts() hardcoverFacts {
	facts := hardcoverFacts{on: m.moduleOn("hardcover"), matches: map[string]hardcoverMatch{}, skipped: map[string]bool{}, ignored: map[string]ignoredBook{}}
	if !facts.on {
		return facts
	}
	for _, match := range m.hardcoverMatches.list() {
		facts.matches[linkKey(match.Library, match.Book)] = match
	}
	for _, book := range m.hardcoverSkipped.list() {
		facts.skipped[linkKey(book.Library, book.Book)] = true
	}
	for _, book := range m.hardcoverIgnored.list() {
		facts.ignored[linkKey(book.Library, book.Book)] = book
	}
	return facts
}

// state says how a book stands with the Hardcover sync, and why in words:
// "ignored" (left out of the sync), "doubt" (synced, but the match deserves a look), "synced", "nomatch" (no Hardcover
// book found for it), "linked" (tied to a Hardcover book but not synced yet), or "" (nothing yet).
func (f hardcoverFacts) state(m *multiServer, lib Library, book catalog.Book) (string, string) {
	if !f.on {
		return "", ""
	}
	key := linkKey(lib.ID, book.ID)
	if ignored, ok := f.ignored[key]; ok {
		return "ignored", "Left out of the Hardcover sync: " + ignored.Reason + "."
	}
	if match, ok := f.matches[key]; ok {
		how := "Matched by its title and author"
		if match.How == "linked" {
			how = "Linked by hand"
		}
		note := how + " to " + match.HardcoverTitle
		if len(match.HardcoverAuthors) > 0 {
			note += " by " + strings.Join(match.HardcoverAuthors, ", ")
		}
		note += " on Hardcover; last sync " + time.Unix(match.At, 0).Format("2 Jan 2006") + "."
		if match.Doubt != "" {
			return "doubt", note + " " + match.Doubt
		}
		return "synced", note
	}
	if f.skipped[key] {
		return "nomatch", "Hardcover has no book for this title and author, so its reading is not sent."
	}
	if _, linked := m.hardcoverLink(lib.ID, book.ID); linked {
		return "linked", "Tied to a Hardcover book; nothing has been sent yet."
	}
	return "", ""
}

// editorHardcoverSync leaves the picked books out of the Hardcover sync, or puts them back in.
func (m *multiServer) editorHardcoverSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !m.moduleOn("hardcover") {
		http.Error(w, "Turn on the Hardcover module in Settings first.", http.StatusServiceUnavailable)
		return
	}
	var in struct {
		Library string   `json:"library"`
		IDs     []string `json:"ids"`
		Ignore  bool     `json:"ignore"`
	}
	if err := decodeJSONRequest(w, r, 4<<20, &in, true); err != nil || len(in.IDs) == 0 {
		http.Error(w, "Pick some books first.", http.StatusBadRequest)
		return
	}
	lib, ok := m.editorLibrary(w, in.Library)
	if !ok {
		return
	}
	if !publicationKind(lib.Kind) {
		http.Error(w, "Hardcover sync is only for book libraries.", http.StatusBadRequest)
		return
	}
	wanted := map[string]bool{}
	for _, id := range in.IDs {
		wanted[id] = true
	}
	var batch []ignoredBook
	for _, book := range lib.Store.Books() {
		if wanted[book.ID] && m.hardcoverIgnored.has(lib.ID, book.ID) != in.Ignore {
			batch = append(batch, ignoredBook{Library: lib.ID, Book: book.ID, Title: book.Title, Authors: append([]string{}, book.Authors...), Reason: ignoredByYou, At: time.Now().Unix()})
		}
	}
	// One write for the whole request, and one honest answer: the old loop rewrote and fsynced the file per book and then
	// reported a count even when every write had failed.
	var err error
	if in.Ignore {
		err = m.hardcoverIgnored.addAll(batch)
	} else {
		err = m.hardcoverIgnored.removeAll(batch)
		// Resuming means the next sync may put the book back on the shelf; the per-book path used to do this inside
		// setHardcoverIgnored, which the batch no longer calls.
		for _, item := range batch {
			m.hardcoverMatches.forgetShelf(item.Library, item.Book)
		}
	}
	if err != nil {
		http.Error(w, "Could not save which books sync to Hardcover", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"changed": len(batch)})
}
