package enrichment

import "strings"

// ValidTMDBProfilePath accepts a single TMDB image filename, never a URL or path.
func ValidTMDBProfilePath(path string) bool {
	if len(path) < 6 || len(path) > 200 || path[0] != '/' {
		return false
	}
	name := path[1:]
	if !(strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".webp")) {
		return false
	}
	stem := name[:strings.LastIndexByte(name, '.')]
	if stem == "" {
		return false
	}
	for _, c := range stem {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
