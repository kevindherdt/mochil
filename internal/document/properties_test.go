package document

import (
	"reflect"
	"testing"
)

func TestPagePropertiesReadsSimpleFrontMatterAndSkipsTags(t *testing.T) {
	page := "---\nstatus: Draft\ntags: [a, b]\naliases:\n  - One\n  - \"Two\"\nrating: [3, 4]\nnotes: |\n  long\n---\nbody #tag\n"
	got := pageProperties(page)
	want := [][2]string{{"status", "Draft"}, {"aliases", "One"}, {"aliases", "Two"}, {"rating", "3"}, {"rating", "4"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if len(pageProperties("no front matter\nstatus: x")) != 0 {
		t.Fatal("only a leading block counts as front matter")
	}
}
