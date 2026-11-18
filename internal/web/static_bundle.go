package web

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// assetManifest describes the bundle written by cmd/assetbuild.
type assetManifest struct {
	Script  string   `json:"script"`
	Style   string   `json:"style"`
	Styles  []string `json:"styles"`
	Preload []string `json:"preload"`
}

var (
	indexStylesheetLink = regexp.MustCompile(`[ \t]*<link rel="stylesheet" href="/assets/([^"?]+)(?:\?[^"]*)?"\s*/?>\n?`)
	indexEntryScript    = regexp.MustCompile(`<script type="module" src="/assets/[^"]+"></script>`)
)

// bundledIndex rewrites index.html to load the production bundle: one hashed
// stylesheet, one hashed entry script, and modulepreload hints for the chunks
// the entry needs immediately. It returns the page unchanged when there is no
// usable manifest, so unbundled sources keep working in development.
func bundledIndex(index, manifestJSON []byte) []byte {
	var manifest assetManifest
	if json.Unmarshal(manifestJSON, &manifest) != nil || manifest.Script == "" || manifest.Style == "" {
		return index
	}
	bundled := map[string]bool{}
	for _, name := range manifest.Styles {
		bundled[name] = true
	}
	page := string(index)
	inserted := false
	page = indexStylesheetLink.ReplaceAllStringFunc(page, func(link string) string {
		match := indexStylesheetLink.FindStringSubmatch(link)
		if match == nil || !bundled[match[1]] {
			return link
		}
		if inserted {
			return ""
		}
		inserted = true
		return fmt.Sprintf("    <link rel=\"stylesheet\" href=\"/assets/%s\" />\n", manifest.Style)
	})
	if !inserted || !indexEntryScript.MatchString(page) {
		return index
	}
	var hints strings.Builder
	for _, chunk := range manifest.Preload {
		fmt.Fprintf(&hints, "    <link rel=\"modulepreload\" href=\"/assets/%s\" />\n", chunk)
	}
	page = strings.Replace(page, "  </head>", hints.String()+"  </head>", 1)
	page = indexEntryScript.ReplaceAllString(page, fmt.Sprintf(`<script type="module" src="/assets/%s"></script>`, manifest.Script))
	return []byte(page)
}
