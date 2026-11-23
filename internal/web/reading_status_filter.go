package web

// The reading-status filter for /api/books (?status=). Reading progress is per account and lives in the reading
// store, while internal/query only ever sees catalogue data — so the filter is built here, as a predicate the
// query layer applies inside its scan (catalogquery.BooksRequest.Keep). That ordering matters: the predicate runs
// before totals and paging, so X-Total-Count, HasMore and the cursor anchor all describe the filtered shelf.
//
// The alternative — query returning a page and this layer dropping rows — was rejected because a client asking for
// 120 items would get however many survived, with a count that neither matched the page nor could be paged through.

import "mochil/internal/catalog"

// The three views ?status= accepts. "unread" is already declared for smart collections (smart_collection_rules.go)
// with the same meaning and is reused; the other two get their own names so the two vocabularies can't drift into
// sharing a constant that quietly means something else in one of them.
const (
	bookStatusInProgress = "in_progress"
	bookStatusRead       = "read"
)

// keepByReadingStatus returns a predicate answering "does this book belong in the <status> view?" for one account.
//
// It takes a single snapshot of that account's positions up front rather than calling reading.Get per book: a
// library here has thousands of books, and a lock acquisition per book while the query holds the shelf scan would
// be both slow and pointless. The snapshot is keyed exactly like the store's own map.
func (m *multiServer) keepByReadingStatus(account, status string) func(libraryID string, book catalog.Book) bool {
	if m.reading == nil {
		// No reading store means nothing is ever in progress or finished, so "unread" is everything. A read view
		// on an install without reading data is honestly empty rather than a fabricated list.
		return func(string, catalog.Book) bool { return status == statusUnread }
	}
	positions := m.reading.Snapshot(account)
	return func(libraryID string, book catalog.Book) bool {
		position, seen := positions[readingKey(libraryID, book.ID)]
		switch status {
		case bookStatusRead:
			return seen && position.Finished
		case bookStatusInProgress:
			// Same definition the Continue shelf uses: started, not finished, not dismissed, not parked. A book
			// opened on page one to look at it is not "in progress".
			return seen && !position.Finished && !position.Dismissed && !position.Parked() && position.Started()
		default: // unread
			// Never opened, or opened and then explicitly dismissed. Finished books are not unread.
			return !seen || position.Dismissed || (!position.Started() && !position.Finished)
		}
	}
}

// readingKey mirrors reading.readingKey ("libraryID\0bookID") so the predicate can look positions up without the
// store exporting its key format. Kept beside its only use, and pinned by a test against the store rather than
// trusted by eye — if the two ever diverge, every status view silently answers "unread".
func readingKey(libraryID, bookID string) string {
	return libraryID + "\x00" + bookID
}
