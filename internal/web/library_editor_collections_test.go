package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func collectionCall(t *testing.T, server *multiServer, body string) (int, map[string]any) {
	t.Helper()
	response := editorCall(server, http.MethodPost, "/api/metadata/editor/collections", body)
	var reply map[string]any
	_ = json.Unmarshal(response.Body.Bytes(), &reply)
	return response.Code, reply
}

func collectionsOf(t *testing.T, server *multiServer, id string) []string {
	t.Helper()
	for _, row := range listEditor(t, server, "").Items {
		if row.ID == id {
			return row.Collections
		}
	}
	t.Fatalf("no book %s", id)
	return nil
}

func TestPickedBooksCanBeAddedToAndRemovedFromACollectionInOneGo(t *testing.T) {
	server := editorTestServer(t)
	server.collections, _ = newCollectionStore("")
	code, reply := collectionCall(t, server, `{"library":"books","name":"Favourites","ids":["a1","a2","n1"]}`)
	id, _ := reply["collection"].(string)
	if code != http.StatusOK || reply["changed"] != float64(3) || id == "" {
		t.Fatalf("create: %d %v", code, reply)
	}
	if got := collectionsOf(t, server, "a2"); len(got) != 1 || got[0] != "Favourites" {
		t.Fatalf("a2 is in %v", got)
	}
	if got := collectionsOf(t, server, "s1"); len(got) != 0 {
		t.Fatalf("s1 is in %v", got)
	}
	if _, reply = collectionCall(t, server, `{"library":"books","collection":"`+id+`","ids":["n1","s1"]}`); reply["changed"] != float64(1) {
		t.Fatalf("adding one that is already in changes only the new one: %v", reply)
	}
	if _, reply = collectionCall(t, server, `{"library":"books","collection":"`+id+`","ids":["a1","a2"],"remove":true}`); reply["changed"] != float64(2) {
		t.Fatalf("remove: %v", reply)
	}
	if got := collectionsOf(t, server, "a1"); len(got) != 0 {
		t.Fatalf("a1 is still in %v", got)
	}
	if rows := listEditor(t, server, "&collection="+id).Items; len(rows) != 2 {
		t.Fatalf("filtering by the collection gives %d books, want n1 and s1", len(rows))
	}
}

func TestACollectionThatFillsItselfCannotBeChangedByHand(t *testing.T) {
	server := editorTestServer(t)
	server.collections, _ = newCollectionStore("")
	smart, _ := server.collections.create(server.readingAccount(httptest.NewRequest(http.MethodGet, "/", nil)), "Dune things", "books", &collectionRule{Query: "dune"})
	if code, _ := collectionCall(t, server, `{"library":"books","collection":"`+smart.ID+`","ids":["n1"]}`); code != http.StatusConflict {
		t.Fatalf("code = %d, want 409", code)
	}
	if got := collectionsOf(t, server, "a1"); len(got) != 1 || got[0] != "Dune things" {
		t.Fatalf("a smart collection still shows on its books: %v", got)
	}
}

func TestACollectionNeedsBooksAndAName(t *testing.T) {
	server := editorTestServer(t)
	server.collections, _ = newCollectionStore("")
	for _, body := range []string{`{"library":"books","name":"x","ids":[]}`, `{"library":"books","name":" ","ids":["a1"]}`, `{"library":"books","name":"` + strings.Repeat("n", 81) + `","ids":["a1"]}`} {
		if code, _ := collectionCall(t, server, body); code != http.StatusBadRequest {
			t.Fatalf("%s: code = %d", body, code)
		}
	}
}

func TestEditorShowsWhereEachBookStandsWithTheHardcoverSync(t *testing.T) {
	server := editorTestServer(t)
	server.hardcoverMatches.put(hardcoverMatch{Library: "books", Book: "a1", Title: "Dune", HardcoverID: "1", HardcoverTitle: "Dune", HardcoverAuthors: []string{"Frank Herbert"}, How: "matched", Result: "added", At: 1700000000})
	server.hardcoverMatches.put(hardcoverMatch{Library: "books", Book: "a2", Title: "Dune Messiah", HardcoverID: "2", HardcoverTitle: "Dune Messiah", How: "linked", Doubt: "Several Hardcover books have this title and author.", At: 1700000000})
	server.hardcoverSkipped.add(skippedBook{Library: "books", Book: "n1", Title: "Neuromancer"})
	server.hardcoverIgnored.add(ignoredBook{Library: "books", Book: "s1", Title: "Roxy", Reason: ignoredRemoved})
	rows := map[string]editorRow{}
	for _, row := range listEditor(t, server, "").Items {
		rows[row.ID] = row
	}
	want := map[string]string{"a1": "synced", "a2": "doubt", "n1": "nomatch", "s1": "ignored", "u1": ""}
	for id, state := range want {
		if rows[id].Hardcover != state {
			t.Errorf("%s: state %q, want %q", id, rows[id].Hardcover, state)
		}
	}
	if !strings.Contains(rows["a1"].HardcoverNote, "Matched by its title and author to Dune by Frank Herbert") || !strings.Contains(rows["a2"].HardcoverNote, "Linked by hand") || !strings.Contains(rows["a2"].HardcoverNote, "Several Hardcover books") {
		t.Errorf("notes: %q / %q", rows["a1"].HardcoverNote, rows["a2"].HardcoverNote)
	}
	if got := listEditor(t, server, "&filter=hardcover_doubt").Items; len(got) != 1 || got[0].ID != "a2" {
		t.Errorf("the doubt filter gives %v", got)
	}
	if counts := listEditor(t, server, "").Counts; counts["hardcover_ignored"] != 1 || counts["hardcover_doubt"] != 1 || counts["hardcover_nomatch"] != 1 {
		t.Errorf("counts = %v", counts)
	}
	if first := listEditor(t, server, "&sort=hardcover").Items[0]; first.Hardcover != "doubt" {
		t.Errorf("sorted by Hardcover the first is %q, want doubt (a-z, empty last)", first.Hardcover)
	}
	server.cfg.Modules = map[string]bool{"hardcover": false}
	if row := listEditor(t, server, "&q=dune").Items[0]; row.Hardcover != "" {
		t.Errorf("with the Hardcover module off nothing is shown: %q", row.Hardcover)
	}
}

// Creating a collection OF BOOKS must be one durable write. Creating the empty collection and filling it in a second save
// left an empty twin behind whenever that second write failed, while the caller was told the save failed and the name was
// free again, so the retry produced a duplicate.
func TestCreatingACollectionOfBooksIsOneDurableWrite(t *testing.T) {
	server := editorTestServer(t)
	store, err := newCollectionStore(filepath.Join(t.TempDir(), "collections.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.collections = store
	saves := 0
	collectionSaveHook = func() error {
		saves++
		if saves == 2 {
			return errors.New("disk full")
		}
		return nil
	}
	defer func() { collectionSaveHook = nil }()

	code, reply := collectionCall(t, server, `{"library":"books","name":"Favourites","ids":["a1","a2"]}`)
	if code != http.StatusOK || reply["changed"] != float64(2) {
		t.Fatalf("create with books: %d %v (saves=%d)", code, reply, saves)
	}
	if saves != 1 {
		t.Fatalf("creating a collection of books took %d writes, want 1", saves)
	}
	if got := collectionsOf(t, server, "a1"); len(got) != 1 || got[0] != "Favourites" {
		t.Fatalf("a1 is in %v", got)
	}
}
