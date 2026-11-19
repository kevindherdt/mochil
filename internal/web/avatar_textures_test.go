package web

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeTexture(t *testing.T, path string, width, height int) {
	t.Helper()
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x += 2 {
		picture.Set(x, 0, color.NRGBA{R: 200, G: 40, B: 40, A: 255})
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, picture); err != nil {
		t.Fatal(err)
	}
}

func TestABigAvatarTextureIsServedSmallerAndASmallOneIsLeftAlone(t *testing.T) {
	directory := t.TempDir()
	big, small, cache := filepath.Join(directory, "big.png"), filepath.Join(directory, "small.png"), filepath.Join(directory, "cache")
	writeTexture(t, big, 4096, 2048)
	writeTexture(t, small, 1024, 1024)
	if got := avatarTextureForPhones(cache, small, 1, 1); got != small {
		t.Fatalf("a small texture must be served as it is: %s", got)
	}
	got := avatarTextureForPhones(cache, big, 1, 1)
	if got == big {
		t.Fatal("a big texture must be served smaller")
	}
	file, err := os.Open(got)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil || config.Width != 2048 || config.Height != 1024 {
		t.Fatalf("size = %+v, %v (the shape must be kept)", config, err)
	}
	if again := avatarTextureForPhones(cache, big, 1, 1); again != got {
		t.Fatal("the smaller copy must be reused")
	}
	if original, _ := os.Stat(big); original == nil || original.Size() == 0 {
		t.Fatal("the original must be left alone")
	}
	if got := avatarTextureForPhones("", big, 1, 1); got != big {
		t.Fatal("with nowhere to keep copies the original is served")
	}
}
