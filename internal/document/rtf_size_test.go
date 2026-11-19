package document

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RTF is uncompressed, so a small document with one embedded picture is far
// larger than its text. The encoded size must not decide whether it is readable.
func TestReadRTFWithAPictureLargerThanTheTextLimit(t *testing.T) {
	root := t.TempDir()
	// A \bin payload is skipped by the decoder but counts towards the file size.
	payload := strings.Repeat("x", 2<<20)
	body := "{\\rtf1 tiny text {\\*\\bin" + itoa(len(payload)) + " " + payload + "}}"
	if err := os.WriteFile(filepath.Join(root, "Picture.rtf"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	content, _, err := store.ReadText(item.ID, DefaultTextLimit)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := strings.TrimSpace(string(content)); got != "tiny text" {
		t.Fatalf("text = %q, want the document's text", got)
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}
