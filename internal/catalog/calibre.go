package catalog

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maximumCalibreSidecarBytes = 8 << 20

// CalibreStore is a read-only filesystem adapter for a Calibre library. It
// treats metadata.opf and cover.jpg as authoritative without touching
// metadata.db or modifying the library.
type CalibreStore struct{ *Store }

func NewCalibreStore(library, cache string) *CalibreStore {
	return &CalibreStore{Store: NewFilesystemStore(library, cache, FilesystemOptions{FolderLayout: "metadata-only", Calibre: true})}
}

func readCalibrePublication(filePath, relativePath string, info os.FileInfo) (Book, error) {
	book, err := readPublication(filePath, relativePath, info)
	if err != nil {
		extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(filePath)), ".")
		switch extension {
		case "pdf", "mobi", "azw3":
			book = Book{ID: bookID(relativePath), Path: relativePath, Title: strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)), Authors: []string{"Unknown"}, AuthorSort: []string{"Unknown"}, Format: extension, MediaType: mediaTypeForPath(filePath), Size: info.Size(), Modified: info.ModTime()}
		default:
			return Book{}, err
		}
	}
	sidecar, err := os.Open(filepath.Join(filepath.Dir(filePath), "metadata.opf"))
	if err != nil {
		return Book{}, errors.New("Calibre book is missing metadata.opf")
	}
	defer sidecar.Close()
	var pkg packageXML
	if err := xml.NewDecoder(io.LimitReader(sidecar, maximumCalibreSidecarBytes)).Decode(&pkg); err != nil {
		return Book{}, errors.New("Calibre metadata.opf is invalid")
	}
	if title := clean(first(pkg.Metadata.Titles)); title != "" {
		book.Title = title
	}
	book.Authors, book.AuthorSort = epubCreators(pkg)
	book.Language = clean(first(pkg.Metadata.Languages))
	book.Description = truncate(clean(first(pkg.Metadata.Descriptions)), 4000)
	book.Identifier = firstIdentifier(pkg)
	book = withExtraMetadata(book, pkg)
	for _, meta := range pkg.Metadata.Metas {
		switch strings.ToLower(meta.Name) {
		case "calibre:series":
			book.Series = clean(meta.Content)
		case "calibre:series_index":
			book.SeriesIndex = parseSeriesIndex(meta.Content)
		}
	}
	book.MetadataRead = true
	if coverInfo, coverErr := os.Stat(filepath.Join(filepath.Dir(filePath), "cover.jpg")); coverErr == nil {
		book.CoverModified = coverInfo.ModTime()
	}
	return book, nil
}

func (s *CalibreStore) Cover(_ context.Context, book Book) ([]byte, error) {
	publication, err := s.safeAbsolutePath(book)
	if err != nil {
		return nil, err
	}
	coverPath := filepath.Join(filepath.Dir(publication), "cover.jpg")
	info, err := os.Lstat(coverPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > maxRemoteCoverBytes {
		return nil, errors.New("Calibre cover is unavailable")
	}
	file, err := os.Open(coverPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxRemoteCoverBytes+1))
	if err != nil || len(data) > maxRemoteCoverBytes {
		return nil, errors.New("Calibre cover is too large")
	}
	return data, nil
}

func calibreFormatRank(name string) int {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".epub":
		return 0
	case ".azw3":
		return 1
	case ".mobi":
		return 2
	case ".pdf":
		return 3
	case ".cbz":
		return 4
	default:
		return 100
	}
}
