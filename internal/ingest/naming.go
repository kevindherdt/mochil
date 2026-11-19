package ingest

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

// Naming presets decide where an imported book lands in the managed library.
const (
	NamingDefault = "default" // Author/Series/Series - Vol 1 - Title.ext
	NamingCalibre = "calibre" // Author/Title/Title - Author.ext, one book per folder as Calibre keeps them
	NamingKavita  = "kavita"  // Series/Series - Vol 01.ext, side stories flat as Series - Side Story - Title.ext
)

func ValidNamingPreset(preset string) bool {
	switch preset {
	case "", NamingDefault, NamingCalibre, NamingKavita:
		return true
	}
	return false
}

const sideStoryMarker = " (Side Story)"

// pathPart makes one safe folder or file name: no separators, no control characters, no leading or trailing dots or spaces.
func pathPart(value string) string {
	value = strings.Map(func(character rune) rune {
		if character == '/' || character == '\\' || character == ':' || character == '*' || character == '?' || character == '"' || character == '<' || character == '>' || character == '|' || character < 32 {
			return '-'
		}
		return character
	}, strings.TrimSpace(value))
	return strings.Trim(strings.Join(strings.Fields(value), " "), " .-")
}

// paddedVolume writes a volume number with a two-digit whole part: 3 becomes 03 and 1.5 becomes 01.5.
func paddedVolume(index float64) string {
	whole := math.Floor(index)
	text := fmt.Sprintf("%02d", int(whole))
	if fraction := index - whole; fraction > 1e-9 {
		text += strings.TrimPrefix(strconv.FormatFloat(fraction, 'f', -1, 64), "0")
	}
	return text
}

// TargetFor is the managed-library path for a publication under a preset. It returns false when the proposal says too
// little to name the book (no title), so the caller can send it to review instead of inventing a name.
func TargetFor(preset, source string, proposal Proposal) (string, bool) {
	if proposal.Kind != Publication {
		return suggestedTarget(source, proposal), true
	}
	if preset == "" || preset == NamingDefault {
		return suggestedTarget(source, proposal), true
	}
	extension := strings.ToLower(filepath.Ext(source))
	title := strings.TrimSpace(proposal.Title)
	sideStory := strings.HasSuffix(title, sideStoryMarker)
	title = strings.TrimSpace(strings.TrimSuffix(title, sideStoryMarker))
	if pathPart(title) == "" {
		return "", false
	}
	author := ""
	if len(proposal.Authors) > 0 {
		author = pathPart(proposal.Authors[0])
	}
	series := pathPart(proposal.Series)
	switch preset {
	case NamingCalibre:
		if author == "" {
			author = "Unknown"
		}
		name := pathPart(title)
		if author != "Unknown" {
			name += " - " + author
		}
		return strings.Join([]string{author, pathPart(title), pathPart(name) + extension}, "/"), true
	case NamingKavita:
		if series == "" {
			return pathPart(title) + "/" + pathPart(title) + extension, true
		}
		switch {
		case sideStory:
			return series + "/" + pathPart(series+" - Side Story - "+title) + extension, true
		case proposal.SeriesIndex != nil:
			return series + "/" + pathPart(series+" - Vol "+paddedVolume(*proposal.SeriesIndex)) + extension, true
		}
		return series + "/" + pathPart(series+" - "+title) + extension, true
	}
	return suggestedTarget(source, proposal), true
}
