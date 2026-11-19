package annotationnotes

import (
	"errors"
	"strings"
	"testing"

	"mochil/internal/annotations"
)

func TestRenderCreatesReadableMarkdownAndOmitsTombstones(t *testing.T) {
	index := 2.5
	note, err := Render(Publication{Title: "A Book", Authors: []string{"A. Writer"}, Series: "Saga", SeriesIndex: &index}, []annotations.Annotation{
		{ID: "one", Kind: "highlight", Quote: "first line\nsecond line", Note: "My thought", Chapter: "Chapter 3", UpdatedAt: 1_700_000_000_000},
		{ID: "gone", Kind: "highlight", Quote: "secret", Deleted: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if note.Title != "Annotations — A Book" || !strings.Contains(note.Content, "**Series:** Saga · 2.5") || !strings.Contains(note.Content, "> first line\n> second line") || !strings.Contains(note.Content, "My thought") || strings.Contains(note.Content, "secret") {
		t.Fatalf("unexpected note: %#v", note)
	}
}

func TestRenderRejectsEmptyAndOversizedExports(t *testing.T) {
	if _, err := Render(Publication{Title: "Empty"}, []annotations.Annotation{{Deleted: true}}); !errors.Is(err, ErrEmpty) {
		t.Fatalf("empty error = %v", err)
	}
	items := make([]annotations.Annotation, 300)
	for index := range items {
		items[index] = annotations.Annotation{ID: "item", Kind: "note", Note: strings.Repeat("x", 8192)}
	}
	if _, err := Render(Publication{Title: "Large"}, items); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("large error = %v", err)
	}
}

func TestRenderBoundsMultibyteTitleWithoutBreakingUTF8(t *testing.T) {
	note, err := Render(Publication{Title: strings.Repeat("🙂", 100)}, []annotations.Annotation{{ID: "one", Kind: "bookmark"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(note.Title) > 300 || !strings.HasPrefix(note.Title, "Annotations — ") {
		t.Fatalf("title = %q (%d bytes)", note.Title, len(note.Title))
	}
}
