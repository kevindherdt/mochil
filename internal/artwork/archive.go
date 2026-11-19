// Package artwork interprets publication artwork without depending on HTTP
// routes, storage adapters, or cache implementations.
package artwork

import (
	"archive/zip"
	"bytes"
	"io"
	"mime"
	"path"
	"strings"
)

const maximumArchiveCoverBytes = 20 << 20

// FromArchive returns the most likely cover image from an EPUB/CBZ-style ZIP.
// Explicitly named covers win; otherwise the first supported image is used.
func FromArchive(data []byte) ([]byte, string, bool) {
	return FromArchiveReader(bytes.NewReader(data), int64(len(data)))
}

// FromArchiveReader is FromArchive for a seekable source. Only the ZIP index
// and the cover entry are read, so a large publication is never loaded whole.
func FromArchiveReader(source io.ReaderAt, size int64) ([]byte, string, bool) {
	reader, err := zip.NewReader(source, size)
	if err != nil {
		return nil, "", false
	}
	var candidates []*zip.File
	for _, file := range reader.File {
		extension := strings.ToLower(path.Ext(file.Name))
		if extension != ".jpg" && extension != ".jpeg" && extension != ".png" && extension != ".webp" {
			continue
		}
		if strings.Contains(strings.ToLower(file.Name), "cover") {
			candidates = append([]*zip.File{file}, candidates...)
		} else {
			candidates = append(candidates, file)
		}
	}
	if len(candidates) == 0 {
		return nil, "", false
	}
	input, err := candidates[0].Open()
	if err != nil {
		return nil, "", false
	}
	defer input.Close()
	if candidates[0].UncompressedSize64 > maximumArchiveCoverBytes {
		return nil, "", false
	}
	image := make([]byte, 0, candidates[0].UncompressedSize64+1)
	buffer := bytes.NewBuffer(image)
	if _, err := io.Copy(buffer, io.LimitReader(input, maximumArchiveCoverBytes+1)); err != nil || buffer.Len() > maximumArchiveCoverBytes {
		return nil, "", false
	}
	image = buffer.Bytes()
	return image, mime.TypeByExtension(path.Ext(candidates[0].Name)), true
}
