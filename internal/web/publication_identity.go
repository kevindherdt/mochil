package web

import (
	"math"
	"regexp"
	"strings"
	"unicode"

	"mochil/internal/catalog"
)

type publicationIdentityEvidence struct {
	Title       string
	Authors     string
	Identifiers string
	Series      string
	SeriesIndex *float64
}

var identifierParts = regexp.MustCompile(`[\s,;|]+`)

func publicationIdentityMatchesBook(evidence publicationIdentityEvidence, book catalog.Book) bool {
	if identifiersOverlap(evidence.Identifiers, book.Identifier) {
		return true
	}
	title := identityText(evidence.Title)
	if title != "" && title == identityText(book.Title) && authorMatches(evidence.Authors, book.Authors) {
		return true
	}
	return identityText(evidence.Series) != "" &&
		identityText(evidence.Series) == identityText(book.Series) &&
		equalSeriesIndex(evidence.SeriesIndex, book.SeriesIndex) &&
		authorMatches(evidence.Authors, book.Authors)
}

func identityText(value string) string {
	var normalized strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			normalized.WriteRune(r)
		}
	}
	return normalized.String()
}

func authorMatches(value string, authors []string) bool {
	wanted := identityText(value)
	if wanted == "" {
		return false
	}
	for _, author := range authors {
		candidate := identityText(author)
		if candidate != "" && (candidate == wanted || strings.Contains(wanted, candidate)) {
			return true
		}
	}
	return false
}

func identifiersOverlap(left, right string) bool {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return false
	}
	rightIDs := make(map[string]struct{})
	for _, value := range identifierParts.Split(right, -1) {
		if normalized := normalizeIdentifier(value); normalized != "" {
			rightIDs[normalized] = struct{}{}
		}
	}
	for _, value := range identifierParts.Split(left, -1) {
		if normalized := normalizeIdentifier(value); normalized != "" {
			if _, exists := rightIDs[normalized]; exists {
				return true
			}
		}
	}
	return false
}

func normalizeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if separator := strings.LastIndexByte(value, ':'); separator >= 0 {
		value = value[separator+1:]
	}
	return identityText(value)
}

func equalSeriesIndex(left, right *float64) bool {
	return left != nil && right != nil && !math.IsNaN(*left) && !math.IsNaN(*right) && math.Abs(*left-*right) < .0001
}
