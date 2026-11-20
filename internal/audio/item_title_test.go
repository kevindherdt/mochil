package audio

import (
	"testing"
	"time"
)

func TestIdentifyStripsOriginalExtensionCase(t *testing.T) {
	for _, extension := range []string{"flac", "FLAC", "FlAc", "MP3", "M4B"} {
		t.Run(extension, func(t *testing.T) {
			t.Parallel()
			file := "Album/My_Song.Live." + extension
			item, ok := Identify("music", file, 100, time.Unix(10, 0))
			if !ok || item.Title != "My Song Live" || item.Path != file {
				t.Fatalf("unexpected identification: %+v accepted=%v", item, ok)
			}
			repeated, _ := Identify("music", file, 100, time.Unix(10, 0))
			if item.ID != repeated.ID || item.Revision != repeated.Revision {
				t.Fatal("unstable identity")
			}
		})
	}
}
