package web

// next_app.go serves the React/TypeScript app — the default since cutover. It is deliberately separate from
// static_http.go's legacy serveStatic: that function carries a lot of legacy-only behaviour
// (bundle-fallback-on-hash-miss, sw.js shell renaming, ?v= cache busting) that does not apply here, because the new
// app's assets are content-hashed by Vite and need none of it.

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

// nextApp serves the SPA from the root. A request for a real built file (an asset under /assets/… or index.html)
// is served as-is; anything else is an app route, so index.html goes back and the client-side router takes over.
func (m *multiServer) nextApp(w http.ResponseWriter, r *http.Request) {
	clean := path.Clean(r.URL.Path)
	name := "static/next" + strings.TrimSuffix(clean, "/")
	if clean == "/" {
		name = "static/next/index.html"
	}
	if strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}

	if _, err := staticFiles.ReadFile(name); err != nil {
		if !fileExistsInFS(staticFiles, "static/next/index.html") {
			// web/ hasn't been built into this image at all (no Node build stage yet, or a dev
			// checkout) — say so plainly rather than serving a 404 that looks like a typo'd URL.
			http.Error(w, "The new app (web/) is not built into this image yet.", http.StatusNotFound)
			return
		}
		m.serveNextFile(w, r, "static/next/index.html")
		return
	}
	m.serveNextFile(w, r, name)
}

func (m *multiServer) serveNextFile(w http.ResponseWriter, r *http.Request, name string) {
	value, ok := staticAssetCache.Load(name)
	if !ok {
		b, err := staticFiles.ReadFile(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		sum := sha256.Sum256(b)
		value, _ = staticAssetCache.LoadOrStore(name, staticAsset{
			data:        b,
			contentType: mime.TypeByExtension(path.Ext(name)),
			etag:        fmt.Sprintf("%q", sum[:12]),
		})
	}
	asset := value.(staticAsset)
	if asset.contentType != "" {
		w.Header().Set("Content-Type", asset.contentType)
	}
	w.Header().Set("ETag", asset.etag)
	if name == "static/next/index.html" {
		// The one file whose own name never changes, so it must never be cached — every deploy needs
		// a fresh index.html to pick up the new, content-hashed asset filenames it links to.
		w.Header().Set("Cache-Control", "no-store")
	} else {
		// Every other built file is named by Vite with a content hash; the name changes when the
		// content does, so it is safe to cache forever.
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	}
	_, _ = w.Write(asset.data)
}

func fileExistsInFS(f fs.FS, name string) bool {
	_, err := fs.Stat(f, name)
	return err == nil
}
