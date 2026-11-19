// Package audio owns local audiobook, podcast, and music indexing. Playback
// state remains family-specific even though the scanner and stream transport
// are shared.
package audio

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Item struct {
	ID               string    `json:"id"`
	LibraryID        string    `json:"library_id"`
	Path             string    `json:"path"`
	Title            string    `json:"title"`
	Artist           string    `json:"artist,omitempty"`
	AlbumArtist      string    `json:"album_artist,omitempty"`
	Album            string    `json:"album,omitempty"`
	Series           string    `json:"series,omitempty"`
	SeriesIndex      float64   `json:"series_index,omitempty"`
	Genre            string    `json:"genre,omitempty"`
	Narrator         string    `json:"narrator,omitempty"`
	Container        string    `json:"container"`
	MediaType        string    `json:"media_type"`
	Revision         string    `json:"revision"`
	Track            int       `json:"track,omitempty"`
	Disc             int       `json:"disc,omitempty"`
	Year             int       `json:"year,omitempty"`
	Bitrate          int64     `json:"bitrate,omitempty"`
	Codec            string    `json:"codec,omitempty"`
	SampleRate       int       `json:"sample_rate,omitempty"`
	BitDepth         int       `json:"bit_depth,omitempty"`
	HasArtwork       bool      `json:"has_artwork,omitempty"`
	Duration         float64   `json:"duration_seconds,omitempty"`
	ReplayTrack      float64   `json:"replaygain_track_db,omitempty"`
	ReplayAlbum      float64   `json:"replaygain_album_db,omitempty"`
	PeakTrack        float64   `json:"replaygain_track_peak,omitempty"`
	PeakAlbum        float64   `json:"replaygain_album_peak,omitempty"`
	AnalysisRevision string    `json:"analysis_revision,omitempty"`
	Size             int64     `json:"size"`
	Modified         time.Time `json:"modified"`
	Chapters         []Chapter `json:"chapters,omitempty"`
	ProbeVersion     int       `json:"probe_version,omitempty"`
}

type Chapter struct {
	Title string  `json:"title"`
	Start float64 `json:"start_seconds"`
	End   float64 `json:"end_seconds"`
}

func cloneItem(item Item) Item {
	item.Chapters = append([]Chapter(nil), item.Chapters...)
	return item
}

func cloneItems(items []Item) []Item {
	out := make([]Item, len(items))
	for index := range items {
		out[index] = cloneItem(items[index])
	}
	return out
}

var formats = map[string]string{
	".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".m4b": "audio/mp4", ".aac": "audio/aac",
	".ogg": "audio/ogg", ".oga": "audio/ogg", ".opus": "audio/ogg", ".flac": "audio/flac", ".wav": "audio/wav", ".wave": "audio/wav",
	".weba": "audio/webm", ".webm": "audio/webm",
}

var audioTitleSeparators = strings.NewReplacer("_", " ", ".", " ")

func Identify(libraryID, relative string, size int64, modified time.Time) (Item, bool) {
	if size <= 0 {
		return Item{}, false
	}
	relative = filepath.ToSlash(filepath.Clean(relative))
	if relative == "." || relative == ".." || strings.HasPrefix(relative, "../") || filepath.IsAbs(relative) {
		return Item{}, false
	}
	ext := strings.ToLower(filepath.Ext(relative))
	mediaType, ok := formats[ext]
	if !ok || strings.HasPrefix(filepath.Base(relative), "._") {
		return Item{}, false
	}
	id := sha256.Sum256([]byte(libraryID + "\x00" + relative))
	revision := sha256.Sum256([]byte(relative + "\x00" + strconv.FormatInt(size, 10) + "\x00" + modified.UTC().Format(time.RFC3339Nano)))
	stem := strings.TrimSuffix(filepath.Base(relative), filepath.Ext(relative))
	title := strings.Join(strings.Fields(audioTitleSeparators.Replace(stem)), " ")
	if title == "" {
		title = "Untitled audio"
	}
	return Item{ID: hex.EncodeToString(id[:16]), LibraryID: libraryID, Path: relative, Title: title, Container: strings.TrimPrefix(ext, "."), MediaType: mediaType, Revision: hex.EncodeToString(revision[:12]), Size: size, Modified: modified}, true
}

func parseNumber(value string) int {
	value = strings.TrimSpace(strings.SplitN(value, "/", 2)[0])
	number, _ := strconv.Atoi(value)
	return number
}
