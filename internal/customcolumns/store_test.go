package customcolumns

import (
	"path/filepath"
	"testing"
)

func open(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "columns.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return store, path
}

func TestColumnsGetStableDistinctIdsAndRejectBadOnes(t *testing.T) {
	store, _ := open(t)
	genre, err := store.Add("  Genre / Mood ", Text)
	if err != nil || genre.ID != "#genre_mood" {
		t.Fatalf("%+v %v", genre, err)
	}
	if _, err := store.Add("genre / mood", Text); err == nil {
		t.Fatal("the same label twice was accepted")
	}
	if clash, err := store.Add("Genre-Mood", Number); err != nil || clash.ID != "#genre_mood_2" {
		t.Fatalf("a different label with the same slug needs its own id: %+v %v", clash, err)
	}
	for _, bad := range [][2]string{{"", Text}, {"!!!", Text}, {"Fine", "colour"}} {
		if _, err := store.Add(bad[0], bad[1]); err == nil {
			t.Errorf("%q/%q was accepted", bad[0], bad[1])
		}
	}
}

func TestValuesAreCheckedByTypeStoredAndSurviveReopening(t *testing.T) {
	store, path := open(t)
	score, _ := store.Add("Score", Number)
	done, _ := store.Add("Reviewed", YesNo)
	mood, _ := store.Add("Mood", Tags)
	if _, err := store.Set("lib", map[string]map[string]string{"b1": {score.ID: "abc"}}); err == nil {
		t.Fatal("text in a number column was accepted")
	}
	if _, err := store.Set("lib", map[string]map[string]string{"b1": {"#missing": "x"}}); err == nil {
		t.Fatal("an unknown column was accepted")
	}
	if _, err := store.Set("lib", map[string]map[string]string{"b1": {score.ID: " 7.50 ", done.ID: "Y", mood.ID: "cosy , dark,, "}}); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.Library("lib")["b1"]
	if got[score.ID] != "7.5" || got[done.ID] != "yes" || got[mood.ID] != "cosy, dark" {
		t.Fatalf("%v", got)
	}
}

func TestUndoPutsEveryTouchedCellBackAndEmptyClears(t *testing.T) {
	store, _ := open(t)
	note, _ := store.Add("Note", Text)
	if _, err := store.Set("lib", map[string]map[string]string{"b1": {note.ID: "old"}}); err != nil {
		t.Fatal(err)
	}
	undo, err := store.Set("lib", map[string]map[string]string{"b1": {note.ID: "new"}, "b2": {note.ID: "fresh"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	library := store.Library("lib")
	if library["b1"][note.ID] != "old" {
		t.Fatalf("b1 = %v", library["b1"])
	}
	if _, there := library["b2"]; there {
		t.Fatalf("b2 should be gone again: %v", library)
	}
	if _, err := store.Set("lib", map[string]map[string]string{"b1": {note.ID: ""}}); err != nil {
		t.Fatal(err)
	}
	if len(store.Library("lib")) != 0 {
		t.Fatal("an empty value should remove the cell")
	}
}

func TestRemovingAColumnDropsItsValuesOnly(t *testing.T) {
	store, path := open(t)
	a, _ := store.Add("A", Text)
	b, _ := store.Add("B", Text)
	if _, err := store.Set("lib", map[string]map[string]string{"b1": {a.ID: "1", b.ID: "2"}}); err != nil {
		t.Fatal(err)
	}
	if err := store.Remove(a.ID); err != nil {
		t.Fatal(err)
	}
	reopened, _ := Open(path)
	if got := reopened.Library("lib")["b1"]; len(got) != 1 || got[b.ID] != "2" || len(reopened.Columns()) != 1 {
		t.Fatalf("%v %v", got, reopened.Columns())
	}
	if err := store.Remove(a.ID); err == nil {
		t.Fatal("removing twice should fail")
	}
}
