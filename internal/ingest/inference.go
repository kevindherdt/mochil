// Package ingest contains conservative, storage-independent media identity
// inference. It proposes metadata and records the evidence behind every value;
// callers remain responsible for deciding whether to accept a proposal.
package ingest

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Kind string

const (
	Publication Kind = "publication"
	Music       Kind = "music"
	Video       Kind = "video"
)

type Confidence string

const (
	High   Confidence = "high"
	Medium Confidence = "medium"
	Low    Confidence = "low"
)

type Evidence struct {
	Field      string     `json:"field"`
	Value      string     `json:"value"`
	Source     string     `json:"source"`
	Confidence Confidence `json:"confidence"`
}

type Proposal struct {
	Kind        Kind              `json:"kind"`
	Title       string            `json:"title,omitempty"`
	Authors     []string          `json:"authors,omitempty"`
	Series      string            `json:"series,omitempty"`
	SeriesIndex *float64          `json:"series_index,omitempty"`
	Year        int               `json:"year,omitempty"`
	Season      int               `json:"season,omitempty"`
	Episode     int               `json:"episode,omitempty"`
	Album       string            `json:"album,omitempty"`
	AlbumArtist string            `json:"album_artist,omitempty"`
	Disc        int               `json:"disc,omitempty"`
	Track       int               `json:"track,omitempty"`
	Identifiers map[string]string `json:"identifiers,omitempty"`
	Evidence    []Evidence        `json:"evidence"`
	// Conflicts lists fields where the book and its path disagree, for a person to settle.
	Conflicts []string `json:"conflicts,omitempty"`
}

var (
	yearSuffix    = regexp.MustCompile(`\s*\((19\d{2}|20\d{2})\)\s*$`)
	identifier    = regexp.MustCompile(`(?i)\[(isbn|tmdbid|tvdbid|imdbid)[=:]([^\]]+)\]`)
	volume        = regexp.MustCompile(`(?i)(?:^|[\s._-])(?:v(?:ol(?:ume)?)?\.?|book|#)\s*(\d+(?:\.\d+)?)(?:[\s._-]|$)`)
	episode       = regexp.MustCompile(`(?i)\bS0*([0-9]{1,3})E0*([0-9]{1,4})\b`)
	seasonDir     = regexp.MustCompile(`(?i)^season[ ._-]*0*([0-9]{1,3})$`)
	trackLead     = regexp.MustCompile(`^(?:(\d{1,2})[-.]?)?(\d{1,3})\s*[-_.]\s*(.+)$`)
	partMarker    = regexp.MustCompile(`(?i)(?:^|[\s._-])(?:part|pt)\.?\s*(\d+)(?:[\s._-]|$)`)
	bracketSuffix = regexp.MustCompile(`\s+\[[^\]]*\].*$`)
	separatorRun  = regexp.MustCompile(`(?:\s*[-–—:_]\s*){2,}`)
)

// InferPath extracts only explicit or structurally strong metadata. Bare
// numbers are not treated as volumes, seasons, or years.
func InferPath(kind Kind, path string) Proposal {
	clean := filepath.ToSlash(filepath.Clean(path))
	parts := strings.Split(clean, "/")
	base := strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1]))
	p := Proposal{Kind: kind, Identifiers: map[string]string{}}
	base = extractIdentifiers(base, &p)
	switch kind {
	case Publication:
		inferPublication(parts, base, &p)
	case Video:
		inferVideo(parts, base, &p)
	case Music:
		inferMusic(parts, base, &p)
	}
	if len(p.Identifiers) == 0 {
		p.Identifiers = nil
	}
	return p
}

func inferPublication(parts []string, base string, p *Proposal) {
	// "!" folders are categories, not authors or series. A side-stories folder keeps its books in the parent series.
	sideStory := false
	var kept []string
	for i, part := range parts {
		if i < len(parts)-1 && strings.HasPrefix(part, "!") {
			if name := strings.ToLower(strings.TrimSpace(part[1:])); name == "side stories" || name == "side story" {
				sideStory = true
			}
			continue
		}
		kept = append(kept, part)
	}
	if len(kept) < len(parts) && len(kept) == 2 {
		p.Series = cleanYear(display(kept[0]), p)
		p.add("series", p.Series, "series folder", Medium)
	}
	parts = kept
	if len(parts) >= 3 {
		p.Authors = []string{display(parts[len(parts)-3])}
		p.Series = cleanYear(display(parts[len(parts)-2]), p)
		p.add("authors", p.Authors[0], "author folder", Low)
		p.add("series", p.Series, "series folder", Medium)
	}
	if p.Series == "" && len(parts) == 2 {
		// Series/Book.epub: the folder is the series when the book's name starts with it
		folder := display(parts[0])
		if key := normalize(folder); len(key) >= 3 && strings.HasPrefix(normalize(base), key) {
			p.Series = cleanYear(folder, p)
			p.add("series", p.Series, "folder named at the start of the filename", High)
		}
	}
	if match := volume.FindStringSubmatch(base); len(match) == 2 {
		if value, err := strconv.ParseFloat(match[1], 64); err == nil && value > 0 {
			p.SeriesIndex = &value
			p.add("series_index", match[1], "explicit volume marker", High)
			// "Volume 1 Part 2" is volume 1.2, as Calibre-style libraries number it.
			if part := partMarker.FindStringSubmatch(base); len(part) == 2 && !strings.Contains(match[1], ".") {
				if sub, err := strconv.Atoi(part[1]); err == nil && sub > 0 && sub < 10 {
					combined, _ := strconv.ParseFloat(match[1]+"."+part[1], 64)
					p.SeriesIndex = &combined
					p.add("series_index", strconv.FormatFloat(combined, 'f', -1, 64), "volume and part markers", High)
				}
			}
		}
	}
	p.Title = cleanPublicationTitle(base)
	if p.Series != "" {
		prefix := p.Series + " - "
		if strings.HasPrefix(strings.ToLower(p.Title), strings.ToLower(prefix)) {
			p.Title = strings.TrimSpace(p.Title[len(prefix):])
		}
	}
	p.add("title", p.Title, "filename", Medium)
	if sideStory && !strings.Contains(strings.ToLower(p.Title), "side stor") {
		p.Title += " (Side Story)"
		p.add("title", p.Title, "side stories folder", Medium)
	}
}

func inferVideo(parts []string, base string, p *Proposal) {
	if match := episode.FindStringSubmatch(base); len(match) == 3 {
		p.Season, _ = strconv.Atoi(match[1])
		p.Episode, _ = strconv.Atoi(match[2])
		p.add("season", strconv.Itoa(p.Season), "SxxEyy filename marker", High)
		p.add("episode", strconv.Itoa(p.Episode), "SxxEyy filename marker", High)
		if len(parts) >= 3 {
			show := extractIdentifiers(parts[len(parts)-3], p)
			p.Series = cleanYear(display(show), p)
			p.add("series", p.Series, "show folder", High)
			if match := seasonDir.FindStringSubmatch(parts[len(parts)-2]); len(match) == 2 {
				season, _ := strconv.Atoi(match[1])
				p.Season = season
			}
		}
		p.Title = displayFilename(strings.TrimLeft(episode.ReplaceAllString(base, ""), " -–—:._"))
		if p.Series != "" && len(p.Title) > len(p.Series) && strings.EqualFold(p.Title[:len(p.Series)], p.Series) {
			p.Title = strings.TrimLeft(p.Title[len(p.Series):], " -–—:._")
		}
		if p.Title != "" {
			p.add("title", p.Title, "episode filename", Medium)
		}
		return
	}
	p.Title = cleanYear(displayFilename(base), p)
	p.add("title", p.Title, "movie filename", Medium)
}

func inferMusic(parts []string, base string, p *Proposal) {
	if len(parts) >= 3 {
		p.AlbumArtist = display(parts[len(parts)-3])
		p.Album = cleanYear(display(parts[len(parts)-2]), p)
		p.add("album_artist", p.AlbumArtist, "artist folder", Medium)
		p.add("album", p.Album, "album folder", Medium)
	}
	if match := trackLead.FindStringSubmatch(base); len(match) == 4 {
		p.Disc, _ = strconv.Atoi(match[1])
		p.Track, _ = strconv.Atoi(match[2])
		p.Title = displayFilename(match[3])
		if p.Disc > 0 {
			p.add("disc", strconv.Itoa(p.Disc), "disc-track filename prefix", High)
		}
		p.add("track", strconv.Itoa(p.Track), "track filename prefix", High)
	} else {
		p.Title = displayFilename(base)
	}
	p.add("title", p.Title, "filename", Medium)
}

func extractIdentifiers(value string, p *Proposal) string {
	for _, match := range identifier.FindAllStringSubmatch(value, -1) {
		key, val := strings.ToLower(match[1]), strings.TrimSpace(match[2])
		if val != "" {
			p.Identifiers[key] = val
			p.add("identifier."+key, val, "filename identifier", High)
		}
	}
	return strings.TrimSpace(identifier.ReplaceAllString(value, ""))
}

func cleanYear(value string, p *Proposal) string {
	if match := yearSuffix.FindStringSubmatch(value); len(match) == 2 {
		p.Year, _ = strconv.Atoi(match[1])
		p.add("year", match[1], "folder or filename year", High)
		return strings.TrimSpace(yearSuffix.ReplaceAllString(value, ""))
	}
	return value
}

func cleanPublicationTitle(value string) string {
	// a release tag at the end, like " [Translator, Publisher]", is not part of the title
	value = bracketSuffix.ReplaceAllString(value, "")
	value = volume.ReplaceAllString(value, " ")
	value = separatorRun.ReplaceAllString(value, " - ")
	return display(strings.Trim(value, " -–—:._"))
}

func display(value string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(value, "_", " ")), " ")
}

func displayFilename(value string) string {
	return display(strings.ReplaceAll(value, ".", " "))
}

func (p *Proposal) add(field, value, source string, confidence Confidence) {
	if strings.TrimSpace(value) != "" {
		p.Evidence = append(p.Evidence, Evidence{Field: field, Value: value, Source: source, Confidence: confidence})
	}
}
