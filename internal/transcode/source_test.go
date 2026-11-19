package transcode

import "testing"

func TestKeySeparatesMediaKindsRevisionsAndVariants(t *testing.T) {
	base := Source{LibraryID: "library", ItemID: "item", Revision: "one", Kind: "audio"}
	first := Key(base, "aac-192k")
	if len(first) != 32 {
		t.Fatalf("key = %q", first)
	}
	changed := []string{
		Key(Source{LibraryID: "library", ItemID: "item", Revision: "two", Kind: "audio"}, "aac-192k"),
		Key(Source{LibraryID: "library", ItemID: "item", Revision: "one", Kind: "video"}, "aac-192k"),
		Key(base, "aac-128k"),
	}
	for _, key := range changed {
		if key == first {
			t.Fatalf("distinct source produced %q", key)
		}
	}
}

func TestKeyRejectsIncompleteSource(t *testing.T) {
	if got := Key(Source{LibraryID: "library", ItemID: "item", Kind: "audio"}, "aac"); got != "" {
		t.Fatalf("invalid key = %q", got)
	}
}
