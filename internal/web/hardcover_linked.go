package web

// Recording what the reader hand-linked on Hardcover. It lives apart from the stores because it is the one
// part of the shelf that talks to Hardcover, and it has to stay off the request path's critical distance.

import (
	"context"
	"time"
)

func (m *multiServer) recordLinkedMatches(ctx context.Context, onlyLibrary, onlyBook string) {
	if m.hardcoverLinks == nil || m.hardcoverEnqueue == nil {
		return
	}
	// This runs on the request path that opens the shelf, and each unrecorded link costs one Hardcover call.
	// At a few hundred hand-linked books that is minutes of hanging, a 429 partway through, and a permanent
	// empty title for whatever failed - so a page load records at most a handful, and not again for a while.
	m.linkedRecordMu.Lock()
	if time.Since(m.linkedRecordedAt) < 2*time.Minute {
		m.linkedRecordMu.Unlock()
		return
	}
	m.linkedRecordedAt = time.Now()
	m.linkedRecordMu.Unlock()

	const perPass = 5
	recorded := 0
	var fresh []hardcoverMatch
	for _, link := range m.hardcoverLinks.all() {
		if recorded >= perPass {
			break
		}
		if onlyLibrary != "" && (link[0] != onlyLibrary || link[1] != onlyBook) {
			continue
		}
		if existing, ok := m.hardcoverMatches.get(link[0], link[1]); ok && existing.HardcoverID == link[2] && existing.HardcoverTitle != "" {
			continue
		}
		lib, book, ok := m.findBook(link[0] + "/" + link[1])
		if !ok {
			continue
		}
		info := m.hardcoverBookInfo(ctx, lib.ID, book.ID, link[2])
		if info.Title == "" {
			continue // the lookup failed; leave it unrecorded so the next pass tries again
		}
		fresh = append(fresh, hardcoverMatch{
			Library: lib.ID, Book: book.ID, Title: book.Title, Authors: append([]string{}, book.Authors...),
			HardcoverID: link[2], HardcoverTitle: info.Title, HardcoverAuthors: info.Authors, URL: info.SourceURL,
			How: "linked", Result: "linked", At: time.Now().Unix(),
		})
		recorded++
	}
	if len(fresh) > 0 {
		m.hardcoverMatches.putAll(fresh)
	}
}
