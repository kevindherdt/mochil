package document

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadRTFExtractsTextAndSkipsNonContentDestinations(t *testing.T) {
	root := t.TempDir()
	content := `{\rtf1\ansi\deff0{\fonttbl{\f0 Arial;}}{\info{\author Private author}}` +
		`\f0 Meeting\tab notes\par Caf\'e9 and \u26085?\par ` +
		`{\*\unknown hidden value}{\pict secret-image-data}Visible \{text\}.}`
	if err := os.WriteFile(filepath.Join(root, "Meeting.rtf"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	item := store.Items()[0]
	text, _, err := store.ReadText(item.ID, 4096)
	if err != nil {
		t.Fatal(err)
	}
	want := "Meeting\tnotes\nCafé and 日\nVisible {text}."
	if string(text) != want {
		t.Fatalf("RTF text = %q, want %q", text, want)
	}
	if item.Format != "rtf" || item.MediaType != "application/rtf" || !TextReadable(item.Format) {
		t.Fatalf("RTF item = %#v", item)
	}
}

func TestDecodeRTFRejectsMalformedAndBoundsOutput(t *testing.T) {
	for _, malformed := range [][]byte{[]byte("plain text"), []byte(`{\rtf1 unclosed`), []byte(`{\rtf1 bad\'xz}`)} {
		if _, err := decodeRTF(malformed, 1024); !errors.Is(err, ErrTextMalformed) {
			t.Fatalf("malformed %q error = %v", malformed, err)
		}
	}
	if _, err := decodeRTF([]byte(`{\rtf1 `+strings.Repeat("x", 40)+`}`), 16); !errors.Is(err, ErrTextTooLarge) {
		t.Fatalf("oversized text error = %v", err)
	}
}

func TestRTFContentIsIncludedInSearch(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Archive.rtf"), []byte(`{\rtf1\ansi Archived jellyfish notes}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore("notes", root, filepath.Join(t.TempDir(), "documents.json"))
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	index := NewTextIndex(store, filepath.Join(t.TempDir(), "text.json"))
	matches, coverage, err := index.Search("jellyfish")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || coverage.Indexed != 1 || !coverage.Complete {
		t.Fatalf("matches = %#v, coverage = %#v", matches, coverage)
	}
	item, ok := identify("notes", "Archive.rtf", 1, time.Unix(1, 0), false)
	if !ok || item.Format != "rtf" {
		t.Fatalf("identify RTF = %#v, %v", item, ok)
	}
}

func TestDecodeRTFEmitsPunctuationControlWords(t *testing.T) {
	got, err := decodeRTF([]byte(`{\rtf1 a\emdash b\endash c \bullet d\lquote e\rquote f \ldblquote g\rdblquote h}`), 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a—b–c •d‘e’f “g”h" {
		t.Fatalf("punctuation = %q", got)
	}
	plain, err := decodeRTF([]byte(`{\rtf1 a\emdash b \bullet c}`), 1024)
	if err != nil || string(plain) != "a—b •c" {
		t.Fatalf("dash and bullet = %q, %v", plain, err)
	}
}

func TestDecodeRTFCombinesSurrogatePairs(t *testing.T) {
	// \u-10179 and \u-8704 are the UTF-16 pair for U+1F600, written the way Word
	// stores a supplementary character: two \u units that only mean something together.
	got, err := decodeRTF([]byte(`{\rtf1 \u-10179?\u-8704?}`), 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "\U0001F600" {
		t.Fatalf("surrogate pair = %q, want %q", got, "\U0001F600")
	}
	// An unpaired high surrogate must not swallow the text after it.
	lone, err := decodeRTF([]byte(`{\rtf1 \u-10179?x}`), 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(lone) != "\uFFFDx" {
		t.Fatalf("lone high surrogate = %q", lone)
	}
}

func TestDecodeRTFSkipsBinaryPayloads(t *testing.T) {
	// \bin4 covers "ab}c": the brace inside the payload must not close the group.
	got, err := decodeRTF([]byte(`{\rtf1{\pict\bin4 ab}c}tail}`), 1024)
	if err != nil || string(got) != "tail" {
		t.Fatalf("binary payload = %q, %v", got, err)
	}
	if _, err := decodeRTF([]byte(`{\rtf1{\pict\bin9999 short}`), 1024); !errors.Is(err, ErrTextMalformed) {
		t.Fatalf("a truncated binary payload must be refused, got %v", err)
	}
}

func TestDecodeRTFHonoursTheDeclaredCodePage(t *testing.T) {
	// "Привет" in Windows-1251, the code page the document declares.
	got, err := decodeRTF([]byte(`{\rtf1\ansi\ansicpg1251 \'cf\'f0\'e8\'e2\'e5\'f2}`), 1024)
	if err != nil || string(got) != "Привет" {
		t.Fatalf("cp1251 = %q, %v", got, err)
	}
	// Without a declaration the historical Windows-1252 reading must not change.
	if plain, err := decodeRTF([]byte(`{\rtf1 Caf\'e9}`), 1024); err != nil || string(plain) != "Café" {
		t.Fatalf("default code page = %q, %v", plain, err)
	}
}
