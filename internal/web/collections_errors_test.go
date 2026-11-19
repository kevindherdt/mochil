package web

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCollectionMemberErrorsAreDistinguishable(t *testing.T) {
	store, err := newCollectionStore(filepath.Join(t.TempDir(), "collections.json"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := store.create("a", "Mine", "books", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.member("a", "missing", collectionMember{Type: "book", LibraryID: "books", BookID: "1"}, true); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unknown collection: %v", err)
	}
	if err := store.member("a", c.ID, collectionMember{Type: "book", LibraryID: "other", BookID: "1"}, true); !errors.Is(err, errWrongLibrary) {
		t.Fatalf("book of another library: %v", err)
	}
}
