package document

import (
	"testing"
	"time"
)

// SVG is a vault attachment format: Obsidian notes embed drawn diagrams, and a format the server refuses to list
// renders as nothing at all (a real note lost its only illustration to this).
func TestIdentifyListsSVGAsAVaultAsset(t *testing.T) {
	item, ok := identify("vault", "Personal/Diagram.svg", 128, time.Unix(0, 0), true)
	if !ok || item.MediaType != "image/svg+xml" {
		t.Fatalf("svg identify = %+v, ok=%v", item, ok)
	}
	// Only vaults take assets; a book library must not start listing random images.
	if _, ok = identify("books", "Diagram.svg", 128, time.Unix(0, 0), false); ok {
		t.Fatal("svg listed for a non-vault library")
	}
}
