package artwork

import (
	"archive/zip"
	"bytes"
	"testing"
)

func testArchive(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range map[string]string{"OEBPS/chapter.xhtml": "text", "OEBPS/Images/cover.jpg": "cover-bytes", "OEBPS/Images/a.png": "other"} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = file.Write([]byte(body))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestFromArchiveReaderMatchesFromArchive(t *testing.T) {
	data := testArchive(t)
	fromBytes, mediaA, okA := FromArchive(data)
	fromReader, mediaB, okB := FromArchiveReader(bytes.NewReader(data), int64(len(data)))
	if !okA || !okB || string(fromBytes) != "cover-bytes" || string(fromReader) != "cover-bytes" || mediaA != mediaB {
		t.Fatalf("got %q/%q %v/%v", fromBytes, fromReader, okA, okB)
	}
}

func TestFromArchiveReaderRejectsNonArchives(t *testing.T) {
	if _, _, ok := FromArchiveReader(bytes.NewReader([]byte("not a zip")), 9); ok {
		t.Fatal("expected failure")
	}
}
