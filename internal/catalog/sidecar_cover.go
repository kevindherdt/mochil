package catalog

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SidecarCoverBackend finds a cover image lying next to a publication, for
// books whose file carries no cover of its own (Calibre leaves cover.jpg beside
// the book, and other tools use <book name>.jpg).
type SidecarCoverBackend interface {
	SidecarCover(ctx context.Context, book Book) ([]byte, error)
}

func SidecarCoverCapability(backend Backend) (SidecarCoverBackend, bool) {
	provider, ok := unwrapBackend(backend).(SidecarCoverBackend)
	return provider, ok
}

var sidecarCoverExtensions = []string{".jpg", ".jpeg", ".png"}

// SidecarCover reads cover.jpg|jpeg|png or <book name>.jpg|jpeg|png beside the
// publication. Symbolic links, oversized files and anything outside the
// library are refused.
func (s *Store) SidecarCover(_ context.Context, book Book) ([]byte, error) {
	publication, err := s.safeAbsolutePath(book)
	if err != nil {
		return nil, err
	}
	directory := filepath.Dir(publication)
	stem := strings.TrimSuffix(filepath.Base(publication), filepath.Ext(publication))
	var names []string
	for _, extension := range sidecarCoverExtensions {
		names = append(names, stem+extension)
	}
	for _, extension := range sidecarCoverExtensions {
		names = append(names, "cover"+extension)
	}
	for _, name := range names {
		path := filepath.Join(directory, name)
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > maxRemoteCoverBytes {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(file, maxRemoteCoverBytes+1))
		file.Close()
		if err == nil && len(data) <= maxRemoteCoverBytes {
			return data, nil
		}
	}
	return nil, errors.New("no cover image beside the publication")
}
