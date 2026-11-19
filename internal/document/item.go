// Package document owns read-only document-library discovery and access.
// Authored wiki pages deliberately live in a separate package and store.
package document

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Item struct {
	ID        string    `json:"id"`
	LibraryID string    `json:"library_id"`
	Path      string    `json:"path"`
	Title     string    `json:"title"`
	Format    string    `json:"format"`
	MediaType string    `json:"media_type"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
	Revision  string    `json:"revision"`
}

var formats = map[string]string{
	".md":       "text/markdown; charset=utf-8",
	".markdown": "text/markdown; charset=utf-8",
	".txt":      "text/plain; charset=utf-8",
	".log":      "text/plain; charset=utf-8",
	".csv":      "text/csv; charset=utf-8",
	".tsv":      "text/tab-separated-values; charset=utf-8",
	".json":     "application/json; charset=utf-8",
	".jsonl":    "application/x-ndjson; charset=utf-8",
	".yaml":     "application/yaml; charset=utf-8",
	".yml":      "application/yaml; charset=utf-8",
	".toml":     "application/toml; charset=utf-8",
	".tex":      "application/x-tex; charset=utf-8",
	".pdf":      "application/pdf",
	".docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".odt":      "application/vnd.oasis.opendocument.text",
	".rtf":      "application/rtf",
}

// Obsidian vault embeds are catalogue assets rather than ordinary document
// entries. Only vault stores opt into these formats.
var vaultAssetFormats = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
	".avif": "image/avif",
}

func identify(libraryID, relative string, size int64, modified time.Time, allowVaultAssets bool) (Item, bool) {
	relative = filepath.ToSlash(filepath.Clean(relative))
	if size < 0 || relative == "." || relative == ".." || strings.HasPrefix(relative, "../") || filepath.IsAbs(relative) {
		return Item{}, false
	}
	for _, part := range strings.Split(relative, "/") {
		if strings.HasPrefix(part, ".") {
			return Item{}, false
		}
	}
	base := filepath.Base(relative)
	extension := strings.ToLower(filepath.Ext(base))
	mediaType, ok := formats[extension]
	if !ok && allowVaultAssets {
		mediaType, ok = vaultAssetFormats[extension]
	}
	if !ok || strings.HasPrefix(base, "._") {
		return Item{}, false
	}
	identity := sha256.Sum256([]byte(libraryID + "\x00" + relative))
	revision := sha256.Sum256([]byte(relative + "\x00" + strconv.FormatInt(size, 10) + "\x00" + modified.UTC().Format(time.RFC3339Nano)))
	title := strings.TrimSpace(strings.Join(strings.Fields(strings.NewReplacer("_", " ", ".", " ").Replace(strings.TrimSuffix(base, filepath.Ext(base)))), " "))
	if title == "" {
		title = "Untitled document"
	}
	return Item{
		ID: hex.EncodeToString(identity[:16]), LibraryID: libraryID, Path: relative,
		Title: title, Format: strings.TrimPrefix(extension, "."), MediaType: mediaType,
		Size: size, Modified: modified, Revision: hex.EncodeToString(revision[:12]),
	}, true
}
