package web

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"mochil/internal/catalog"
)

func (m *multiServer) catalog(w http.ResponseWriter, r *http.Request) {
	// The root feed needs a count and the newest modification per library, not
	// the books themselves. Asking each store for its catalogue copied every
	// book -- tens of megabytes on a large install, on every client's landing
	// page.
	type shelf struct {
		count  int
		latest time.Time
	}
	shelves := make([]shelf, len(m.libraries))
	var newest time.Time
	for index, library := range m.libraries {
		if library.Store == nil {
			continue
		}
		count, modified := catalogueSummary(library.Store)
		shelves[index] = shelf{count: count, latest: modified}
		if modified.After(newest) {
			newest = modified
		}
	}
	feed := m.root.feedUpdated("Mochil", "urn:opdshelf:root", updatedAt(newest))
	when := updatedAt(newest)
	// The navigation entry follows the feed: "Continue Reading" is for books actually being read,
	// so a shelf holding only books opened on page one, or parked as on hold or dropped, offers nothing.
	if m.reading != nil && len(m.reading.ListContinue(m.readingAccount(r), "", 1)) > 0 {
		feed.Entries = append(feed.Entries, navEntry("Continue Reading", "Your recently opened books", m.root.href("/opds/continue"), when, "continue-reading"))
	}
	if m.collections != nil && len(m.collections.list(m.readingAccount(r))) > 0 {
		feed.Entries = append(feed.Entries, navEntry("Collections", "Curated books and series", m.root.href("/opds/collections"), when, "collections"))
	}
	for index, library := range m.libraries {
		if library.Store == nil {
			continue
		}
		href := m.root.href("/opds/libraries/" + url.PathEscape(library.ID))
		description := fmt.Sprintf("%d items", shelves[index].count)
		if library.Kind != "" {
			description += " - " + library.Kind
		}
		feed.Entries = append(feed.Entries, navEntry(library.Name, description, href, updatedAt(shelves[index].latest), "library:"+library.ID))
	}
	m.root.writeFeed(w, feed, "navigation")
}

func (m *multiServer) opdsContinue(w http.ResponseWriter, r *http.Request) {
	if m.reading == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	// Same rule as the web and device shelves: only books actually being read, not ones merely
	// opened at page one and not ones parked as on hold or dropped.
	positions := m.reading.ListContinue(m.readingAccount(r), "", 100)
	books := make([]catalog.Book, 0, len(positions))
	type item struct {
		library  Library
		book     catalog.Book
		position readingPosition
	}
	items := make([]item, 0, len(positions))
	for _, position := range positions {
		library, book, ok := m.findBook(position.LibraryID + "/" + position.BookID)
		if !ok {
			continue
		}
		books = append(books, book)
		items = append(items, item{library: library, book: book, position: position})
	}
	feed := m.root.feed("Continue Reading", "urn:mochil:continue-reading", books)
	for _, current := range items {
		authors := make([]Person, 0, len(current.book.Authors))
		for _, author := range current.book.Authors {
			authors = append(authors, Person{Name: author})
		}
		name := filepath.Base(filepath.FromSlash(current.book.Path))
		href := m.root.href("/books/" + url.PathEscape(current.library.ID) + "/" + current.book.ID + "/" + url.PathEscape(name))
		progress := strconv.Itoa(int(math.Round(current.position.Fraction*100))) + "% read"
		feed.Entries = append(feed.Entries, Entry{
			ID: "urn:mochil:book:" + current.library.ID + ":" + current.book.ID, Title: current.book.Title,
			Updated: time.UnixMilli(current.position.UpdatedAt).UTC().Format(time.RFC3339), Authors: authors,
			Content: Content{Type: "text", Text: progress},
			Links:   []Link{{Rel: acquisitionRel, Href: href, Type: current.book.MediaType, Title: "Download"}},
		})
	}
	m.root.writeFeed(w, feed, "acquisition")
}
