package document

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

// `\bin` takes a byte count. Adding that count to the cursor overflowed for a huge
// number and panicked on a negative index instead of rejecting the document.
func TestRTFBinaryCountOverflowIsMalformedNotAPanic(t *testing.T) {
	text := `{\rtf1 before \bin9223372036854775807 after}`
	if _, err := decodeRTF([]byte(text), 1<<20); !errors.Is(err, ErrTextMalformed) {
		t.Fatalf("a binary count of the maximum integer = %v, want ErrTextMalformed", err)
	}
}

// A FIFO in a vault blocks on open forever. It used to be indexed as a document
// and then stall text indexing, with every later scan queued behind it.
func TestSpecialFilesAreNotIndexed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("named pipes are not available")
	}
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "blocked.md"), 0o600); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "real.md"), []byte("# Real note"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewVaultStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	items := store.Items()
	if len(items) != 1 || filepath.Base(items[0].Path) != "real.md" {
		t.Fatalf("indexed items = %#v, want only real.md", items)
	}
}
