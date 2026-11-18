package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"mochil/internal/acquisition"
	"mochil/internal/catalog"
	"mochil/internal/webpub"
)

const readiumEntryPolicy = "sandbox; script-src 'none'; object-src 'none'"

// readiumBook serves an EPUB as a Readium Web Publication so the optional
// Readium reader can render it: /api/readium/<library>/<book>/manifest.json
// and, for every archive entry, /api/readium/<library>/<book>/<entry path>.
func (m *multiServer) readiumBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.EscapedPath(), "/api/readium/")
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) != 3 || parts[2] == "" {
		http.NotFound(w, r)
		return
	}
	entry, err := url.PathUnescape(parts[2])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lib, book, ok := m.findBook(parts[0] + "/" + parts[1])
	if !ok || !isEPUB(book) || lib.Store == nil {
		http.NotFound(w, r)
		return
	}
	// Book content is untrusted. The reader fetches these entries and renders them in its own frames, but a chapter opened
	// directly would otherwise run its scripts as this origin; a sandbox without allow-scripts keeps it inert.
	w.Header().Set("Content-Security-Policy", readiumEntryPolicy)
	if kind := derivedKind(entry); kind != "" {
		if body, ok := readiumDerived.get(derivedKey(lib.ID, book, kind)); ok {
			writeDerived(w, entry, body)
			return
		}
	}
	epub, closeEPUB, err := openEPUBFile(lib, book)
	if err != nil {
		http.Error(w, "Could not open this book", http.StatusBadGateway)
		return
	}
	defer closeEPUB()

	if kind := derivedKind(entry); kind != "" {
		body, status := buildDerived(kind, epub, book)
		if status != 0 {
			http.Error(w, derivedErrors[kind], status)
			return
		}
		readiumDerived.put(derivedKey(lib.ID, book, kind), body)
		writeDerived(w, entry, body)
		return
	}
	if webpub.IsVirtual(entry) {
		manifest, err := webpub.Build(epub.reader, epub.size)
		page, ok := "", false
		if err == nil {
			page, ok = manifest.VirtualPage(entry)
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xhtml+xml; charset=utf-8")
		w.Header().Set("Cache-Control", "private, max-age=3600")
		_, _ = io.WriteString(w, page)
		return
	}
	// Stored entries (narration audio is) are served with range support so a
	// player can seek inside a multi-hour track without downloading it first.
	if seeker, _, ok := webpub.OpenSeekable(epub.reader, epub.size, entry); ok {
		if contentType := mime.TypeByExtension(path.Ext(entry)); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.Header().Set("Cache-Control", "private, max-age=3600")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, entry, time.Time{}, seeker)
		return
	}
	handle, size, ok := webpub.Open(epub.reader, epub.size, entry)
	if !ok {
		http.NotFound(w, r)
		return
	}
	defer handle.Close()
	contentType := mime.TypeByExtension(path.Ext(entry))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(w, handle)
}

type epubFile struct {
	reader io.ReaderAt
	size   int64
}

// openEPUBFile opens a library EPUB for random access, buffering it in memory
// only when the store cannot seek.
func openEPUBFile(lib Library, book catalog.Book) (epubFile, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), positionBridgeTimeout)
	opened, err := (acquisition.Service{}).Open(ctx, lib.Store, book, "")
	if err != nil {
		cancel()
		return epubFile{}, nil, err
	}
	closeAll := func() { _ = opened.Close(); cancel() }
	if opened.Reader == nil {
		closeAll()
		return epubFile{}, nil, io.ErrUnexpectedEOF
	}
	if at, ok := opened.Reader.(io.ReaderAt); ok {
		if size, err := opened.Reader.Seek(0, io.SeekEnd); err == nil {
			return epubFile{at, size}, closeAll, nil
		}
	}
	data, err := io.ReadAll(io.LimitReader(opened.Reader, positionBridgeMaxEPUB))
	if err != nil {
		closeAll()
		return epubFile{}, nil, err
	}
	return epubFile{bytes.NewReader(data), int64(len(data))}, closeAll, nil
}

var derivedErrors = map[string]string{
	"manifest":  "This book cannot be opened by the Readium reader",
	"positions": "This book cannot be opened by the Readium reader",
	"overlay":   "This book has no read-along narration",
}

// derivedKind names the JSON documents built from the EPUB rather than read from it.
func derivedKind(entry string) string {
	switch entry {
	case "manifest.json":
		return "manifest"
	case "positions.json":
		return "positions"
	case "overlay.json":
		return "overlay"
	}
	return ""
}

func buildDerived(kind string, epub epubFile, book catalog.Book) ([]byte, int) {
	var value any
	switch kind {
	case "overlay":
		overlay, err := webpub.BuildOverlay(epub.reader, epub.size)
		if err != nil {
			return nil, http.StatusUnprocessableEntity
		}
		value = overlay
	default:
		manifest, err := webpub.Build(epub.reader, epub.size)
		if err != nil {
			return nil, http.StatusUnprocessableEntity
		}
		if kind == "positions" {
			positions, err := manifest.Positions()
			if err != nil {
				return nil, http.StatusUnprocessableEntity
			}
			value = positions
			break
		}
		if manifest.Metadata.Title == "" {
			manifest.Metadata.Title = book.Title
		}
		manifest.Links = []webpub.Link{
			{Href: "manifest.json", Type: "application/webpub+json", Rel: []string{"self"}},
			{Href: "positions.json", Type: "application/vnd.readium.position-list+json", Rel: []string{"http://readium.org/positions"}},
		}
		value = manifest
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	return append(body, '\n'), 0
}

func writeDerived(w http.ResponseWriter, entry string, body []byte) {
	switch entry {
	case "positions.json":
		w.Header().Set("Content-Type", "application/vnd.readium.position-list+json")
		w.Header().Set("Cache-Control", "private, no-cache")
	case "overlay.json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "private, max-age=300")
	default:
		w.Header().Set("Content-Type", "application/webpub+json")
		w.Header().Set("Cache-Control", "private, no-cache")
	}
	_, _ = w.Write(body)
}
