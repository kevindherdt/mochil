package catalog

import (
	"sort"
	"strconv"
	"strings"
)

// publicationPartitions returns conservative edition-aware grouping keys.
// Missing language or identifier data can join one known edition, but becomes
// ambiguous—and therefore stays visible—when a cohort contains conflicts.
func publicationPartitions(books []Book) map[int]string {
	cohorts := make(map[string][]int)
	for index, book := range books {
		if key := publicationGroupKey(book); key != "" {
			cohorts[key] = append(cohorts[key], index)
		}
	}
	partitions := make(map[int]string, len(books))
	for base, indexes := range cohorts {
		languages, identifiers := map[string]bool{}, map[string]bool{}
		for _, index := range indexes {
			if value := normalizeIdentity(books[index].Language); value != "" {
				languages[value] = true
			}
			if value := normalizePublicationIdentifier(books[index].Identifier); value != "" {
				identifiers[value] = true
			}
		}
		for _, index := range indexes {
			book := books[index]
			key := base
			if len(languages) > 1 {
				key += "\x00language:" + discriminator(normalizeIdentity(book.Language), book, "language")
			}
			if len(identifiers) > 1 {
				key += "\x00identifier:" + discriminator(normalizePublicationIdentifier(book.Identifier), book, "identifier")
			}
			partitions[index] = key
		}
	}
	return partitions
}

func discriminator(value string, book Book, kind string) string {
	if value != "" {
		return value
	}
	identity := book.ID
	if identity == "" {
		identity = book.Path
	}
	return "unknown-" + kind + ":" + identity
}

var publicationIdentifierSeparators = strings.NewReplacer("-", "", " ", "")

func normalizePublicationIdentifier(value string) string {
	value = normalizeIdentity(value)
	candidate := value
	for _, prefix := range []string{"urn:isbn:", "isbn:", "isbn-10:", "isbn-13:"} {
		if strings.HasPrefix(candidate, prefix) {
			candidate = strings.TrimPrefix(candidate, prefix)
			break
		}
	}
	compact := publicationIdentifierSeparators.Replace(candidate)
	if len(compact) != 10 && len(compact) != 13 {
		return value
	}
	// Only ISBN-shaped values lose separators. Other providers may assign
	// distinct identifiers differing solely in punctuation. This is format
	// normalization, not checksum validation or ISBN-10/13 conversion.
	for i := range compact {
		if compact[i] >= '0' && compact[i] <= '9' {
			continue
		}
		if len(compact) == 10 && i == 9 && compact[i] == 'x' {
			continue
		}
		return value
	}
	return compact
}

func publicationGroupKey(book Book) string {
	title := normalizeIdentity(book.Title)
	series := normalizeIdentity(book.Series)
	authors := make([]string, 0, len(book.Authors))
	for _, author := range book.Authors {
		value := normalizeIdentity(author)
		if value != "" && value != "unknown" && value != "unknown author" {
			authors = append(authors, value)
		}
	}
	sort.Strings(authors)
	if title == "" || (len(authors) == 0 && (series == "" || book.SeriesIndex == nil)) {
		return ""
	}
	index := ""
	if book.SeriesIndex != nil {
		index = strconv.FormatFloat(*book.SeriesIndex, 'g', -1, 64)
	}
	role := book.PublicationRole
	if role == "" {
		role = ClassifyPublication(book.Title, book.Path)
	}
	return title + "\x00" + series + "\x00" + index + "\x00" + strings.Join(authors, "&") + "\x00" + string(role)
}

func normalizeIdentity(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}
