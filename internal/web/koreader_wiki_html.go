package web

import (
	"bytes"
	"html"
	"net/http"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
)

// koReaderWikiHTML renders every one of the account's wiki/Notes pages into a single, self-contained HTML
// document with no external resources, so KOReader's crengine (the same engine EPUB/HTML/TXT use) can open it
// directly as a readable, formatted book — unlike the raw Markdown the pages are stored as, which KOReader
// cannot render on its own.
//
// goldmark.Convert is called with its default renderer (no html.WithUnsafe()), which is the sole barrier
// between a page's Markdown source and the HTML returned here: by default it replaces any raw HTML/script in
// the source with an inert comment instead of passing it through. Never add html.WithUnsafe() to this call.
func (m *multiServer) koReaderWikiHTML(w http.ResponseWriter, r *http.Request, account string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	summaries := m.wiki.List(account, false)
	sort.Slice(summaries, func(i, j int) bool { return strings.ToLower(summaries[i].Title) < strings.ToLower(summaries[j].Title) })
	var body bytes.Buffer
	body.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>My Notes &amp; Highlights</title></head><body>")
	body.WriteString("<h1>My Notes &amp; Highlights</h1>")
	if len(summaries) == 0 {
		body.WriteString("<p>No notes yet.</p>")
	}
	for _, summary := range summaries {
		page, ok := m.wiki.Get(account, summary.ID)
		if !ok {
			continue
		}
		body.WriteString("<h2>")
		body.WriteString(html.EscapeString(page.Title))
		body.WriteString("</h2>")
		if err := goldmark.Convert([]byte(page.Content), &body); err != nil {
			body.WriteString("<p><em>This page could not be rendered.</em></p>")
		}
		body.WriteString("<hr>")
	}
	body.WriteString("</body></html>")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(body.Bytes())
}
