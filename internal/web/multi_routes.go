package web

import (
	"net/http"
	"net/url"
	"strings"
)

// Routing helpers the multi-library server shares between mounts: everything that has to rewrite the path of an incoming
// request before a per-library handler sees it. Extracted from multi.go, which the module-size budget holds at 550 lines.

func (m *multiServer) serveAcquisition(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, "/books/")
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) > 0 {
		id, err := url.PathUnescape(parts[0])
		if err == nil {
			if _, ok := m.handlers[id]; ok {
				m.serveLibrary(w, r, "/books/", false)
				return
			}
		}
	}
	if len(m.libraries) == 1 {
		serveRewritten(m.handlers[m.libraries[0].ID], w, r, r.URL.Path)
		return
	}
	http.NotFound(w, r)
}

func (m *multiServer) serveStream(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, "/stream/")
	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id, err := url.PathUnescape(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	handler, ok := m.handlers[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveRewritten(handler, w, r, "/stream/"+parts[1])
}

func (m *multiServer) serveLibrary(w http.ResponseWriter, r *http.Request, prefix string, catalogRoute bool) {
	remainder := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(remainder, "/", 2)
	id, err := url.PathUnescape(parts[0])
	if err != nil || id == "" {
		http.NotFound(w, r)
		return
	}
	handler, ok := m.handlers[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	suffix := ""
	if len(parts) == 2 {
		suffix = "/" + parts[1]
	}
	if suffix == "/" {
		suffix = ""
	}
	internalPath := "/books" + suffix
	if catalogRoute {
		if suffix == "/opensearch.xml" {
			internalPath = "/opensearch.xml"
		} else {
			internalPath = "/opds" + suffix
		}
	}
	serveRewritten(handler, w, r, internalPath)
}

func (m *multiServer) serveLegacy(w http.ResponseWriter, r *http.Request) {
	library := m.libraries[0]
	handler := m.handlers[library.ID]
	serveRewritten(handler, w, r, r.URL.Path)
}

func serveRewritten(handler http.Handler, w http.ResponseWriter, r *http.Request, path string) {
	clone := r.Clone(r.Context())
	clonedURL := *r.URL
	clonedURL.Path = path
	clonedURL.RawPath = ""
	clone.URL = &clonedURL
	handler.ServeHTTP(w, clone)
}
