package reading

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

// Every individual write is within the locator limits, but the record keeps the
// union of what clients send. Without checking the union, successive writes grew
// one position past the limits until the whole state file exceeded the size the
// loader accepts and every account's progress stopped being saved.
func TestMergedLocatorsCannotGrowPastTheLimit(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	base := Position{LibraryID: "books", BookID: "one", Fraction: .3}
	position, err := store.Put("reader", base, 0)
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= MaximumLocatorTypes; index++ {
		next := base
		next.Fraction = position.Fraction
		next.Locators = map[string]string{fmt.Sprintf("kind%d", index): "value"}
		position, err = store.Put("reader", next, position.Revision)
		if err != nil {
			t.Fatalf("locator %d: %v", index, err)
		}
	}
	if len(position.Locators) != MaximumLocatorTypes {
		t.Fatalf("locators before the last write: %d", len(position.Locators))
	}

	overflow := base
	overflow.Fraction = position.Fraction
	overflow.Locators = map[string]string{"last": "value"}
	if _, err := store.Put("reader", overflow, position.Revision); !errors.Is(err, ErrInvalid) {
		t.Fatalf("a ninth locator kind = %v, want ErrInvalid", err)
	}
	kept, ok := store.Get("reader", "books", "one")
	if !ok || kept.Revision != position.Revision || len(kept.Locators) != len(position.Locators) {
		t.Fatalf("the rejected write changed the record: %#v", kept)
	}
}
