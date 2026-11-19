package audio

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestArtworkStoreCreatesDistinctBoundedVariants(t *testing.T) {
	root := t.TempDir()
	argumentsPath := filepath.Join(root, "arguments")
	commandPath := filepath.Join(root, "extract")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + strconv.Quote(argumentsPath) + "\n" +
		"for last do :; done\n" +
		"printf image > \"$last\"\n"
	if err := os.WriteFile(commandPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := NewArtworkStore(filepath.Join(root, "cache"), commandPath, 1<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{ID: "track", LibraryID: "music", Revision: "revision"}
	inputPath := filepath.Join(root, "source")
	if err = os.WriteFile(inputPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, 2)
	for _, variant := range []ArtworkVariant{ArtworkCard, ArtworkPlayer} {
		input, openErr := os.Open(inputPath)
		if openErr != nil {
			t.Fatal(openErr)
		}
		generated, coverErr := store.CoverVariant(input, item, variant, item.Revision)
		if coverErr != nil {
			t.Fatal(coverErr)
		}
		paths = append(paths, generated)
	}
	if paths[0] == paths[1] {
		t.Fatal("card and player variants shared a cache path")
	}
	arguments, err := os.ReadFile(argumentsPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(arguments)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "scale=320:480:force_original_aspect_ratio=decrease") || !strings.Contains(lines[1], "scale=720:1080:force_original_aspect_ratio=decrease") {
		t.Fatalf("unexpected extraction arguments: %q", lines)
	}
}

func TestArtworkVariantKeyIncludesSourceRevision(t *testing.T) {
	root := t.TempDir()
	commandPath := filepath.Join(root, "extract")
	script := "#!/bin/sh\nfor last do :; done\nprintf image > \"$last\"\n"
	if err := os.WriteFile(commandPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := NewArtworkStore(filepath.Join(root, "cache"), commandPath, 1<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{ID: "track", LibraryID: "music"}
	var paths []string
	for _, revision := range []string{"sidecar:1", "sidecar:2"} {
		inputPath := filepath.Join(root, revision)
		if err = os.WriteFile(inputPath, []byte("source"), 0o600); err != nil {
			t.Fatal(err)
		}
		input, openErr := os.Open(inputPath)
		if openErr != nil {
			t.Fatal(openErr)
		}
		generated, coverErr := store.CoverVariant(input, item, ArtworkCard, revision)
		if coverErr != nil {
			t.Fatal(coverErr)
		}
		paths = append(paths, generated)
	}
	if paths[0] == paths[1] {
		t.Fatal("changed sidecar revision reused a stale cache path")
	}
}

func TestArtworkSiblingsPrioritizeKnownEmbeddedCovers(t *testing.T) {
	store := &Store{}
	store.publish([]Item{
		{ID: "plain-1", Path: "Album/01.flac"},
		{ID: "cover", Path: "Album/12.flac", HasArtwork: true},
		{ID: "target", Path: "Album/08.flac"},
		{ID: "plain-2", Path: "Album/02.flac"},
	})
	siblings := store.ArtworkSiblings("target", 2)
	if len(siblings) != 2 || siblings[0].ID != "cover" {
		t.Fatalf("siblings=%+v", siblings)
	}
}
