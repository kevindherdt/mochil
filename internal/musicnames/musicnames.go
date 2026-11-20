// Package musicnames proposes tidy file names for the tracks of one album from their tags. It only proposes: it
// never touches a file. Every proposal is checked so that a later step could apply it without breaking anything:
// it stays inside the library, keeps the file type, collides with nothing, and never relies on a missing tag.
package musicnames

import (
	"errors"
	"path"
	"strconv"
	"strings"
	"unicode"
)

// Track is one file of the album, as the library knows it.
type Track struct {
	ID          string
	Path        string // relative to the library folder, with forward slashes
	Title       string
	Artist      string
	AlbumArtist string
	Album       string
	Track       int
	Disc        int
}

// Proposal is the answer for one track. Status is "same" (already named like this), "change" or "problem".
type Proposal struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to,omitempty"`
	Status  string `json:"status"`
	Problem string `json:"problem,omitempty"`
}

const (
	maximumComponent = 120 // characters in one folder or file name
	maximumPath      = 240 // characters in the whole relative path; most file systems and sync tools allow ~255 per name and 4096 overall
)

// Clean turns a tag into one safe path component: no separators or characters Windows and sync tools reject, no
// control characters, no leading or trailing dots and spaces, and a bounded length. It returns "" if nothing is left.
func Clean(value string) string {
	value = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsControl(r), strings.ContainsRune(`/\:*?"<>|`, r):
			return ' '
		}
		return r
	}, value)
	value = strings.Trim(strings.Join(strings.Fields(value), " "), ". ")
	if runes := []rune(value); len(runes) > maximumComponent {
		value = strings.TrimRight(string(runes[:maximumComponent]), ". ")
	}
	return value
}

func albumArtist(t Track) string {
	if a := Clean(t.AlbumArtist); a != "" {
		return a
	}
	return Clean(t.Artist)
}

// ProposeFileNames is Propose for renaming files where they are: every track keeps its own folder, so cover art,
// lyrics and anything else next to it stay with it, and only the file name changes.
func ProposeFileNames(tracks []Track, exists func(relative string) bool) []Proposal {
	return propose(tracks, exists, true)
}

func propose(tracks []Track, exists func(relative string) bool, keepFolder bool) []Proposal {
	multiDisc := false
	for _, t := range tracks {
		if t.Disc > 1 {
			multiDisc = true
		}
	}
	width := max(2, len(strconv.Itoa(len(tracks))))
	proposals := make([]Proposal, len(tracks))
	targets := map[string][]int{}
	own := map[string]bool{}
	for _, t := range tracks {
		own[strings.ToLower(t.Path)] = true
	}
	for i, t := range tracks {
		p := Proposal{ID: t.ID, From: t.Path}
		extension := strings.ToLower(path.Ext(t.Path))
		artist, album, title := albumArtist(t), Clean(t.Album), Clean(t.Title)
		switch {
		case extension == "":
			p.Problem = "the file has no extension"
		case artist == "":
			p.Problem = "no artist tag"
		case album == "":
			p.Problem = "no album tag"
		case title == "":
			p.Problem = "no title tag"
		case t.Track <= 0:
			p.Problem = "no track number"
		}
		if p.Problem == "" {
			number := leftPad(t.Track, width)
			if multiDisc {
				number = strconv.Itoa(max(t.Disc, 1)) + "-" + number
			}
			p.To = artist + "/" + album + "/" + number + " - " + title + extension
			if keepFolder {
				p.To = path.Join(path.Dir(t.Path), number+" - "+title+extension)
			}
			if len([]rune(p.To)) > maximumPath {
				p.To, p.Problem = "", "the new path would be too long"
			}
		}
		proposals[i] = p
		if p.To != "" {
			key := strings.ToLower(p.To) // case-insensitive file systems treat these as one name
			targets[key] = append(targets[key], i)
		}
	}
	for key, who := range targets {
		for _, i := range who {
			switch {
			case len(who) > 1:
				proposals[i].Problem = "another track of this album would get the same name (check track numbers)"
			case exists != nil && !own[key] && exists(proposals[i].To):
				proposals[i].Problem = "a different file with that name already exists"
			}
			if proposals[i].Problem != "" {
				proposals[i].To = ""
			}
		}
	}
	for i := range proposals {
		switch {
		case proposals[i].Problem != "":
			proposals[i].Status = "problem"
		case proposals[i].To == proposals[i].From:
			proposals[i].Status = "same"
			proposals[i].To = ""
		default:
			proposals[i].Status = "change"
		}
	}
	return proposals
}

// ValidateRename says whether renaming one file to a new name is safe to do: same folder, a single clean name, the same
// file type, a bounded length. It does not look at the disk; the caller still checks that the target is free.
func ValidateRename(from, to string) error {
	switch {
	case from == "" || to == "" || from == to:
		return errors.New("nothing to rename")
	case path.Dir(from) != path.Dir(to):
		return errors.New("a file may only be renamed inside its own folder")
	case !strings.EqualFold(path.Ext(from), path.Ext(to)) || path.Ext(to) == "":
		return errors.New("the file type must stay the same")
	}
	base := path.Base(to)
	stem := strings.TrimSuffix(base, path.Ext(base))
	if stem == "" || Clean(stem) != stem || len([]rune(to)) > maximumPath {
		return errors.New("that is not a safe file name")
	}
	return nil
}

func leftPad(n, width int) string {
	s := strconv.Itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}
