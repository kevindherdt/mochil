package audio

import (
	"path/filepath"
	"strconv"
	"strings"
)

func inferFolderMetadata(item *Item) {
	inferBookFolderMetadata(item)
	inferMusicFolderMetadata(item)
}

// inferMusicFolderMetadata handles explicitly labelled music layouts without
// treating every hyphenated filename as structured metadata. Embedded tags
// always win.
func inferMusicFolderMetadata(item *Item) {
	parts := strings.Split(filepath.ToSlash(item.Path), "/")
	if len(parts) < 2 {
		return
	}
	label := -1
	for index := len(parts) - 2; index >= 0; index-- {
		name := strings.ToLower(strings.TrimSpace(parts[index]))
		if name == "albums" || name == "tracks" {
			label = index
			break
		}
	}
	if label < 0 {
		return
	}
	switch strings.ToLower(strings.TrimSpace(parts[label])) {
	case "albums":
		if label+2 >= len(parts) {
			return
		}
		artist, album, ok := splitMusicName(parts[label+1])
		if !ok {
			return
		}
		if item.Artist == "" {
			item.Artist = artist
		}
		if item.AlbumArtist == "" {
			item.AlbumArtist = artist
		}
		if item.Album == "" {
			item.Album = album
		}
		inferMusicFilename(item, artist, album, parts[len(parts)-1])
	case "tracks":
		name := strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))
		artist, _, ok := splitMusicName(name)
		if ok && item.Artist == "" {
			item.Artist = artist
		}
	}
}

func inferMusicFilename(item *Item, artist, album, filename string) {
	stem := strings.TrimSuffix(filename, filepath.Ext(filename))
	if !strings.EqualFold(strings.TrimSpace(item.Title), strings.TrimSpace(stem)) {
		return
	}
	prefix := artist + " - " + album + " - "
	if len(stem) < len(prefix) || !strings.EqualFold(stem[:len(prefix)], prefix) {
		return
	}
	remainder := strings.TrimSpace(stem[len(prefix):])
	ordinal, title, found := strings.Cut(remainder, " ")
	if !found || strings.TrimSpace(title) == "" {
		return
	}
	left, right, pair := strings.Cut(ordinal, "-")
	trackText := left
	if pair {
		trackText = right
		if item.Disc == 0 {
			if disc, err := strconv.Atoi(strings.TrimLeft(left, "0")); err == nil && disc > 0 {
				item.Disc = disc
			}
		}
	}
	if item.Track == 0 {
		if track, err := strconv.Atoi(strings.TrimLeft(trackText, "0")); err == nil && track > 0 {
			item.Track = track
		}
	}
	if item.Track > 0 {
		item.Title = strings.TrimSpace(title)
	}
}

func splitMusicName(value string) (string, string, bool) {
	left, right, found := strings.Cut(strings.TrimSpace(value), " - ")
	left, right = strings.TrimSpace(left), strings.TrimSpace(right)
	return left, right, found && left != "" && right != ""
}

// inferBookFolderMetadata handles the common audiobook layout
// Author/Series/Book NN/title.m4b. The explicit Book/Volume marker keeps this
// from guessing at ordinary Artist/Album music folders.
func inferBookFolderMetadata(item *Item) {
	parts := strings.Split(filepath.ToSlash(item.Path), "/")
	if len(parts) < 4 {
		return
	}
	bookFolder := parts[len(parts)-2]
	volume, ok := bookFolderNumber(bookFolder)
	if !ok {
		return
	}
	series := strings.TrimSpace(parts[len(parts)-3])
	author := strings.TrimSpace(parts[len(parts)-4])
	if item.Series == "" && series != "" {
		item.Series = series
	}
	if item.Artist == "" && author != "" {
		item.Artist = author
	}
	if item.Track == 0 {
		item.Track = volume
	}
	if item.SeriesIndex == 0 {
		item.SeriesIndex = float64(volume)
	}
}

func bookFolderNumber(value string) (int, bool) {
	lower := strings.ToLower(strings.TrimSpace(value))
	matched := false
	for _, prefix := range []string{"volume", "book", "vol"} {
		if strings.HasPrefix(lower, prefix) {
			lower = strings.TrimLeft(strings.TrimPrefix(lower, prefix), " ._-")
			matched = true
			break
		}
	}
	if !matched {
		return 0, false
	}
	end := 0
	for end < len(lower) && ((lower[end] >= '0' && lower[end] <= '9') || lower[end] == '.') {
		end++
	}
	if end == 0 {
		return 0, false
	}
	number, err := strconv.ParseFloat(strings.TrimRight(lower[:end], "."), 64)
	if err != nil || number <= 0 {
		return 0, false
	}
	return int(number), true
}
