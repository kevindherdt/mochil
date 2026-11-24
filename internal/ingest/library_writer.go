package ingest

// library_writer.go — writing metadata beside books in a library folder that is not necessarily the managed root.
//
// The executor's two roots exist because copying must not read what it is writing; that rule is about imports and it
// makes the executor refuse a library whose folder *is* the drop folder — which is exactly how a person uses a folder
// of books they dragged in. A sidecar is not a copy: it goes where the book already is, so the only guarantee needed is
// that the path resolves inside the library it was asked about. That is what this type checks, and then it hands the
// actual writing to the same code the executor uses (`opf.WriteBeside`, `embedEPUB`), so there is one writer of each
// file, not two.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mochil/internal/opf"
)

// LibraryWriter writes metadata into one library folder. Paths are relative to Root, the same form a catalog book
// carries, and nothing outside it can be reached: "..", an absolute path, a backslash and a symlink that leaves the
// root are all refused before a byte is written.
type LibraryWriter struct {
	root string
	mu   sync.Mutex
}

func NewLibraryWriter(root string) (*LibraryWriter, error) {
	resolved, err := filepath.EvalSymlinks(strings.TrimSpace(root))
	if err != nil {
		return nil, errors.New("the library folder cannot be read")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return nil, errors.New("the library folder is not a directory")
	}
	return &LibraryWriter{root: resolved}, nil
}

// Root is the resolved library folder every path is judged against.
func (w *LibraryWriter) Root() string { return w.root }

// resolve turns a catalog path into the file it names, or refuses it.
func (w *LibraryWriter) resolve(bookPath string, publicationsOnly bool) (string, error) {
	if !safeJournalPath(bookPath) || strings.HasPrefix(bookPath, ".ingest/") {
		return "", errors.New("the path is not a managed library path")
	}
	if publicationsOnly {
		if kind, ok := kindForExtension(filepath.Ext(bookPath)); !ok || kind != Publication {
			return "", errors.New("sidecars are written for books only")
		}
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(w.root, filepath.FromSlash(bookPath)))
	if err != nil {
		return "", errors.New("the book is not in the library")
	}
	if !withinRoot(w.root, resolved) {
		return "", errors.New("the book resolves outside the library")
	}
	return resolved, nil
}

// WriteSidecar writes the metadata.opf beside one book and returns its path relative to the library root.
func (w *LibraryWriter) WriteSidecar(bookPath string, meta opf.Metadata, overwrite bool) (string, error) {
	resolved, err := w.resolve(bookPath, true)
	if err != nil {
		return "", err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	written, err := opf.WriteBeside(resolved, meta, overwrite)
	if err != nil {
		return "", err
	}
	relative, relErr := filepath.Rel(w.root, written)
	if relErr != nil {
		return "", relErr
	}
	return filepath.ToSlash(relative), nil
}

// EmbedMetadata writes the fields into the EPUB itself, so the file carries them with no sidecar beside it.
func (w *LibraryWriter) EmbedMetadata(bookPath string, meta opf.Metadata) error {
	resolved, err := w.resolve(bookPath, false)
	if err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Ext(resolved), ".epub") {
		return errors.New("only EPUB files can have metadata embedded")
	}
	if strings.TrimSpace(meta.Title) == "" {
		return errors.New("a book needs a title to embed")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return embedEPUB(resolved, meta)
}
