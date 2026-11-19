package web

import (
	"net/http"
	"strings"
)

// publicBase is the address other devices should use to reach this server. When public_url is
// configured it wins; otherwise the request's own host is used, so a plain LAN install can pair a TV
// without editing any configuration first. The value only travels back to the caller that already
// reached this server, so a spoofed Host cannot redirect anyone who is not already talking to it.
func (m *multiServer) publicBase(r *http.Request) string {
	if base := strings.TrimRight(strings.TrimSpace(m.cfg.PublicURL), "/"); base != "" {
		return base
	}
	if r == nil {
		return ""
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// A reverse proxy terminates TLS, so a forwarded "https" is believed — but never a forwarded
	// "http": letting a client ask for cleartext would hand a device or a verification link an
	// address that downgrades a connection the server is actually serving over TLS.
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
		scheme = "https"
	}
	host := strings.TrimSpace(r.Host)
	// Only a bare host:port is acceptable. A Host carrying credentials, a path or whitespace would
	// otherwise be echoed into a link a person is asked to open.
	if host == "" || strings.ContainsAny(host, "/\\?#@ 	") {
		return ""
	}
	return scheme + "://" + host
}
