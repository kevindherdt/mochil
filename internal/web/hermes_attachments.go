package web

// Turn a chat attachment into the content part Hermes expects. Split out of hermes_http.go, which is at its line
// budget, and because the allow-list below is the part of attachments most likely to change.

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func hermesAttachmentParts(attachments []hermesAttachment) ([]any, string) {
	if len(attachments) == 0 {
		return nil, ""
	}
	if len(attachments) > maximumHermesAttachments {
		return nil, fmt.Sprintf("Send at most %d files at once.", maximumHermesAttachments)
	}
	parts := make([]any, 0, len(attachments))
	total := 0
	for _, attachment := range attachments {
		name := strings.TrimSpace(attachment.Name)
		if name == "" {
			name = "attachment"
		}
		data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(attachment.Data))
		if err != nil || len(data) == 0 {
			return nil, fmt.Sprintf("%q could not be read.", name)
		}
		if len(data) > maximumHermesAttachmentBytes {
			return nil, fmt.Sprintf("%q is larger than %d MB.", name, maximumHermesAttachmentBytes/1_000_000)
		}
		total += len(data)
		if total > maximumHermesAttachmentTotal {
			return nil, fmt.Sprintf("Those files are larger than %d MB together.", maximumHermesAttachmentTotal/1_000_000)
		}
		mediaType := strings.ToLower(strings.TrimSpace(attachment.MediaType))
		switch {
		case strings.HasPrefix(mediaType, "image/"):
			parts = append(parts, map[string]any{
				"type":      "image_url",
				"image_url": map[string]string{"url": "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(data)},
			})
		case strings.HasPrefix(mediaType, "text/") || mediaType == "application/json" || mediaType == "application/xml" || isPlainTextName(name):
			text := strings.TrimSpace(string(data))
			// A note or a CSV can be long; the tail is what gets dropped rather than the whole file.
			if utf8.RuneCountInString(text) > 200_000 {
				text = string([]rune(text)[:200_000]) + "\n\n[truncated]"
			}
			parts = append(parts, map[string]any{"type": "text", "text": "Attached file " + name + ":\n\n" + text})
		default:
			return nil, fmt.Sprintf("%q is a %s file — only images and plain text can be sent today.", name, mediaType)
		}
	}
	return parts, ""
}

// isPlainTextName catches the common text files a browser reports with an unhelpful media type (text/plain is usual,
// but some send application/octet-stream for .md and .csv).
func isPlainTextName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt", ".md", ".markdown", ".csv", ".json", ".yaml", ".yml", ".log", ".toml", ".ini":
		return true
	default:
		return false
	}
}

// hermesChat sends one message into a Hermes session and relays the streamed reply (server-sent events). The request
// to Hermes runs to the end even if the browser goes away, so the reply is saved in the session and any device can
// open it afterwards.
