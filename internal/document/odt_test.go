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

func TestReadODTExtractsBoundedDocumentText(t *testing.T) {
	root := t.TempDir()
	writeTestODT(t, filepath.Join(root, "Notes.odt"), `<office:document-content xmlns:office="urn:office" xmlns:text="urn:text"><office:body><office:text><text:h>Meeting notes</text:h><text:p>First <text:span>item</text:span><text:tab/>done</text:p><text:p>Next<text:line-break/>line</text:p></office:text></office:body></office:document-content>`)
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	content, _, err := store.ReadText(item.ID, 1024)
	if err != nil || string(content) != "Meeting notes\nFirst item\tdone\nNext\nline" {
		t.Fatalf("ODT text = %q, %v", content, err)
	}
	if item.Format != "odt" || item.MediaType != "application/vnd.oasis.opendocument.text" || !TextReadable(item.Format) {
		t.Fatalf("ODT item = %#v", item)
	}
}

func TestReadODTRejectsMalformedAndOversizedText(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Broken.odt"), []byte("not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestODT(t, filepath.Join(root, "Large.odt"), `<document><p>`+strings.Repeat("x", 200)+`</p></document>`)
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	byTitle := map[string]Item{}
	for _, item := range store.Items() {
		byTitle[item.Title] = item
	}
	if _, _, err := store.ReadText(byTitle["Broken"].ID, 1024); !errors.Is(err, ErrTextMalformed) {
		t.Fatalf("malformed ODT error = %v", err)
	}
	if _, _, err := store.ReadText(byTitle["Large"].ID, 32); !errors.Is(err, ErrTextTooLarge) {
		t.Fatalf("large ODT error = %v", err)
	}
}

func writeTestODT(t *testing.T, path, contentXML string) {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entry, err := archive.Create("content.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entry.Write([]byte(contentXML)); err != nil {
		t.Fatal(err)
	}
	if err = archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReadODTPreservesRepeatedSpaces(t *testing.T) {
	root := t.TempDir()
	writeTestODT(t, filepath.Join(root, "Spaces.odt"), `<office:document-content xmlns:office="urn:office" xmlns:text="urn:text"><office:body><office:text><text:p>a<text:s text:c="4"/>b<text:s/>c</text:p></office:text></office:body></office:document-content>`)
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	content, _, err := store.ReadText(store.Items()[0].ID, 1024)
	if err != nil || string(content) != "a    b c" {
		t.Fatalf("ODT spaces = %q, %v", content, err)
	}
}
