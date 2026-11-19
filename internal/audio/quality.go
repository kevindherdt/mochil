package audio

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const qualitySampleLimit = 50

var copySuffixPattern = regexp.MustCompile(`(?i)(?:\s*\(\d+\)|\s*-\s*copy)$`)

type QualityIssue struct {
	Code   string   `json:"code"`
	Title  string   `json:"title"`
	Detail string   `json:"detail"`
	Paths  []string `json:"paths,omitempty"`
}

type QualityReport struct {
	Tracks int            `json:"tracks"`
	Albums int            `json:"albums"`
	Counts map[string]int `json:"counts"`
	Issues []QualityIssue `json:"issues"`
}

// InspectQuality reports deterministic tag problems without reading media
// payloads. The catalogue's probe cache already records attached artwork, so a
// bounded sidecar lookup is needed only when no track in the album is known to
// contain an embedded image.
func InspectQuality(items []Item, hasFolderArtwork func(Item) bool) QualityReport {
	report := QualityReport{Tracks: len(items), Counts: map[string]int{}}
	albums := map[string][]Item{}
	for _, item := range items {
		if unknownAudioValue(item.Artist) {
			report.add("unknown_artist", QualityIssue{Code: "unknown_artist", Title: item.Title, Detail: "Artist is missing or marked Unknown.", Paths: []string{item.Path}})
		}
		key := qualityAlbumFolder(item.Path)
		if albumIdentity(item) != "" && key != "" {
			albums[key] = append(albums[key], item)
		}
	}
	report.Albums = len(albums)
	artworkFolders := map[string]struct{}{}
	for _, tracks := range albums {
		inspectDuplicateTracks(&report, tracks)
		folder := qualityAlbumFolder(tracks[0].Path)
		if _, checked := artworkFolders[folder]; hasFolderArtwork != nil && !checked {
			artworkFolders[folder] = struct{}{}
			if !albumHasEmbeddedArtwork(tracks) && !hasFolderArtwork(tracks[0]) {
				report.add("missing_artwork", QualityIssue{Code: "missing_artwork", Title: albumDisplayName(tracks[0]), Detail: "No known embedded artwork or cover, folder, or poster sidecar was found.", Paths: []string{folder}})
			}
		}
		if fields := inconsistentAlbumFields(tracks); len(fields) > 0 {
			report.add("inconsistent_album_tags", QualityIssue{Code: "inconsistent_album_tags", Title: albumDisplayName(tracks[0]), Detail: "Tracks in one folder disagree on: " + strings.Join(fields, ", ") + ".", Paths: []string{folder}})
		}
	}
	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Code != report.Issues[j].Code {
			return report.Issues[i].Code < report.Issues[j].Code
		}
		return strings.ToLower(report.Issues[i].Title) < strings.ToLower(report.Issues[j].Title)
	})
	return report
}

func qualityAlbumFolder(itemPath string) string {
	folder := filepath.ToSlash(filepath.Dir(itemPath))
	if IsDiscFolder(filepath.Base(folder)) {
		folder = filepath.ToSlash(filepath.Dir(folder))
	}
	if folder == "." {
		return ""
	}
	return folder
}

func albumHasEmbeddedArtwork(items []Item) bool {
	for _, item := range items {
		if item.HasArtwork {
			return true
		}
	}
	return false
}

func (r *QualityReport) add(code string, issue QualityIssue) {
	previous := r.Counts[code]
	r.Counts[code] = previous + 1
	if previous < qualitySampleLimit {
		r.Issues = append(r.Issues, issue)
	}
}

func inspectDuplicateTracks(report *QualityReport, tracks []Item) {
	identities := map[string][]Item{}
	for _, item := range tracks {
		identity := normalizedAudioValue(item.Title)
		if item.Track > 0 {
			identity = strconv.Itoa(item.Disc) + ":" + strconv.Itoa(item.Track) + ":" + identity
		}
		identities[identity] = append(identities[identity], item)
	}
	for _, duplicates := range identities {
		if len(duplicates) < 2 {
			continue
		}
		pathCount := min(len(duplicates), 8)
		paths := make([]string, pathCount)
		for index, item := range duplicates[:pathCount] {
			paths[index] = item.Path
		}
		code := "track_identity_conflict"
		detail := "Multiple files share these album, disc, track, and title tags. Review their tags or editions; they are not assumed to be identical."
		if likelyFileCopies(duplicates) {
			code = "duplicate_file_candidate"
			detail = "Likely file copy: the files share a folder, normalized filename, and byte size. Compare them before removing either file."
		}
		report.add(code, QualityIssue{Code: code, Title: duplicates[0].Title, Detail: detail, Paths: paths})
	}
}

func likelyFileCopies(items []Item) bool {
	if len(items) < 2 || items[0].Size <= 0 {
		return false
	}
	folder := filepath.ToSlash(filepath.Dir(items[0].Path))
	stem := normalizedCopyStem(items[0].Path)
	extension := strings.ToLower(filepath.Ext(items[0].Path))
	size := items[0].Size
	for _, item := range items[1:] {
		if item.Size != size || strings.ToLower(filepath.Ext(item.Path)) != extension || filepath.ToSlash(filepath.Dir(item.Path)) != folder || normalizedCopyStem(item.Path) != stem {
			return false
		}
	}
	return true
}

func normalizedCopyStem(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return normalizedAudioValue(copySuffixPattern.ReplaceAllString(base, ""))
}

func inconsistentAlbumFields(items []Item) []string {
	fields := []struct {
		name string
		get  func(Item) string
	}{{"album", func(item Item) string { return item.Album }}, {"album artist", func(item Item) string { return item.AlbumArtist }}, {"genre", func(item Item) string { return item.Genre }}}
	out := []string{}
	for _, field := range fields {
		values := map[string]struct{}{}
		for _, item := range items {
			if value := normalizedAudioValue(field.get(item)); value != "" {
				values[value] = struct{}{}
			}
		}
		if len(values) > 1 {
			out = append(out, field.name)
		}
	}
	return out
}

func albumIdentity(item Item) string {
	album := normalizedAudioValue(item.Album)
	if album == "" {
		return ""
	}
	artist := normalizedAudioValue(item.AlbumArtist)
	if artist == "" {
		artist = normalizedAudioValue(item.Artist)
	}
	return artist + "\x00" + album
}

func albumDisplayName(item Item) string {
	if strings.TrimSpace(item.Album) != "" {
		return item.Album
	}
	return filepath.Base(filepath.Dir(item.Path))
}

func unknownAudioValue(value string) bool {
	value = normalizedAudioValue(value)
	return value == "" || value == "unknown" || value == "unknown artist"
}

func normalizedAudioValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}
