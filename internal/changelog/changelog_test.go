package changelog

import "testing"

func TestEntriesAreWellFormedAndNewestFirst(t *testing.T) {
	all := List("", 0)
	if len(all) == 0 {
		t.Fatal("the changelog is empty")
	}
	seen := map[string]bool{}
	for _, e := range all {
		key := e.Component + "/" + e.Version
		if e.Component == "" || e.Version == "" || e.Date == "" || seen[key] {
			t.Fatalf("bad or duplicate entry %q", key)
		}
		seen[key] = true
		if len(e.New)+len(e.Fixed)+len(e.Performance)+len(e.Changes) == 0 {
			t.Fatalf("entry %q has no notes", key)
		}
	}
	for i := 1; i < len(all); i++ {
		if all[i].Component == all[0].Component && all[i].Date > all[0].Date {
			t.Fatalf("entries are not newest first at %d", i)
		}
	}
}

func TestListFiltersAndLimits(t *testing.T) {
	if got := List("android", 3); len(got) != 3 || got[0].Component != "android" {
		t.Fatalf("got %+v", got)
	}
	if len(List("nothing", 0)) != 0 {
		t.Fatal("unknown component returned entries")
	}
	if _, ok := For("android", "0.18.96"); !ok {
		t.Fatal("current release is missing from the changelog")
	}
}
