package audio

import "testing"

func TestAudioDiscFolderSharedNormalizer(t *testing.T) {
	for name, want := range map[string]bool{
		" CD 1 ": true, "Disc_2": true, "DISK-03": true,
		"disc0": false, "disc": false, "Albums": false,
		"disc one": false, "CD 2 extras": false,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for i := 0; i < 100; i++ {
				if got := IsDiscFolder(name); got != want {
					t.Fatalf("got %v want %v", got, want)
				}
			}
		})
	}
}
