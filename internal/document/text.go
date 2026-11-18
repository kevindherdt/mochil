package document

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var (
	ErrTextUnsupported = errors.New("document does not have readable text")
	ErrTextTooLarge    = errors.New("document text exceeds the read limit")
	ErrTextEncoding    = errors.New("document text is not valid UTF-8")
	ErrTextMalformed   = errors.New("document container is malformed")
)

const DefaultTextLimit int64 = 2 << 20

// ReadText opens a previously indexed item through the store's path-security
// boundary and reads at most maxBytes. PDFs deliberately remain native files;
// text extraction for them will be a separate bounded worker capability.
func (s *Store) ReadText(id string, maxBytes int64) ([]byte, Item, error) {
	item, ok := s.Item(id)
	if !ok {
		return nil, Item{}, fmt.Errorf("open document text: %w", errors.New("not found"))
	}
	if !TextReadable(item.Format) {
		return nil, item, ErrTextUnsupported
	}
	if maxBytes <= 0 {
		maxBytes = DefaultTextLimit
	}
	// RTF is uncompressed, so one embedded photo pushes a document whose text is a
	// few lines past any byte limit meant for compressed containers. Its decoder
	// enforces the limit on the TEXT it produces, so the encoded size is not the
	// thing to measure; the container cap below keeps the read bounded.
	if item.Format != "rtf" && !officeTextReadable(item.Format) && item.Size > maxBytes {
		return nil, item, ErrTextTooLarge
	}
	file, item, err := s.Open(id)
	if err != nil {
		return nil, item, err
	}
	defer file.Close()
	if item.Format == "docx" {
		content, extractErr := extractDOCX(file, item.Size, maxBytes)
		return content, item, extractErr
	}
	if item.Format == "odt" {
		content, extractErr := extractODT(file, item.Size, maxBytes)
		return content, item, extractErr
	}
	limit := maxBytes + 1
	if item.Format == "rtf" {
		limit = maximumOfficeContainer
	}
	content, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return nil, item, err
	}
	if item.Format != "rtf" && int64(len(content)) > maxBytes {
		return nil, item, ErrTextTooLarge
	}
	// Editable vault revisions must describe the exact bytes the writer checks.
	// Normalizing line endings or stripping a BOM here causes false conflicts
	// and can rewrite unrelated text when the document is saved.
	if s.allowVaultAssets && IsMarkdown(item.Format) {
		if !utf8.Valid(content) {
			return nil, item, ErrTextEncoding
		}
		return content, item, nil
	}
	content = bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf})
	if item.Format == "rtf" {
		decoded, decodeErr := decodeRTF(content, maxBytes)
		return decoded, item, decodeErr
	}
	if !utf8.Valid(content) {
		return nil, item, ErrTextEncoding
	}
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	return content, item, nil
}

func TextReadable(format string) bool {
	return plainTextReadable(format) || format == "rtf" || officeTextReadable(format)
}

// IsMarkdown reports whether a catalogue format is Markdown. Both extensions are Markdown
// everywhere: indexing, tags, backlinks, the link graph and the wiki editor.
func IsMarkdown(format string) bool { return format == "md" || format == "markdown" }

// IsMarkdownPath answers the same question for a file name, using the one extension table the
// catalogue scans with, so a writer that only knows paths cannot drift from it.
func IsMarkdownPath(name string) bool {
	extension := strings.ToLower(filepath.Ext(name))
	if _, ok := formats[extension]; !ok {
		return false
	}
	return IsMarkdown(strings.TrimPrefix(extension, "."))
}

func plainTextReadable(format string) bool {
	switch format {
	case "md", "markdown", "txt", "log", "csv", "tsv", "json", "jsonl", "yaml", "yml", "toml", "tex":
		return true
	default:
		return false
	}
}

func officeTextReadable(format string) bool { return format == "docx" || format == "odt" }

// manageableTextSize reports whether reading this item stays bounded by its own
// format's limits rather than by the encoded file size.
func manageableTextSize(item Item) bool {
	return item.Size <= DefaultTextLimit || officeTextReadable(item.Format) || item.Format == "rtf"
}
