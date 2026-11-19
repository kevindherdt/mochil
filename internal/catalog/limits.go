package catalog

import (
	"path/filepath"
	"strings"
)

const (
	maxManifestItems    = 1_000_000
	maxRemoteCoverBytes = 20 << 20
)

func mediaTypeForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".epub":
		return "application/epub+zip"
	case ".cbz":
		return "application/vnd.comicbook+zip"
	case ".pdf":
		return "application/pdf"
	case ".mobi", ".azw3":
		return "application/x-mobipocket-ebook"
	default:
		return "application/octet-stream"
	}
}
