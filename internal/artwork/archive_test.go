package artwork

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestFromArchivePrefersNamedCover(t *testing.T) {
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	for name, value := range map[string]string{"images/001.jpg": "page", "images/cover.png": "cover"} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = entry.Write([]byte(value))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	image, mediaType, ok := FromArchive(data.Bytes())
	if !ok || string(image) != "cover" || mediaType != "image/png" {
		t.Fatalf("cover = %q, %q, %v", image, mediaType, ok)
	}
}

func TestFromArchiveRejectsInvalidOrImagelessData(t *testing.T) {
	if _, _, ok := FromArchive([]byte("not a zip")); ok {
		t.Fatal("invalid archive produced a cover")
	}
}
