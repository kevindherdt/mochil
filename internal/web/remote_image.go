package web

import (
	"net/http"
	"strings"
)

// rasterImageType accepts only picture formats a browser draws without running anything. An SVG from a remote server
// can carry a script, which would run on this site's own address if someone opened the proxied link directly.
func rasterImageType(contentType string) (string, bool) {
	mediaType, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(contentType)), ";")
	switch strings.TrimSpace(mediaType) {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif":
		return strings.TrimSpace(mediaType), true
	}
	return "", false
}

func setRemoteImageHeaders(w http.ResponseWriter, mediaType string) {
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; sandbox")
}

// remoteMediaType keeps an audio or video type and turns anything else (a page, a script) into a plain download, so
// a feed cannot make this site serve its own HTML.
func remoteMediaType(contentType string) string {
	base, _, _ := strings.Cut(strings.TrimSpace(contentType), ";")
	base = strings.ToLower(strings.TrimSpace(base))
	if (strings.HasPrefix(base, "audio/") || strings.HasPrefix(base, "video/")) && len(base) > 6 && !strings.ContainsAny(base, " <>\"") {
		return strings.TrimSpace(contentType)
	}
	return "application/octet-stream"
}

func setRemoteMediaHeaders(w http.ResponseWriter, contentType string) {
	w.Header().Set("Content-Type", remoteMediaType(contentType))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
}
