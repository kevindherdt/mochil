package vaultwriter

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// MaximumAttachmentBytes bounds one pasted or uploaded image.
const MaximumAttachmentBytes = 12 << 20

var ErrUnsupportedAttachment = errors.New("unsupported attachment type")

var imageTypes = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif", ".webp": "image/webp",
}

func safeAttachmentName(name string) (stem, extension string, err error) {
	name = filepath.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	extension = strings.ToLower(filepath.Ext(name))
	if _, ok := imageTypes[extension]; !ok {
		return "", "", ErrUnsupportedAttachment
	}
	stem = strings.TrimSuffix(name, filepath.Ext(name))
	stem = strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`/\:*?"<>|#^[]`, r) {
			return '-'
		}
		return r
	}, stem)
	stem = strings.TrimLeft(strings.TrimSpace(stem), ".")
	if stem == "" || len(stem) > 120 {
		return "", "", ErrInvalidPath
	}
	return stem, extension, nil
}

// CreateAttachment stores an image at the top of the vault (Obsidian's default
// attachment location) and returns the file name it was given. An existing file
// is never replaced: a number is added to the name instead.
func (w *Writer) CreateAttachment(name string, data []byte) (string, error) {
	stem, extension, err := safeAttachmentName(name)
	if err != nil {
		return "", err
	}
	if len(data) == 0 || len(data) > MaximumAttachmentBytes {
		return "", ErrInvalidPath
	}
	if http.DetectContentType(data) != imageTypes[extension] {
		return "", ErrUnsupportedAttachment
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for attempt := 0; attempt < 1000; attempt++ {
		candidate := stem + extension
		if attempt > 0 {
			candidate = fmt.Sprintf("%s %d%s", stem, attempt, extension)
		}
		path, err := w.resolve(candidate, false)
		if err != nil {
			return "", err
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
		if errors.Is(err, fs.ErrExist) {
			continue
		} else if err != nil {
			return "", fmt.Errorf("create vault attachment: %w", err)
		}
		_, err = file.Write(data)
		if err == nil {
			err = file.Sync()
		}
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
			return "", fmt.Errorf("create vault attachment: %w", err)
		}
		return candidate, syncDirectory(filepath.Dir(path))
	}
	return "", ErrExists
}

// MoveFolder renames or moves a folder with everything in it. The destination
// is checked first, so an existing folder or page is never replaced.
func (w *Writer) MoveFolder(relative, target string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	sourcePath, err := w.resolve(relative, false)
	if err != nil {
		return err
	}
	targetPath, err := w.resolve(target, false)
	if err != nil || sourcePath == targetPath || strings.HasPrefix(targetPath, sourcePath+string(filepath.Separator)) {
		return ErrInvalidPath
	}
	info, err := os.Lstat(sourcePath)
	if err != nil || !info.IsDir() {
		return ErrInvalidPath
	}
	if _, err = os.Lstat(targetPath); err == nil {
		return ErrExists
	}
	// rename(2) refuses to replace a folder that has anything in it, so a
	// concurrent change can at worst replace an empty folder.
	if err = os.Rename(sourcePath, targetPath); err != nil {
		return fmt.Errorf("move vault folder: %w", err)
	}
	if err = syncDirectory(filepath.Dir(targetPath)); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(sourcePath))
}
