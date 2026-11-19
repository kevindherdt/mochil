package web

import (
	"encoding/json"
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

func (m *multiServer) app(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	m.serveStatic(w, r, "static/index.html")
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

// currentBundle maps a superseded bundle name to the one this build ships. Only the content-hashed
// files under static/dist qualify: their names change with every build, so a miss means "old build
// asked", not "typo".
func currentBundle(missing string, manifestJSON []byte) (string, bool) {
	if !strings.HasPrefix(missing, "static/dist/") || missing == "static/dist/manifest.json" {
		return "", false
	}
	var manifest assetManifest
	if json.Unmarshal(manifestJSON, &manifest) != nil {
		return "", false
	}
	switch path.Ext(missing) {
	case ".css":
		if manifest.Style != "" {
			return "static/" + manifest.Style, true
		}
	case ".js", ".mjs":
		if manifest.Script != "" {
			return "static/" + manifest.Script, true
		}
	}
	return "", false
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

func (m *multiServer) assets(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/assets/")
	if name == "" || strings.Contains(name, "..") {
		http.NotFound(w, r)
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
			// A browser that still holds the previous shell asks for a bundle hash this build does
			// not have. Answering 404 is what left the app unstyled and apparently broken until a
			// hard refresh, so serve the current bundle of the same kind instead — the page is then
			// merely the wrong build rather than no build, and the next load corrects it.
			manifestJSON, _ := staticFiles.ReadFile("static/dist/manifest.json")
			if current, mangled := currentBundle(name, manifestJSON); mangled {
				m.serveBundleFallback(w, r, current)
				return
			}
			http.NotFound(w, r)
			return
		}
		if name == "static/index.html" {
			if manifest, manifestErr := staticFiles.ReadFile("static/dist/manifest.json"); manifestErr == nil {
				b = bundledIndex(b, manifest)
			}
		}
		sum := sha256.Sum256(b)
		value, _ = staticAssetCache.LoadOrStore(name, staticAsset{data: b, contentType: mime.TypeByExtension(path.Ext(name)), etag: fmt.Sprintf("\"%x\"", sum[:12])})
	}
	asset := value.(staticAsset)
	if asset.contentType != "" {
		w.Header().Set("Content-Type", asset.contentType)
	}
	w.Header().Set("ETag", asset.etag)
	if strings.HasPrefix(name, "static/dist/") && name != "static/dist/manifest.json" {
		// Bundle file names carry a content hash, so they never change.
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	} else if strings.HasPrefix(name, "static/") && name != "static/index.html" && name != "static/reader.html" && name != "static/reader-readium.html" && name != "static/editor.html" && name != "static/import.html" && name != "static/hermes.html" {
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
	policy := "default-src 'self'; img-src 'self' data: blob:; style-src 'self' blob: 'unsafe-inline'; font-src 'self' data: blob:; media-src 'self' data: blob:; script-src 'self'; frame-src 'self' blob: data:; worker-src 'self' blob:; connect-src 'self'; form-action 'self'; object-src 'none'; base-uri 'none'"
	if name == "static/reader-readium.html" {
		// The Readium navigator renders each chapter in a blob: frame, injects its
		// own blob: scripts and sets a <base>. Allow exactly that on this page only.
		policy = strings.Replace(policy, "script-src 'self'", "script-src 'self' blob:", 1)
		policy = strings.Replace(policy, "base-uri 'none'", "base-uri 'self'", 1)
	}
	if name == "static/hermes.html" {
		// The optional avatar draws a Live2D model whose Cubism core is WebAssembly. This allows compiling
		// WebAssembly on this page only; it does not allow eval.
		policy = strings.Replace(policy, "script-src 'self'", "script-src 'self' 'wasm-unsafe-eval'", 1)
	}
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
