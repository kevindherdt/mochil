package web

import (
	"testing"

	"mochil/internal/metadata"
)

func TestMetadataApplyResponseDescribesFilesystemImpact(t *testing.T) {
	response := newMetadataApplyResponse(metadata.Job{ID: "review"}, "filesystem", 2, 1, metadataApplyEffects{Scanned: 324})
	if response.ApplyMode != "filesystem" || response.AppliedBooks != 3 || response.SourceBooks != 2 || response.OverrideBooks != 1 {
		t.Fatalf("unexpected counts: %#v", response)
	}
	if !response.PublicationArchivesChanged || response.ComicInfoChanged || response.ScanBooks != 324 {
		t.Fatalf("unexpected file effects: %#v", response)
	}
}

func TestMetadataApplyResponseDescribesCalibreWithoutCallingRecordsSourceFiles(t *testing.T) {
	response := newMetadataApplyResponse(metadata.Job{ID: "review"}, "calibre-database", 2, 0, metadataApplyEffects{Warning: "refresh failed"})
	if response.ApplyMode != "calibre-database" || response.SourceBooks != 2 || response.AppliedBooks != 2 {
		t.Fatalf("unexpected Calibre counts: %#v", response)
	}
	if response.PublicationArchivesChanged || response.ComicInfoChanged || response.ScanWarning != "refresh failed" {
		t.Fatalf("unexpected Calibre file effects: %#v", response)
	}
}

func TestMetadataApplyResponseDescribesCatalogueOverridesAsNonMutating(t *testing.T) {
	response := newMetadataApplyResponse(metadata.Job{ID: "review"}, "catalogue-overrides", 0, 4, metadataApplyEffects{})
	if response.SourceBooks != 0 || response.OverrideBooks != 4 || response.PublicationArchivesChanged || response.ComicInfoChanged {
		t.Fatalf("unexpected override effects: %#v", response)
	}
}
