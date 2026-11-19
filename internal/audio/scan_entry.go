package audio

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// audioScanInfo avoids stat calls for sidecars and AppleDouble files. The walk
// owns directory/symlink handling; Identify still validates paths and file size.
func audioScanInfo(entry fs.DirEntry) (fs.FileInfo, error) {
	name := entry.Name()
	if _, supported := formats[strings.ToLower(filepath.Ext(name))]; !supported || strings.HasPrefix(name, "._") {
		return nil, nil
	}
	return entry.Info()
}
