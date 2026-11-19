// Package transcode owns media-family-neutral segmented compatibility-stream
// primitives. Audio and video adapters provide codec arguments and source
// identity; this package owns the HLS timeline and resource addressing rules.
package transcode

import (
	"fmt"
	"regexp"
	"strings"
)

var SegmentName = regexp.MustCompile(`^seg[0-9]{5}\.(?:m4s|ts)$`)

func Manifest(duration float64, segmentSeconds int) []byte {
	return manifest(duration, segmentSeconds, true)
}

func TransportStreamManifest(duration float64, segmentSeconds int) []byte {
	return manifest(duration, segmentSeconds, false)
}

func manifest(duration float64, segmentSeconds int, fragmentedMP4 bool) []byte {
	if segmentSeconds < 1 {
		segmentSeconds = 2
	}
	if duration <= 0 {
		duration = float64(segmentSeconds)
	}
	count := int(duration+float64(segmentSeconds)-0.000001) / segmentSeconds
	var out strings.Builder
	version := 3
	if fragmentedMP4 {
		version = 7
	}
	fmt.Fprintf(&out, "#EXTM3U\n#EXT-X-VERSION:%d\n#EXT-X-TARGETDURATION:%d\n#EXT-X-MEDIA-SEQUENCE:0\n#EXT-X-PLAYLIST-TYPE:VOD\n", version, segmentSeconds)
	if fragmentedMP4 {
		out.WriteString("#EXT-X-MAP:URI=\"init.mp4\"\n")
	}
	extension := ".ts"
	if fragmentedMP4 {
		extension = ".m4s"
	}
	for index := 0; index < count; index++ {
		length := float64(segmentSeconds)
		if remaining := duration - float64(index*segmentSeconds); remaining < length {
			length = remaining
		}
		if length < .1 {
			length = .1
		}
		fmt.Fprintf(&out, "#EXTINF:%.6f,\nseg%05d%s\n", length, index, extension)
	}
	out.WriteString("#EXT-X-ENDLIST\n")
	return []byte(out.String())
}

func RewriteManifest(data []byte, base string) []byte {
	return RewriteManifestVersioned(data, base, "")
}

func RewriteManifestVersioned(data []byte, base, version string) []byte {
	suffix := ""
	if version != "" {
		suffix = "?v=" + version
	}
	lines := strings.Split(string(data), "\n")
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "init.mp4" || SegmentName.MatchString(trimmed) {
			lines[index] = base + trimmed + suffix
		}
		if strings.Contains(line, `URI="init.mp4"`) {
			lines[index] = strings.ReplaceAll(line, `URI="init.mp4"`, `URI="`+base+`init.mp4`+suffix+`"`)
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
