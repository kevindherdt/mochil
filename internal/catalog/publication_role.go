package catalog

import "strings"

// PublicationRole distinguishes material that belongs to a series without
// pretending it occupies a numbered place in the main reading order.
type PublicationRole string

const (
	PublicationSideStory PublicationRole = "side_story"
	PublicationPreface   PublicationRole = "preface"
	PublicationGlossary  PublicationRole = "glossary"
	PublicationOmnibus   PublicationRole = "omnibus"
	PublicationCompanion PublicationRole = "companion"
)

var publicationRoleSeparators = strings.NewReplacer(
	"_", " ", "-", " ", "/", " ", "\\", " ", "!", " ",
)

// ClassifyPublication is deliberately conservative: an empty role means the
// publication remains part of the ordinary reading order.
func ClassifyPublication(title, path string) PublicationRole {
	value := " " + strings.Join(strings.Fields(publicationRoleSeparators.Replace(strings.ToLower(title+" "+path))), " ") + " "
	contains := func(phrases ...string) bool {
		for _, phrase := range phrases {
			if strings.Contains(value, " "+phrase+" ") {
				return true
			}
		}
		return false
	}
	switch {
	case contains("side story", "side stories", "short story", "short stories", "gaiden", "bonus story", "bonus stories", "backer stories"):
		return PublicationSideStory
	case contains("preface"):
		return PublicationPreface
	case contains("glossary"):
		return PublicationGlossary
	case contains("omnibus"):
		return PublicationOmnibus
	case contains("fanbook", "artbook", "art book", "guidebook", "companion"):
		return PublicationCompanion
	default:
		return ""
	}
}
