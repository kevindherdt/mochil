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

// policyOptions are the per-page exceptions. Two pages need more than the shell does, and nothing else.
type policyOptions struct {
	// Script is appended to script-src: the reader injects blob: scripts, the avatar compiles wasm.
	Script []string
	// BaseSelf lets a page set a <base>, which the reader needs to resolve chapter resources.
	BaseSelf bool
}

// documentPolicy is the one policy for a page Mochil serves itself; every page goes through it, so a
// directive can only be introduced in one place. It allows the generated covers and scenes (data: and blob:
// images) the shell draws, and any inline script by the hash of the bytes actually served.
func documentPolicy(inlineHashes []string, options policyOptions) string {
	script := append([]string{"'self'"}, inlineHashes...)
	script = append(script, options.Script...)
	base := "base-uri 'none'"
	if options.BaseSelf {
		base = "base-uri 'self'"
	}
	return "default-src 'self'; style-src 'self' blob: 'unsafe-inline'; font-src 'self' data: blob:; " +
		"img-src 'self' data: blob:; media-src 'self' data: blob:; " +
		"script-src " + strings.Join(script, " ") + "; " +
		"connect-src 'self'; frame-src 'self' blob: data:; worker-src 'self' blob:; " +
		"form-action 'self'; frame-ancestors 'none'; " + base + "; object-src 'none'"
}
