package video

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPairCuesAlternatingInkAndBlank(t *testing.T) {
	frames := []frameStat{{Time: 10}, {Time: 12, Ink: true}, {Time: 14}, {Time: 20, Ink: true}, {Time: 23}}

	spans := pairCues(frames)
	if len(spans) != 2 {
		t.Fatalf("expected 2 cues, got %d", len(spans))
	}
	if spans[0].Start != 1 || spans[0].End != 2 {
		t.Fatalf("first cue = %+v, want frame 1 closing at frame 2", spans[0])
	}
	if spans[1].Start != 3 || spans[1].End != 4 {
		t.Fatalf("second cue = %+v, want frame 3 closing at frame 4", spans[1])
	}
}

func TestPairCuesClosesACueReplacedWithoutABlankFrame(t *testing.T) {
	// PGS updates a line in place without an empty display set between them.
	frames := []frameStat{{Time: 5, Ink: true}, {Time: 6, Ink: true}, {Time: 8}}

	spans := pairCues(frames)
	if len(spans) != 2 {
		t.Fatalf("expected the replacement to end the first cue, got %d cues", len(spans))
	}
	if spans[0].End != 1 {
		t.Fatalf("first cue should end where the replacement starts, got %+v", spans[0])
	}
}

func TestPairCuesLeavesATrailingCueOpen(t *testing.T) {
	frames := []frameStat{{Time: 1}, {Time: 2, Ink: true}}

	spans := pairCues(frames)
	if len(spans) != 1 || spans[0].End != -1 {
		t.Fatalf("a cue with no hide frame stays open, got %+v", spans)
	}
}

func TestReadFrameStatsReadsTimeAndInk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frames.txt")
	content := "frame:0    pts:141559  pts_time:141.559\n" +
		"lavfi.signalstats.YMIN=0\n" +
		"lavfi.signalstats.YMAX=0\n" +
		"frame:1    pts:144102  pts_time:144.102\n" +
		"lavfi.signalstats.YMIN=0\n" +
		"lavfi.signalstats.YMAX=235\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	frames, err := readFrameStats(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
	if frames[0].Time != 141.559 || frames[0].Ink {
		t.Fatalf("first frame = %+v, want 141.559 with no ink", frames[0])
	}
	if frames[1].Time != 144.102 || !frames[1].Ink {
		t.Fatalf("second frame = %+v, want 144.102 with ink", frames[1])
	}
}

func TestBoxAfterTakesTheBoxFromTheFollowingFrame(t *testing.T) {
	// cropdetect reports the caption's box on the frame after the one it measured.
	frames := []frameStat{
		{Time: 1, Ink: true},
		{Time: 1.1, X: 656, Y: 828, W: 606, H: 140},
	}

	box := boxAfter(frames, 0)
	if box.W != 606 || box.H != 140 || box.X != 656 || box.Y != 828 {
		t.Fatalf("box = %+v, want the following frame's box", box)
	}
}

func TestBoxAfterFallsBackToItsOwnFrame(t *testing.T) {
	frames := []frameStat{{Time: 1, Ink: true, X: 10, Y: 20, W: 30, H: 40}}

	box := boxAfter(frames, 0)
	if box.W != 30 {
		t.Fatalf("box = %+v, want the frame's own box", box)
	}
}
