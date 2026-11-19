// Package vaultwriter owns guarded mutations for local Obsidian-compatible
// vaults. It intentionally stays separate from the read-only document index.
package vaultwriter

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mochil/internal/document"
)

var (
	ErrConflict    = errors.New("vault page changed since it was opened")
	ErrExists      = errors.New("vault entry already exists")
	ErrInvalidPath = errors.New("invalid vault path")
)

const MaximumPageBytes = 8 << 20

type Writer struct {
	root string
	mu   sync.Mutex
}

func New(root string) (*Writer, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve vault root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("resolve vault root: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return nil, errors.New("vault root is not a directory")
	}
	return &Writer{root: resolved}, nil
}

func Revision(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:16])
}

func (w *Writer) CreateFolder(relative string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	path, err := w.resolve(relative, false)
	if err != nil {
		return err
	}
	if err = os.Mkdir(path, 0o750); errors.Is(err, fs.ErrExist) {
		return ErrExists
	} else if err != nil {
		return fmt.Errorf("create vault folder: %w", err)
	}
	return syncDirectory(filepath.Dir(path))
}

func (w *Writer) CreatePage(relative string, content []byte) (string, error) {
	if len(content) > MaximumPageBytes {
		return "", errors.New("vault page is too large")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	path, err := w.resolve(relative, true)
	if err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if errors.Is(err, fs.ErrExist) {
		return "", ErrExists
	} else if err != nil {
		return "", fmt.Errorf("create vault page: %w", err)
	}
	if _, err = file.Write(content); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("create vault page: %w", err)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("create vault page: %w", closeErr)
	}
	if err = syncDirectory(filepath.Dir(path)); err != nil {
		return "", err
	}
	return Revision(content), nil
}

func (w *Writer) UpdatePage(relative, expectedRevision string, content []byte) (string, error) {
	if expectedRevision == "" || len(content) > MaximumPageBytes {
		return "", ErrInvalidPath
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	path, err := w.resolve(relative, true)
	if err != nil {
		return "", err
	}
	current, mode, err := readPage(path)
	if err != nil {
		return "", err
	}
	if Revision(current) != expectedRevision {
		return "", ErrConflict
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".mochil-save-*")
	if err != nil {
		return "", fmt.Errorf("prepare vault save: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err = temporary.Chmod(mode.Perm()); err == nil {
		_, err = temporary.Write(content)
	}
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("write vault page: %w", err)
	}
	latest, _, err := readPage(path)
	if err != nil {
		return "", err
	}
	if Revision(latest) != expectedRevision {
		return "", ErrConflict
	}
	if err = os.Rename(temporaryName, path); err != nil {
		return "", fmt.Errorf("replace vault page: %w", err)
	}
	if err = syncDirectory(filepath.Dir(path)); err != nil {
		// The page is already replaced, so reporting failure would make the editor's
		// retry a permanent conflict against its own save. Durability is best effort.
		slog.Warn("vault page synced without a directory flush", "path", path, "error", err)
	}
	return Revision(content), nil
}

// MovePage renames or moves one Markdown page without overwriting another
// vault entry. The content revision prevents a stale client from moving a page
// that Obsidian or another Mochil client changed after it was opened.
func (w *Writer) MovePage(relative, target, expectedRevision string) error {
	if expectedRevision == "" {
		return ErrInvalidPath
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	sourcePath, err := w.resolve(relative, true)
	if err != nil {
		return err
	}
	targetPath, err := w.resolve(target, true)
	if err != nil || sourcePath == targetPath {
		return ErrInvalidPath
	}
	content, _, err := readPage(sourcePath)
	if err != nil {
		return err
	}
	if Revision(content) != expectedRevision {
		return ErrConflict
	}
	// Link is the POSIX no-clobber primitive: unlike Rename it cannot silently
	// replace a page created concurrently by Obsidian.
	if err = os.Link(sourcePath, targetPath); errors.Is(err, fs.ErrExist) {
		return ErrExists
	} else if err != nil {
		return fmt.Errorf("move vault page: %w", err)
	}
	return completeLinkedMove(sourcePath, targetPath, syncDirectory)
}

// TrashPage removes a page from the visible vault by moving it into Obsidian's
// conventional hidden .trash directory. It remains recoverable outside
// Mochil and never replaces an existing trashed file.
func (w *Writer) TrashPage(relative, expectedRevision string) error {
	if expectedRevision == "" {
		return ErrInvalidPath
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	sourcePath, err := w.resolve(relative, true)
	if err != nil {
		return err
	}
	content, _, err := readPage(sourcePath)
	if err != nil {
		return err
	}
	if Revision(content) != expectedRevision {
		return ErrConflict
	}
	trash := filepath.Join(w.root, ".trash")
	if info, statErr := os.Lstat(trash); errors.Is(statErr, fs.ErrNotExist) {
		if err = os.Mkdir(trash, 0o750); err != nil {
			return fmt.Errorf("create vault trash: %w", err)
		}
		if err = syncDirectory(w.root); err != nil {
			return err
		}
	} else if statErr != nil || !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return ErrInvalidPath
	}
	base := filepath.Base(sourcePath)
	extension := filepath.Ext(base)
	stem := strings.TrimSuffix(base, extension)
	for suffix := 0; suffix < 10_000; suffix++ {
		name := base
		if suffix > 0 {
			name = fmt.Sprintf("%s-%d%s", stem, suffix, extension)
		}
		target := filepath.Join(trash, name)
		if err = os.Link(sourcePath, target); errors.Is(err, fs.ErrExist) {
			continue
		} else if err != nil {
			return fmt.Errorf("trash vault page: %w", err)
		}
		return completeLinkedMove(sourcePath, target, syncDirectory)
	}
	return ErrExists
}

func (w *Writer) resolve(relative string, markdown bool) (string, error) {
	relative = filepath.FromSlash(strings.TrimSpace(relative))
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", ErrInvalidPath
	}
	parts := strings.Split(clean, string(filepath.Separator))
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".") || strings.ContainsRune(part, '\x00') {
			return "", ErrInvalidPath
		}
	}
	if markdown && !document.IsMarkdownPath(clean) {
		return "", ErrInvalidPath
	}
	resolved := filepath.Join(w.root, clean)
	parent, err := filepath.EvalSymlinks(filepath.Dir(resolved))
	if err != nil || (parent != w.root && !strings.HasPrefix(parent, w.root+string(filepath.Separator))) {
		return "", ErrInvalidPath
	}
	return resolved, nil
}

func readPage(path string) ([]byte, fs.FileMode, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, 0, ErrInvalidPath
	}
	if info.Size() > MaximumPageBytes {
		return nil, 0, errors.New("vault page is too large")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("open vault page: %w", err)
	}
	defer file.Close()
	content, err := readPageContent(file)
	if err != nil {
		return nil, 0, fmt.Errorf("read vault page: %w", err)
	}
	return content, info.Mode(), nil
}

// A page may grow after Stat when an external editor is writing to the vault.
// Bound the actual read too, and never treat a truncated page as valid content.
func readPageContent(reader io.Reader) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, MaximumPageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > MaximumPageBytes {
		return nil, errors.New("vault page is too large")
	}
	return content, nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open vault directory: %w", err)
	}
	defer directory.Close()
	if err = directory.Sync(); err != nil {
		return fmt.Errorf("sync vault directory: %w", err)
	}
	return nil
}
