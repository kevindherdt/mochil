package video

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnavailable = errors.New("video is unavailable")

// Open returns an indexed regular file only when its current resolved path is
// still contained by the configured root. This revalidates the path at request
// time so a file cannot be replaced with a symlink after a scan.
func (s *Store) Open(id string) (*os.File, Item, error) {
	item, ok := s.Item(id)
	if !ok {
		return nil, Item{}, ErrUnavailable
	}
	root, err := filepath.Abs(s.root)
	if err != nil {
		return nil, Item{}, ErrUnavailable
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, Item{}, ErrUnavailable
	}
	target := filepath.Join(root, filepath.FromSlash(item.Path))
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil || !pathWithin(resolvedRoot, resolvedTarget) {
		return nil, Item{}, ErrUnavailable
	}
	file, err := os.Open(resolvedTarget)
	if err != nil {
		return nil, Item{}, ErrUnavailable
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return nil, Item{}, ErrUnavailable
	}
	current, supported := Identify(s.libraryID, item.Path, info.Size(), info.ModTime())
	if !supported || current.Revision != item.Revision {
		file.Close()
		return nil, Item{}, ErrUnavailable
	}
	return file, item, nil
}

func pathWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
