package catalog

import (
	"strings"
	"testing"
)

func TestPublicationSharedNormalizers(t *testing.T) {
	for _, input := range []string{"", "A_side-story!", "Folder/Bonus\\Stories", "Straße_日本語", "\tA-B_C/!D\n"} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			want := strings.NewReplacer("_", " ", "-", " ", "/", " ", "\\", " ", "!", " ").Replace(input)
			for i := 0; i < 100; i++ {
				if got := publicationRoleSeparators.Replace(input); got != want {
					t.Fatalf("role normalization got %q want %q", got, want)
				}
			}
		})
	}
	for input, want := range map[string]string{
		"ISBN: 978-1-234-56789-0": "9781234567890",
		"urn:isbn:123456789X":     "123456789x",
		"custom-provider:ab-cd":   "custom-provider:ab-cd", "": "",
		"provider:ab-cd": "provider:ab-cd",
	} {
		if got := normalizePublicationIdentifier(input); got != want {
			t.Fatalf("identifier %q got %q want %q", input, got, want)
		}
	}
}

func BenchmarkPublicationRoleSeparators(b *testing.B) {
	input := "The_Series/Volume-01/Bonus_Stories!"
	for _, shared := range []bool{false, true} {
		name := "previous"
		if shared {
			name = "shared"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				replacer := publicationRoleSeparators
				if !shared {
					replacer = strings.NewReplacer("_", " ", "-", " ", "/", " ", "\\", " ", "!", " ")
				}
				if replacer.Replace(input) == "" {
					b.Fatal("empty result")
				}
			}
		})
	}
}
