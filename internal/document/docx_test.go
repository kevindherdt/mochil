package document

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadDOCXExtractsBoundedParagraphText(t *testing.T) {
	root := t.TempDir()
	writeTestDOCX(t, filepath.Join(root, "Report.docx"), `<w:document xmlns:w="urn:word"><w:body><w:p><w:r><w:t>Quarterly </w:t></w:r><w:r><w:t>report</w:t></w:r></w:p><w:p><w:r><w:t>Revenue</w:t><w:tab/><w:t>42</w:t></w:r></w:p></w:body></w:document>`)
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	content, _, err := store.ReadText(item.ID, 1024)
	if err != nil || string(content) != "Quarterly report\nRevenue\t42" {
		t.Fatalf("DOCX text = %q, %v", content, err)
	}
	if item.Format != "docx" || !TextReadable(item.Format) {
		t.Fatalf("DOCX item = %#v", item)
	}
}

func TestReadDOCXRejectsMalformedAndOversizedText(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Broken.docx"), []byte("not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestDOCX(t, filepath.Join(root, "Large.docx"), `<w:document xmlns:w="urn:word"><w:body><w:p><w:r><w:t>`+strings.Repeat("x", 200)+`</w:t></w:r></w:p></w:body></w:document>`)
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	byTitle := map[string]Item{}
	for _, item := range store.Items() {
		byTitle[item.Title] = item
	}
	if _, _, err := store.ReadText(byTitle["Broken"].ID, 1024); !errors.Is(err, ErrTextMalformed) {
		t.Fatalf("malformed DOCX error = %v", err)
	}
	if _, _, err := store.ReadText(byTitle["Large"].ID, 32); !errors.Is(err, ErrTextTooLarge) {
		t.Fatalf("large DOCX error = %v", err)
	}
}

func writeTestDOCX(t *testing.T, path, documentXML string) {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entry, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entry.Write([]byte(documentXML)); err != nil {
		t.Fatal(err)
	}
	if err = archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}
