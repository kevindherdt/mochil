package web

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCoverCacheMaintainLeavesSourceWhenDestinationStatFails(t *testing.T) {
	cache := newCoverCache(t.TempDir(), 1<<30, 300, 450, 75, nil)
	legacy := cache.legacyPath("books", "loop")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, testJPEG(t, 400, 600, 11), 0o600); err != nil {
		t.Fatal(err)
	}
	destination := cache.path("books", "loop")
	if err := os.Symlink(destination, destination); err != nil {
		t.Fatal(err)
	}

	cache.maintain(context.Background())

	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("maintenance removed the source after destination stat failed: %v", err)
	}
	if info, err := os.Lstat(destination); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("maintenance replaced the uninspectable destination: info=%v err=%v", info, err)
	}
}

func TestCoverCacheMaintainSkipsEvictionScanWhenCacheIsStable(t *testing.T) {
	cache := newCoverCache(t.TempDir(), 1<<30, 300, 450, 75, nil)
	path := cache.path("books", "stable")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	data := []byte("already optimized")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cache.mu.Lock()
	cache.size = int64(len(data))
	cache.sizeReady = true
	cache.mu.Unlock()

	evictions := 0
	cache.evictHook = func() { evictions++ }
	cache.maintain(context.Background())
	if evictions != 0 {
		t.Fatalf("stable maintenance ran %d full eviction scans, want none", evictions)
	}
}

func TestCoverCacheMaintainDoesNotHoldLockDuringOptimize(t *testing.T) {
	cache := newCoverCache(t.TempDir(), 0, 300, 450, 75, nil)
	legacy := cache.legacyPath("books", "slow")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, testJPEG(t, 400, 600, 9), 0o600); err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseOptimize := func() { releaseOnce.Do(func() { close(release) }) }
	cache.optimizeHook = func() {
		close(started)
		<-release
	}

	done := make(chan struct{})
	go func() {
		cache.maintain(context.Background())
		close(done)
	}()
	defer func() {
		releaseOptimize()
		<-done
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("maintenance optimization never started")
	}

	lockAcquired := make(chan struct{})
	go func() {
		cache.mu.Lock()
		cache.mu.Unlock()
		close(lockAcquired)
	}()
	defer func() {
		releaseOptimize()
		<-done
		<-lockAcquired
	}()
	select {
	case <-lockAcquired:
	case <-time.After(200 * time.Millisecond):
		releaseOptimize()
		t.Fatal("cover cache lock was held during maintenance image optimization")
	}

	releaseOptimize()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("maintenance never completed")
	}
	if _, err := os.Stat(cache.path("books", "slow")); err != nil {
		t.Fatalf("maintenance did not write the optimized cover: %v", err)
	}
}

// TestCoverCacheReadDoesNotHoldLockDuringOptimize guards against read()
// holding c.mu across the CPU-bound decode/resize/encode on a cache miss,
// which would serialize all concurrent cover generation onto one goroutine.
// write() already computes outside the lock; read() must do the same.
func TestCoverCacheReadDoesNotHoldLockDuringOptimize(t *testing.T) {
	cache := newCoverCache(t.TempDir(), 0, 300, 450, 75, nil)
	legacy := cache.legacyPath("books", "slow")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, testJPEG(t, 400, 600, 5), 0o600); err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	cache.optimizeHook = func() {
		close(started)
		<-release
	}

	done := make(chan struct{})
	go func() {
		cache.read("books", "slow")
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("optimize never started")
	}

	// The CPU-bound work is "in progress" right now. The mutex must still be
	// free to acquire; if read() held it across optimize, this would block.
	lockAcquired := make(chan struct{})
	go func() {
		cache.mu.Lock()
		cache.mu.Unlock()
		close(lockAcquired)
	}()
	select {
	case <-lockAcquired:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("cover cache lock was held during optimize; cover generation is serialized")
	}

	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("read never completed")
	}
	if cached, ok := cache.read("books", "slow"); !ok || len(cached) == 0 {
		t.Fatal("optimized cover was not cached after read")
	}
}

func testJPEG(t *testing.T, width, height int, seed uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x)%251 ^ seed, uint8(y)%241 ^ seed, uint8(x+y)%239 ^ seed, 255})
		}
	}
	var output bytes.Buffer
	if err := jpeg.Encode(&output, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestCoverCacheResizesAndEncodesJPEG(t *testing.T) {
	cache := &coverCache{maximumWidth: 600, maximumHeight: 900, jpegQuality: 82}
	optimized, ok := cache.optimize(testJPEG(t, 1200, 1800, 7))
	if !ok {
		t.Fatal("cover optimization failed")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(optimized))
	if err != nil {
		t.Fatal(err)
	}
	if format != "jpeg" || config.Width != 600 || config.Height != 900 {
		t.Fatalf("unexpected optimized cover: %s %dx%d", format, config.Width, config.Height)
	}
}

func TestCoverCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache := &coverCache{directory: t.TempDir(), maximumBytes: 1 << 30, maximumWidth: 300, maximumHeight: 450, jpegQuality: 75}
	if _, ok := cache.write("books", "old", testJPEG(t, 400, 600, 1)); !ok {
		t.Fatal("write old cover")
	}
	if _, ok := cache.write("books", "new", testJPEG(t, 400, 600, 2)); !ok {
		t.Fatal("write new cover")
	}
	oldPath, newPath := cache.path("books", "old"), cache.path("books", "new")
	oldInfo, _ := os.Stat(oldPath)
	newInfo, _ := os.Stat(newPath)
	oldTime := time.Now().Add(-time.Hour)
	_ = os.Chtimes(oldPath, oldTime, oldTime)
	cache.maximumBytes = oldInfo.Size() + newInfo.Size() - 1
	cache.mu.Lock()
	cache.evictLocked()
	cache.mu.Unlock()
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old cover was not evicted: %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new cover was evicted: %v", err)
	}
}

func TestCoverThumbnailIsSmallerAndCached(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 600, 900))
	for i := range source.Pix {
		source.Pix[i] = byte(i % 251)
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, source, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	cache := newCoverCache(t.TempDir(), 0, 600, 900, 82, nil)
	small, ok := cache.thumbnail("books", "id-w240", encoded.Bytes(), 240)
	if !ok {
		t.Fatal("thumbnail failed")
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(small))
	if err != nil || config.Width != 240 || config.Height != 360 {
		t.Fatalf("thumbnail is %dx%d (%v)", config.Width, config.Height, err)
	}
	if len(small) >= encoded.Len() {
		t.Fatalf("thumbnail %d bytes is not smaller than %d", len(small), encoded.Len())
	}
	if cached, ok := cache.read("books", "id-w240"); !ok || !bytes.Equal(cached, small) {
		t.Fatal("thumbnail must be served from the cache afterwards")
	}
	for value, want := range map[string]int{"160": 160, "240": 240, "360": 360, "": 0, "9999": 0, "abc": 0} {
		if got := coverVariantWidth(value); got != want {
			t.Fatalf("coverVariantWidth(%q) = %d, want %d", value, got, want)
		}
	}
}
