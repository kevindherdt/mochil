package transcode

import (
	"strings"
	"testing"
)

func TestManifestCoversCompleteBoundedTimeline(t *testing.T) {
	manifest := string(Manifest(4.25, 2))
	if strings.Count(manifest, "#EXTINF:") != 3 || !strings.Contains(manifest, "#EXTINF:0.250000,\nseg00002.m4s") || !strings.HasSuffix(manifest, "#EXT-X-ENDLIST\n") {
		t.Fatalf("manifest = %q", manifest)
	}
}

func TestTransportStreamManifestHasSelfContainedSegments(t *testing.T) {
	manifest := string(TransportStreamManifest(4, 2))
	if strings.Contains(manifest, "EXT-X-MAP") || !strings.Contains(manifest, "seg00000.ts") || !strings.Contains(manifest, "#EXT-X-VERSION:3") {
		t.Fatalf("transport stream manifest = %q", manifest)
	}
}

func TestRewriteManifestAddressesOnlyMediaResources(t *testing.T) {
	manifest := []byte("#EXTM3U\n#EXT-X-MAP:URI=\"init.mp4\"\nseg00000.m4s\n#comment\n")
	got := string(RewriteManifest(manifest, "/compat/item/"))
	if !strings.Contains(got, `URI="/compat/item/init.mp4"`) || !strings.Contains(got, "\n/compat/item/seg00000.m4s\n") || !strings.Contains(got, "\n#comment\n") {
		t.Fatalf("rewritten manifest = %q", got)
	}
}
