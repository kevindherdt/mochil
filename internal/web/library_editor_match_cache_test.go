package web

import (
	"path/filepath"
	"testing"
	"time"

	"mochil/internal/filefacts"
)

func TestEditorForgetsCachedMatchesWhenMatchInputsChange(t *testing.T) {
	store, err := filefacts.Open[bulkMatch](filepath.Join(t.TempDir(), "found.json"))
	if err != nil {
		t.Fatal(err)
	}
	modified := time.Unix(1700000000, 0)
	for _, id := range []string{"edited", "rated", "later"} {
		store.Put("lib", id, 10, modified, bulkMatch{})
	}
	server := &multiServer{matchFound: store}

	server.forgetEditorMatches(Library{ID: "lib"},
		map[string]map[string]string{"edited": {"title": "A new title"}},
		map[string][]string{"rated": {"rating"}},
		nil)

	if _, ok := store.Get("lib", "edited", 10, modified); ok {
		t.Fatal("a title change must forget the cached match")
	}
	if _, ok := store.Get("lib", "rated", 10, modified); !ok {
		t.Fatal("a rating change must not forget the cached match")
	}
	if _, ok := store.Get("lib", "later", 10, modified); !ok {
		t.Fatal("untouched books must keep their cached match")
	}
}
