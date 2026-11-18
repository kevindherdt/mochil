package audio

import (
	"strconv"
	"strings"
)

var discFolderSeparators = strings.NewReplacer(" ", "", "-", "", "_", "")

// IsDiscFolder recognizes numbered disc directories shared by album browsing
// and metadata inspection. It does not treat arbitrary child folders as discs.
func IsDiscFolder(name string) bool {
	cleaned := discFolderSeparators.Replace(strings.ToLower(strings.TrimSpace(name)))
	for _, prefix := range []string{"cd", "disc", "disk"} {
		if strings.HasPrefix(cleaned, prefix) {
			number, err := strconv.Atoi(strings.TrimPrefix(cleaned, prefix))
			return err == nil && number > 0
		}
	}
	return false
}
