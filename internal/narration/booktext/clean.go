package booktext

import (
	"regexp"
	"strings"
)

// What is not worth reading aloud: copyright notices, catalogue records, web
// addresses and advertisements for other books. Kept in step with
// narration-clean.js in the web reader, which makes the same choices.
var noise = []*regexp.Regexp{
	regexp.MustCompile(`(?i)https?://|\bwww\.|\b[\w.+-]+@[\w-]+\.[a-z]{2,}\b|\b[\w-]+\.(?:com|net|org|co\.uk|com\.au)\b`),
	regexp.MustCompile(`(?i)©|\(c\)\s*\d{4}|\bISBN\b|\ball rights reserved\b|\bLibrary of Congress\b|\bCataloging-in-Publication\b|^(?:printed in|first (?:published|edition|ebook edition)|cover (?:design|illustration|art)|ebook edition|source isbn)\b`),
	regexp.MustCompile(`(?i)\b(?:newsletter|sign up for|discover the world|don['’]t miss)\b`),
}

// WorthReading is false for a sentence that should be skipped.
func WorthReading(text string) bool {
	for _, pattern := range noise {
		if pattern.MatchString(text) {
			return false
		}
	}
	return true
}

var matterTitle = regexp.MustCompile(`(?i)^(?:copyright(?: page)?|title page|half title|contents|table of contents|also by[\w\s.,'’-]*|books by[\w\s.,'’-]*|about the (?:author|publisher)s?|about the translator|acknowledg(?:e)?ments?|praise(?: for[\w\s.,'’:-]*)?|newsletter|discover more[\w\s.,'’-]*|colophon|credits|permissions|bibliography|index|excerpt|preview|sneak peek|read on|coming soon|ebook extras?)\s*$`)

// IsMatterTitle is true for a chapter that is front or back matter, going by its title.
func IsMatterTitle(title string) bool {
	return matterTitle.MatchString(strings.Join(strings.Fields(title), " "))
}
