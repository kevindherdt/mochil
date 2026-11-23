package web

import (
	"encoding/base64"
	"strings"
	"testing"
)

// Attachment handling has one rule per media type and several ways to go wrong silently, so each of those gets a
// test: text is inlined (works on every model), images ride as data URLs, and anything else is refused with a
// reason rather than dropped — a person who attached a PDF has to be told it was not sent.
func TestHermesAttachmentParts(t *testing.T) {
	encode := func(value string) string { return base64.StdEncoding.EncodeToString([]byte(value)) }

	t.Run("plain text is inlined", func(t *testing.T) {
		parts, problem := hermesAttachmentParts([]hermesAttachment{
			{Name: "note.md", MediaType: "application/octet-stream", Data: encode("# hello\nbody")},
		})
		if problem != "" {
			t.Fatalf("problem = %q", problem)
		}
		if len(parts) != 1 {
			t.Fatalf("parts = %d", len(parts))
		}
		part, _ := parts[0].(map[string]any)
		text, _ := part["text"].(string)
		if !strings.Contains(text, "note.md") || !strings.Contains(text, "body") {
			t.Fatalf("text part = %q", text)
		}
	})

	t.Run("images become data URLs", func(t *testing.T) {
		parts, problem := hermesAttachmentParts([]hermesAttachment{
			{Name: "shot.png", MediaType: "image/png", Data: encode("\x89PNG")},
		})
		if problem != "" || len(parts) != 1 {
			t.Fatalf("problem = %q, parts = %d", problem, len(parts))
		}
		part, _ := parts[0].(map[string]any)
		if part["type"] != "image_url" {
			t.Fatalf("type = %v", part["type"])
		}
		image, _ := part["image_url"].(map[string]string)
		if !strings.HasPrefix(image["url"], "data:image/png;base64,") {
			t.Fatalf("url = %q", image["url"])
		}
	})

	t.Run("a pdf is refused with a reason", func(t *testing.T) {
		_, problem := hermesAttachmentParts([]hermesAttachment{
			{Name: "tax.pdf", MediaType: "application/pdf", Data: encode("%PDF-1.4")},
		})
		if !strings.Contains(problem, "tax.pdf") || !strings.Contains(problem, "plain text") {
			t.Fatalf("problem = %q", problem)
		}
	})

	t.Run("too many files", func(t *testing.T) {
		many := make([]hermesAttachment, maximumHermesAttachments+1)
		for index := range many {
			many[index] = hermesAttachment{Name: "a.txt", MediaType: "text/plain", Data: encode("x")}
		}
		if _, problem := hermesAttachmentParts(many); problem == "" {
			t.Fatal("expected a refusal")
		}
	})

	t.Run("broken base64 is refused", func(t *testing.T) {
		if _, problem := hermesAttachmentParts([]hermesAttachment{{Name: "a.png", MediaType: "image/png", Data: "not base64!!"}}); problem == "" {
			t.Fatal("expected a refusal")
		}
	})
}
