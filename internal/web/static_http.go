package web

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
)

//go:embed static
var staticFiles embed.FS

type staticAsset struct {
	data        []byte
	contentType string
	etag        string
}

var staticAssetCache sync.Map

// legacyApp serves the three pages that have not been ported: the spreadsheet editor, managed import and the old
// chat. The retired shell that used to answer /legacy itself is gone — its 245 modules and 19 000 lines existed only to
// render a library screen the app owns now — so anything else under /legacy is a stale bookmark and goes home.
func (m *multiServer) legacyApp(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(path.Clean(r.URL.Path), "/legacy")
	switch trimmed {
	case "/editor":
		m.serveStatic(w, r, "static/editor.html")
	case "/editor/import":
		m.serveStatic(w, r, "static/import.html")
	case "/hermes":
		m.serveStatic(w, r, "static/hermes.html")
	default:
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

// notFoundPage answers a request for a path no route recognizes with the app's
// own branded 404 page (logo, message, link home) instead of the bare Go
// default "404 page not found" text.
func (m *multiServer) notFoundPage(w http.ResponseWriter, r *http.Request) {
	b, err := staticFiles.ReadFile("static/404.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(b)
}

func (m *multiServer) serveBundleFallback(w http.ResponseWriter, r *http.Request, name string) {
	b, err := staticFiles.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}

// assets serves /assets/… for two builds now that the new app lives at the root: its own content-hashed files are
// emitted under static/next/assets/, and everything the legacy pages still need (reader, login, pairing, icons, design
// tokens) lives under static/assets/. The new build wins where it has the file, because a hash in the name means the
// request could only have come from the build that produced it.
func (m *multiServer) assets(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/assets/")
	if name == "" || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	if fileExistsInFS(staticFiles, "static/next/assets/"+name) {
		m.serveNextFile(w, r, "static/next/assets/"+name)
		return
	}
	m.serveStatic(w, r, "static/"+name)
}

func (m *multiServer) reader(w http.ResponseWriter, r *http.Request) {
	if len(strings.Split(strings.Trim(r.URL.Path, "/"), "/")) != 3 {
		http.NotFound(w, r)
		return
	}
	if r.URL.Query().Get("engine") != "classic" {
		m.serveStatic(w, r, "static/reader-readium.html")
		return
	}
	m.serveStatic(w, r, "static/reader.html")
}

func (m *multiServer) serveStatic(w http.ResponseWriter, r *http.Request, name string) {
	value, ok := staticAssetCache.Load(name)
	if !ok {
		b, err := staticFiles.ReadFile(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		sum := sha256.Sum256(b)
		value, _ = staticAssetCache.LoadOrStore(name, staticAsset{data: b, contentType: mime.TypeByExtension(path.Ext(name)), etag: fmt.Sprintf("\"%x\"", sum[:12])})
	}
	asset := value.(staticAsset)
	// The worker no longer keeps a shell cache, so there is nothing to name after the build: it deletes its caches and
	// unregisters on activate. The rewrite that used to live here went with the shell.
	if asset.contentType != "" {
		w.Header().Set("Content-Type", asset.contentType)
	}
	w.Header().Set("ETag", asset.etag)
	if strings.HasPrefix(name, "static/") && name != "static/reader.html" && name != "static/reader-readium.html" && name != "static/editor.html" && name != "static/import.html" && name != "static/hermes.html" {
		if path.Ext(name) == ".js" {
			// ES modules import one another by URL. Revalidate them so a deploy
			// cannot combine a new parent with an old, immutable dependency.
			w.Header().Set("Cache-Control", "private, no-cache")
		} else if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "private, max-age=86400")
		}
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	// One policy builder for every page: the per-page differences are options, not string surgery, so a
	// directive cannot drift between the pages that carry it.
	options := policyOptions{}
	if name == "static/reader-readium.html" {
		// The Readium navigator renders each chapter in a blob: frame, injects its own blob: scripts and
		// sets a <base>. Allow exactly that on this page only.
		options.Script = []string{"blob:"}
		options.BaseSelf = true
	}
	if name == "static/hermes.html" {
		// The optional avatar draws a Live2D model whose Cubism core is WebAssembly. This allows compiling
		// WebAssembly on this page only; it does not allow eval.
		options.Script = []string{"'wasm-unsafe-eval'"}
	}
	policy := documentPolicy(inlineScriptHashes(string(asset.data)), options)
	w.Header().Set("Content-Security-Policy", policy)
	if r.Header.Get("If-None-Match") == asset.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write(asset.data)
}

// pwaFile serves the installable-app files from the site root: a service worker
// may only control the paths at or below its own URL, so it cannot live under
// /assets/.
func (m *multiServer) pwaFile(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/sw.js" {
		w.Header().Set("Service-Worker-Allowed", "/")
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "application/manifest+json")
	}
	m.serveStatic(w, r, "static"+r.URL.Path)
}
