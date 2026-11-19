package web

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sync"

	xdraw "golang.org/x/image/draw"
)

// A phone's graphics chip cannot hold the textures some Live2D models come with (one 8192x8192 picture is 256 MB once
// it is on the chip), so the avatar stays blank there. The picture is served at most this many pixels on a side. The
// model files use relative coordinates, so a smaller picture fits the same model, and the original is never touched.
const maximumAvatarTexture = 2048

var avatarTextureLock sync.Mutex // one big picture is decoded at a time, so memory stays bounded

// avatarTextureForPhones returns the file to serve for a texture: the original when it is small enough (or cannot be
// shrunk), otherwise a smaller copy kept in the cache folder.
func avatarTextureForPhones(cacheDirectory, original string, size int64, modified int64) string {
	if cacheDirectory == "" {
		return original
	}
	file, err := os.Open(original)
	if err != nil {
		return original
	}
	config, _, err := image.DecodeConfig(file)
	file.Close()
	if err != nil || (config.Width <= maximumAvatarTexture && config.Height <= maximumAvatarTexture) {
		return original
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%d\x00%d", original, size, modified, maximumAvatarTexture)))
	cached := filepath.Join(cacheDirectory, hex.EncodeToString(sum[:12])+".png")
	if _, err := os.Stat(cached); err == nil {
		return cached
	}
	avatarTextureLock.Lock()
	defer avatarTextureLock.Unlock()
	if _, err := os.Stat(cached); err == nil {
		return cached
	}
	if shrinkTexture(original, cached, config) != nil {
		return original
	}
	return cached
}

func shrinkTexture(original, target string, config image.Config) error {
	file, err := os.Open(original)
	if err != nil {
		return err
	}
	decoded, err := png.Decode(file)
	file.Close()
	if err != nil {
		return err
	}
	// Premultiplied pixels, or the edges of the artwork pick up a coloured fringe when shrunk.
	source := image.NewRGBA(decoded.Bounds())
	draw.Draw(source, source.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	decoded = nil
	width, height := config.Width, config.Height
	if width >= height {
		height, width = height*maximumAvatarTexture/width, maximumAvatarTexture
	} else {
		width, height = width*maximumAvatarTexture/height, maximumAvatarTexture
	}
	smaller := image.NewRGBA(image.Rect(0, 0, max(width, 1), max(height, 1)))
	xdraw.CatmullRom.Scale(smaller, smaller.Bounds(), source, source.Bounds(), xdraw.Src, nil)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	temporary := target + ".tmp"
	out, err := os.Create(temporary)
	if err != nil {
		return err
	}
	if err := png.Encode(out, smaller); err != nil {
		out.Close()
		os.Remove(temporary)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(temporary)
		return err
	}
	return os.Rename(temporary, target)
}
