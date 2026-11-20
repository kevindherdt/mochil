package catalog

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type comicInfo struct {
	Title       string `xml:"Title"`
	Series      string `xml:"Series"`
	Number      string `xml:"Number"`
	Writer      string `xml:"Writer"`
	Summary     string `xml:"Summary"`
	LanguageISO string `xml:"LanguageISO"`
	Year        int    `xml:"Year"`
}

const maximumComicInfoBytes = 4 << 20

func applyComicInfo(book Book, reader io.Reader) (Book, error) {
	var comic comicInfo
	if err := xml.NewDecoder(io.LimitReader(reader, maximumComicInfoBytes)).Decode(&comic); err != nil {
		return book, err
	}
	if value := clean(comic.Title); value != "" {
		book.Title = value
	}
	book.Series = clean(comic.Series)
	book.Description = truncate(clean(comic.Summary), 4000)
	book.Language = clean(comic.LanguageISO)
	if value := clean(comic.Writer); value != "" {
		book.Authors = splitComicCreators(value)
		book.AuthorSort = append([]string(nil), book.Authors...)
	}
	if value := strings.TrimSpace(comic.Number); value != "" {
		book.SeriesIndex = parseSeriesIndex(value)
	}
	book.MetadataRead = true
	return book, nil
}

func readCBZ(path, rel string, info os.FileInfo) (Book, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return Book{}, err
	}
	defer zr.Close()
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	book := Book{
		ID:         bookID(rel),
		Path:       rel,
		Title:      base,
		Authors:    []string{"Unknown"},
		AuthorSort: []string{"Unknown"},
		Format:     "cbz",
		MediaType:  "application/vnd.comicbook+zip",
		Size:       info.Size(),
		Modified:   info.ModTime(),
	}
	var metadata *zip.File
	for _, entry := range zr.File {
		if isPageImage(entry.Name) && !entry.FileInfo().IsDir() {
			book.PageCount++
			pageType := pageMediaType(entry.Name)
			if book.PageCount == 1 {
				book.PageMediaType = pageType
			} else if book.PageMediaType != pageType {
				book.PageMediaType = ""
			}
		}
		if strings.EqualFold(filepath.Base(entry.Name), "ComicInfo.xml") {
			metadata = entry
		}
	}
	if book.PageCount == 0 {
		return Book{}, errors.New("CBZ contains no supported page images")
	}
	if metadata == nil {
		return book, nil
	}
	r, err := metadata.Open()
	if err != nil {
		// ComicInfo.xml is optional decoration for a readable archive: falling back
		// to the page-derived book beats dropping the comic from the catalogue.
		return book, nil
	}
	defer r.Close()
	comicBook, err := applyComicInfo(book, r)
	if err != nil {
		return book, nil
	}
	return comicBook, nil
}

func isPageImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func pageMediaType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

func splitComicCreators(value string) []string {
	parts := strings.Split(value, ";")
	creators := make([]string, 0, len(parts))
	for _, part := range parts {
		if name := clean(part); name != "" {
			creators = append(creators, name)
		}
	}
	if len(creators) == 0 {
		return []string{"Unknown"}
	}
	return creators
}
