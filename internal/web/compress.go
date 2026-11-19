package web

import (
	"compress/gzip"
	"mime"
	"net/http"
	"strings"
	"sync"
)

var gzipWriters = sync.Pool{New: func() any {
	writer, _ := gzip.NewWriterLevel(nil, 5)
	return writer
}}

// Compress gzips text-like responses (HTML, CSS, JavaScript, JSON, XML, SVG).
// Media, archives, images, ranged and already-encoded responses pass through
// untouched, so streaming and seeking behave exactly as before.
func Compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead || r.Header.Get("Range") != "" || r.Header.Get("Upgrade") != "" ||
			!strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		// Compressed responses carry a weak ETag; handlers compare strong ones.
		if match := r.Header.Get("If-None-Match"); strings.Contains(match, "W/") {
			r.Header.Set("If-None-Match", strings.ReplaceAll(match, "W/", ""))
		}
		cw := &compressWriter{ResponseWriter: w}
		defer cw.close()
		next.ServeHTTP(cw, r)
	})
}

type compressWriter struct {
	http.ResponseWriter
	gzip        *gzip.Writer
	wroteHeader bool
}

func compressible(contentType string) bool {
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	switch {
	case strings.HasPrefix(media, "text/") && media != "text/event-stream":
		return true
	case media == "application/json", media == "application/javascript", media == "application/xml",
		media == "image/svg+xml", strings.HasSuffix(media, "+json"), strings.HasSuffix(media, "+xml"):
		return true
	}
	return false
}

func (c *compressWriter) WriteHeader(status int) {
	if c.wroteHeader {
		return
	}
	c.wroteHeader = true
	header := c.Header()
	if status >= 200 && status != http.StatusNoContent && status != http.StatusNotModified && status != http.StatusPartialContent &&
		header.Get("Content-Encoding") == "" && compressible(header.Get("Content-Type")) {
		header.Add("Vary", "Accept-Encoding")
		header.Set("Content-Encoding", "gzip")
		header.Del("Content-Length")
		if etag := header.Get("Etag"); etag != "" && !strings.HasPrefix(etag, "W/") {
			header.Set("Etag", "W/"+etag)
		}
		c.gzip = gzipWriters.Get().(*gzip.Writer)
		c.gzip.Reset(c.ResponseWriter)
	}
	c.ResponseWriter.WriteHeader(status)
}

func (c *compressWriter) Write(data []byte) (int, error) {
	if !c.wroteHeader {
		if c.Header().Get("Content-Type") == "" {
			c.Header().Set("Content-Type", http.DetectContentType(data))
		}
		c.WriteHeader(http.StatusOK)
	}
	if c.gzip != nil {
		return c.gzip.Write(data)
	}
	return c.ResponseWriter.Write(data)
}

func (c *compressWriter) Flush() {
	if c.gzip != nil {
		_ = c.gzip.Flush()
	}
	if flusher, ok := c.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (c *compressWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }

func (c *compressWriter) close() {
	if c.gzip != nil {
		_ = c.gzip.Close()
		gzipWriters.Put(c.gzip)
		c.gzip = nil
	}
}
