package document

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPageTagsReadsFrontmatterAndInlineTagsButNotCode(t *testing.T) {
	content := "---\ntitle: X\ntags:\n  - Project\n  - \"#Deep/Work\"\n---\n# Heading\nText #idea and (#Idea) with a [[Page#Section]] and https://x.test/#anchor.\n`#code` and #2024 are not tags\n```\n#fenced\n```\n#last-one\n"
	got := pageTags(content)
	want := []string{"project", "deep/work", "idea", "last-one"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
}

func TestFrontmatterTagsInlineForms(t *testing.T) {
	for _, front := range []string{"tags: [a, b]\n", "tags: a, b\n", "tag: a b\n"} {
		if got := pageTags("---\n" + front + "---\nBody"); !reflect.DeepEqual(got, []string{"a", "b"}) {
			t.Fatalf("%q => %#v", front, got)
		}
	}
	if got := pageTags("No frontmatter, #solo"); !reflect.DeepEqual(got, []string{"solo"}) {
		t.Fatalf("got %#v", got)
	}
}

func TestPageTagsKeepsTheOpeningFenceUntilItsOwnCloser(t *testing.T) {
	content := "```\n~~~\n#inside-code\n```\n#real\n"
	if got := pageTags(content); !reflect.DeepEqual(got, []string{"real"}) {
		t.Fatalf("mixed fences = %#v, want [real]", got)
	}
	tilde := "~~~\n```\n#inside-code\n~~~\n#real\n"
	if got := pageTags(tilde); !reflect.DeepEqual(got, []string{"real"}) {
		t.Fatalf("tilde fence = %#v, want [real]", got)
	}
}

func TestStoreTagsIncludeEveryMarkdownExtension(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{"one.md": "#shared first", "two.markdown": "#shared second"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	groups, err := store.Tags()
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range groups {
		if group.Tag == "shared" {
			if len(group.IDs) != 2 {
				t.Fatalf("shared ids = %d, want 2 (a .markdown note is a Markdown note)", len(group.IDs))
			}
			return
		}
	}
	t.Fatal("shared tag missing")
}
