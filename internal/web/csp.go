package web

// The page carries one inline script: the theme pre-paint that applies the saved palette before app.js
// loads, so a light or e-ink reader does not get a dark flash on every navigation. A document policy of
// script-src 'self' blocks it for exactly the people it exists for, so the policy carries its hash — the
// hash is computed from the page that is actually served, never typed by hand, which means editing the
// script cannot silently disable it.

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// inlineScriptHashes returns the CSP source for every inline <script> body in the document.
func inlineScriptHashes(document string) []string {
	var hashes []string
	for rest := document; ; {
		open := strings.Index(rest, "<script")
		if open < 0 {
			break
		}
		body := strings.Index(rest[open:], ">")
		if body < 0 {
			break
		}
		start := open + body + 1
		close := strings.Index(rest[start:], "</script>")
		if close < 0 {
			break
		}
		content := rest[start : start+close]
		if strings.TrimSpace(content) != "" {
			sum := sha256.Sum256([]byte(content))
			hashes = append(hashes, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
		}
		rest = rest[start+close+len("</script>"):]
	}
	return hashes
}

// documentPolicy is the policy for a page Mochil serves itself. It allows the generated covers and scenes
// (data: and blob: images) that the shell draws, and the inline theme script by hash.
func documentPolicy(inlineHashes []string) string {
	script := "script-src 'self'"
	if len(inlineHashes) > 0 {
		script += " " + strings.Join(inlineHashes, " ")
	}
	return "default-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; " +
		"img-src 'self' data: blob:; media-src 'self' data: blob:; " + script + "; " +
		"connect-src 'self'; frame-src 'self' blob: data:; worker-src 'self' blob:; " +
		"form-action 'self'; frame-ancestors 'none'; base-uri 'none'; object-src 'none'"
}
