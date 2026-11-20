package catalog

import "testing"

func TestClassifyPublicationIsConservative(t *testing.T) {
	cases := map[string]PublicationRole{
		"Saga/!Side Stories/After.epub": PublicationSideStory,
		"Saga Omnibus Edition":          PublicationOmnibus,
		"World Guidebook":               PublicationCompanion,
		"Series Preface":                PublicationPreface,
		"Series Glossary":               PublicationGlossary,
		"The Companionable Hero":        "",
		"Ordinary Volume 2":             "",
	}
	for value, expected := range cases {
		if actual := ClassifyPublication(value, value); actual != expected {
			t.Errorf("ClassifyPublication(%q)=%q want %q", value, actual, expected)
		}
	}
}
