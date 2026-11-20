package web

import (
	"bytes"
	"html"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/yuin/goldmark"

	"mochil/internal/document"
)

// koReaderVaultHTML renders every Markdown note in one vault-kind library into a single, self-contained
// HTML document, the same way koReaderWikiHTML renders the account's own Notes & wiki pages -- so
// KOReader's crengine can open a live Obsidian-style vault directly as a readable, formatted book.
// Non-Markdown vault files (images, PDFs, ...) are skipped: this is a bundle of readable notes, not a
// file browser, and KOReader has no use for an image or PDF embedded inside one HTML document.
//
// goldmark.Convert is called with its default renderer (no html.WithUnsafe()) for the exact reason
// koReaderWikiHTML's doc comment explains: it is the sole barrier against raw HTML/script in a note.
func (m *multiServer) koReaderVaultHTML(w http.ResponseWriter, r *http.Request, libraryID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(libraryID)
	if !ok || library.Kind != "vault" || library.Documents == nil {
		http.NotFound(w, r)
		return
	}
	items := library.Documents.Items()
	sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title) })
	var body bytes.Buffer
	title := html.EscapeString(library.Name)
	body.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>" + title + "</title></head><body>")
	body.WriteString("<h1>" + title + "</h1>")
	var rendered int
	for _, item := range items {
		if !document.IsMarkdown(item.Format) {
			continue
		}
		content, err := readVaultItem(library.Documents, item)
		if err != nil {
			continue
		}
		rendered++
		body.WriteString("<h2>")
		body.WriteString(html.EscapeString(item.Title))
		body.WriteString("</h2>")
		if err := goldmark.Convert(content, &body); err != nil {
			body.WriteString("<p><em>This note could not be rendered.</em></p>")
		}
		body.WriteString("<hr>")
	}
	if rendered == 0 {
		body.WriteString("<p>No notes yet.</p>")
	}
	body.WriteString("</body></html>")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(body.Bytes())
}

func readVaultItem(store *document.Store, item document.Item) ([]byte, error) {
	file, _, err := store.Open(item.ID)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	const maxVaultNoteBytes = 2 << 20
	return io.ReadAll(io.LimitReader(file, maxVaultNoteBytes))
}
