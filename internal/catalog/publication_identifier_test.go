package catalog

import "testing"

func TestPublicationIdentifierNormalizationIsConservative(t *testing.T) {
	for input, want := range map[string]string{
		"isbn-10: 123-456-789-X":     "123456789x",
		"ISBN-13: 978-1-234-56789-0": "9781234567890",
		"urn:isbn:123456789X":        "123456789x",
		"provider:ab-cd":             "provider:ab-cd",
		"isbn:other-provider":        "isbn:other-provider",
		"12345678X9":                 "12345678x9",
		"1234-5678X9":                "1234-5678x9",
		"isbn:isbn:123456789X":       "isbn:isbn:123456789x",
	} {
		if got := normalizePublicationIdentifier(input); got != want {
			t.Errorf("%q normalized to %q, want %q", input, got, want)
		}
	}
}

func TestPreferredFormatsPreservesProviderIdentifierPunctuation(t *testing.T) {
	for _, test := range []struct {
		first, second string
		count         int
	}{
		{"provider:ab-cd", "provider:abcd", 2},
		{"isbn:other-provider", "other-provider", 2},
		{"urn:isbn:123-456-789-X", "123456789x", 1},
		{"ISBN-13:978-1-234-56789-0", "9781234567890", 1},
	} {
		backend := &preferredTestBackend{books: []Book{
			{ID: "one", Title: "Shared", Authors: []string{"Writer"}, Format: "epub", Identifier: test.first},
			{ID: "two", Title: "Shared", Authors: []string{"Writer"}, Format: "pdf", Identifier: test.second},
		}}
		if got := PreferFormats(backend, []string{"epub", "pdf"}).Books(); len(got) != test.count {
			t.Errorf("%q / %q: got %d publications, want %d", test.first, test.second, len(got), test.count)
		}
	}
}
