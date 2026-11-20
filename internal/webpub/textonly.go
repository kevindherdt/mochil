package webpub

import (
	"archive/zip"
	"io"
	"path"
	"strings"
)

// WriteTextOnly copies an EPUB without its audio, for a device that streams the narration of a
// read-along book instead of storing it. Entries are copied as they are stored, without being
// unpacked, so a book of hundreds of megabytes costs little to send. It reports how many audio
// files were left out; a book without audio is copied whole.
func WriteTextOnly(w io.Writer, reader io.ReaderAt, size int64) (int, error) {
	archive, _, opfPath, pkg, err := openPackage(reader, size)
	if err != nil {
		return 0, err
	}
	base := path.Dir(opfPath)
	skip := map[string]bool{}
	for _, item := range pkg.Manifest {
		if strings.HasPrefix(strings.ToLower(item.MediaType), "audio/") {
			skip[resolveIn(base, item.Href)] = true
		}
	}
	// Audio the narration points at goes too, even where the book does not declare its type.
	if overlay, err := BuildOverlay(reader, size); err == nil {
		for _, audio := range overlay.Audio {
			skip[audio] = true
		}
	}
	out := zip.NewWriter(w)
	for _, file := range archive.File {
		if skip[file.Name] {
			continue
		}
		raw, err := file.OpenRaw()
		if err != nil {
			return 0, err
		}
		header := file.FileHeader
		entry, err := out.CreateRaw(&header)
		if err != nil {
			return 0, err
		}
		if _, err := io.Copy(entry, raw); err != nil {
			return 0, err
		}
	}
	return len(skip), out.Close()
}
