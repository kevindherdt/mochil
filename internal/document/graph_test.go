package document

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGraphResolvesLinksByPathThenUniqueName(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string) {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Home.md", "[[Plan]] [[Projects/Plan|alias]] [[Notes/Dup]] [[Missing]] [[Home]]")
	write("Projects/Plan.md", "back to [[home]]")
	write("Notes/Dup.md", "x")
	write("Archive/Dup.md", "y")
	store := NewVaultStore("v", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	graph, err := store.Graph()
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(graph.Nodes))
	for index, node := range graph.Nodes {
		names[index] = node.Path
	}
	edge := func(from, to string) bool {
		for _, pair := range graph.Edges {
			if names[pair[0]] == from && names[pair[1]] == to {
				return true
			}
		}
		return false
	}
	if !edge("Home.md", "Projects/Plan.md") || !edge("Projects/Plan.md", "Home.md") || !edge("Home.md", "Notes/Dup.md") {
		t.Fatalf("missing expected edges: %v over %v", graph.Edges, names)
	}
	if len(graph.Edges) != 3 {
		t.Fatalf("edges = %v over %v (no self links, no missing pages, no duplicates)", graph.Edges, names)
	}
}

// A note saved as .markdown is a Markdown page everywhere else in the store, so
// the graph must carry it as a node and its [[links]] as edges.
func TestGraphIncludesMarkdownExtensionNotes(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{
		"Home.md":           "Welcome.",
		"Longform.markdown": "See [[Home]].",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := NewVaultStore("v", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	graph, err := store.Graph()
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(graph.Nodes))
	for index, node := range graph.Nodes {
		names[index] = node.Path
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("nodes = %v, want the .markdown note as a page too", names)
	}
	linked := false
	for _, pair := range graph.Edges {
		if names[pair[0]] == "Longform.markdown" && names[pair[1]] == "Home.md" {
			linked = true
		}
	}
	if !linked {
		t.Fatalf("edges = %v over %v, want Longform.markdown -> Home.md", graph.Edges, names)
	}
}
