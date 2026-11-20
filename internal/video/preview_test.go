package video

import "testing"

func TestPreviewPositionClampsAndQuantizes(t *testing.T) {
	item := Item{Duration: 95.5}
	tests := []struct {
		requested int
		want      int
	}{
		{0, 0},
		{9, 0},
		{10, 10},
		{94, 90},
		{500, 90},
	}
	for _, test := range tests {
		got, err := previewPosition(item, test.requested)
		if err != nil || got != test.want {
			t.Fatalf("previewPosition(%d) = %d, %v; want %d", test.requested, got, err, test.want)
		}
	}
	if _, err := previewPosition(item, -1); err == nil {
		t.Fatal("negative position was accepted")
	}
	if _, err := previewPosition(Item{}, 0); err == nil {
		t.Fatal("unknown duration was accepted")
	}
}

func TestPreviewKeyIncludesRevisionAndPosition(t *testing.T) {
	item := Item{LibraryID: "movies", ID: "one", Revision: "first"}
	if previewKey(item, 10) == previewKey(item, 20) {
		t.Fatal("position did not change preview key")
	}
	changed := item
	changed.Revision = "second"
	if previewKey(item, 10) == previewKey(changed, 10) {
		t.Fatal("revision did not change preview key")
	}
}
