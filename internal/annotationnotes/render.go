// Package annotationnotes converts synchronized publication annotations into
// a portable note without coupling either persistent store to the other.
package annotationnotes

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"mochil/internal/annotations"
)

const MaximumContentBytes = 2 << 20

var (
	ErrEmpty    = errors.New("publication has no annotations to export")
	ErrTooLarge = errors.New("annotation note exceeds the wiki page limit")
)

type Publication struct {
	Title       string
	Authors     []string
	Series      string
	SeriesIndex *float64
}

type Note struct {
	Title   string
	Content string
}

func Render(publication Publication, items []annotations.Annotation) (Note, error) {
	visible := make([]annotations.Annotation, 0, len(items))
	for _, item := range items {
		if !item.Deleted {
			visible = append(visible, item)
		}
	}
	if len(visible) == 0 {
		return Note{}, ErrEmpty
	}

	title := boundedUTF8("Annotations — "+strings.TrimSpace(publication.Title), 300)
	if title == "Annotations —" {
		title = "Publication annotations"
	}
	var content strings.Builder
	write := func(value string) error {
		if content.Len()+len(value) > MaximumContentBytes {
			return ErrTooLarge
		}
		content.WriteString(value)
		return nil
	}
	if len(publication.Authors) > 0 {
		if err := write("**Author:** " + strings.Join(publication.Authors, ", ") + "\n\n"); err != nil {
			return Note{}, err
		}
	}
	if publication.Series != "" {
		series := publication.Series
		if publication.SeriesIndex != nil {
			series += fmt.Sprintf(" · %g", *publication.SeriesIndex)
		}
		if err := write("**Series:** " + series + "\n\n"); err != nil {
			return Note{}, err
		}
	}
	for _, item := range visible {
		heading := "Highlight"
		if item.Kind == "bookmark" {
			heading = "Bookmark"
		}
		if item.Kind == "note" {
			heading = "Note"
		}
		if item.Chapter != "" {
			heading += " · " + singleLine(item.Chapter)
		}
		if err := write("## " + heading + "\n\n"); err != nil {
			return Note{}, err
		}
		if quote := strings.TrimSpace(item.Quote); quote != "" {
			for _, line := range strings.Split(quote, "\n") {
				if err := write("> " + line + "\n"); err != nil {
					return Note{}, err
				}
			}
			if err := write("\n"); err != nil {
				return Note{}, err
			}
		}
		if note := strings.TrimSpace(item.Note); note != "" {
			if err := write(note + "\n\n"); err != nil {
				return Note{}, err
			}
		}
		if item.UpdatedAt > 0 {
			if err := write("_Saved " + time.UnixMilli(item.UpdatedAt).UTC().Format(time.RFC3339) + "_\n\n"); err != nil {
				return Note{}, err
			}
		}
	}
	return Note{Title: title, Content: strings.TrimSpace(content.String()) + "\n"}, nil
}

func singleLine(value string) string { return strings.Join(strings.Fields(value), " ") }

func boundedUTF8(value string, maximum int) string {
	if len(value) <= maximum {
		return value
	}
	for len(value) > maximum {
		_, width := utf8.DecodeLastRuneInString(value)
		if width == 0 {
			return ""
		}
		value = value[:len(value)-width]
	}
	return value
}
