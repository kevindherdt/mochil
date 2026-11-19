package document

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadTextIsBoundedAndFormatRestricted(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string][]byte{
		"note.md":    []byte("\xef\xbb\xbf# Hello\r\nWorld"),
		"large.txt":  []byte("too large"),
		"manual.pdf": []byte("%PDF"),
		"broken.txt": {0xff, 0xfe},
		"data.json":  []byte("{\"name\":\"Bobo\"}\r\n"),
	} {
		if err := os.WriteFile(filepath.Join(root, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	byTitle := map[string]Item{}
	for _, item := range store.Items() {
		byTitle[item.Title] = item
	}
	content, _, err := store.ReadText(byTitle["note"].ID, 100)
	if err != nil || string(content) != "# Hello\nWorld" {
		t.Fatalf("note = %q, %v", content, err)
	}
	if _, _, err := store.ReadText(byTitle["large"].ID, 3); !errors.Is(err, ErrTextTooLarge) {
		t.Fatalf("large error = %v", err)
	}
	if _, _, err := store.ReadText(byTitle["manual"].ID, 100); !errors.Is(err, ErrTextUnsupported) {
		t.Fatalf("PDF error = %v", err)
	}
	if _, _, err := store.ReadText(byTitle["broken"].ID, 100); !errors.Is(err, ErrTextEncoding) {
		t.Fatalf("encoding error = %v", err)
	}
	content, _, err = store.ReadText(byTitle["data"].ID, 100)
	if err != nil || string(content) != "{\"name\":\"Bobo\"}\n" {
		t.Fatalf("JSON = %q, %v", content, err)
	}
}
