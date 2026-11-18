package main

import (
	"io/fs"
	"path/filepath"
	"strings"
)

var narrationAudioExtensions = map[string]bool{
	".mp3": true, ".m4a": true, ".m4b": true, ".aac": true, ".flac": true, ".ogg": true, ".opus": true, ".wav": true,
}

// holdsOnlyNarratedBooks reports whether a folder added as an audiobook library
// contains EPUBs but no loose audio files. Storyteller-style EPUBs carry their
// narration inside the book, so such a folder belongs on the book shelves, where
// read-along works. The walk is bounded so a huge tree cannot slow startup.
func holdsOnlyNarratedBooks(root string) bool {
	const visitLimit = 20000
	visited, epubs := 0, 0
	stop := fs.SkipAll
	found := false
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if visited++; visited > visitLimit {
			return stop
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if narrationAudioExtensions[extension] {
			found = true
			return stop
		}
		if extension == ".epub" {
			epubs++
		}
		return nil
	})
	return epubs > 0 && !found
}
