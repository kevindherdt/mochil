package web

import (
	"crypto/md5"
	"encoding/hex"
	"io"

	"mochil/internal/acquisition"
	"mochil/internal/catalog"
)

// koSyncPartialMD5 is KOReader's default "binary" document id: the MD5 of twelve
// 1 KiB samples taken at offsets 0, 1024, 4096, 16384 and so on (step<<2i).
func koSyncPartialMD5(reader io.ReaderAt) string {
	hash := md5.New()
	buffer := make([]byte, 1024)
	for sample, offset := 0, int64(0); sample < 12; sample++ {
		n, err := reader.ReadAt(buffer, offset)
		if n > 0 {
			_, _ = hash.Write(buffer[:n])
		}
		if err != nil {
			break
		}
		if sample == 0 {
			offset = 1024
		} else {
			offset *= 4
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func koSyncFilenameMD5(filename string) string {
	sum := md5.Sum([]byte(filename))
	return hex.EncodeToString(sum[:])
}

func publicationDigest(cache *acquisition.PublicationCache, libraryID string, book catalog.Book) (string, bool) {
	file, _, ok := cache.Open(libraryID, book)
	if !ok {
		return "", false
	}
	defer file.Close()
	return koSyncPartialMD5(file), true
}
