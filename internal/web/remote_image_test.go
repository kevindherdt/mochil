package web

import "testing"

func TestRemoteImageTypesExcludeScriptableFormats(t *testing.T) {
	for _, accepted := range []string{"image/png", "IMAGE/JPEG; charset=binary", " image/webp "} {
		if _, ok := rasterImageType(accepted); !ok {
			t.Errorf("%q should be accepted", accepted)
		}
	}
	for _, refused := range []string{"", "image/svg+xml", "image/svg+xml; charset=utf-8", "text/html", "image/", "application/octet-stream"} {
		if _, ok := rasterImageType(refused); ok {
			t.Errorf("%q should be refused", refused)
		}
	}
}

func TestRemoteMediaTypeNeverServesPages(t *testing.T) {
	for in, want := range map[string]string{
		"audio/mpeg": "audio/mpeg", "Audio/MP4; codecs=x": "Audio/MP4; codecs=x", "video/mp4": "video/mp4",
		"": "application/octet-stream", "text/html": "application/octet-stream", "image/svg+xml": "application/octet-stream", "audio/": "application/octet-stream",
	} {
		if got := remoteMediaType(in); got != want {
			t.Errorf("remoteMediaType(%q) = %q, want %q", in, got, want)
		}
	}
}
