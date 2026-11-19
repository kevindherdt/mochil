package video

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Before this fix, stopping playback in a client had no way to signal the
// server at all: the ffmpeg encoder kept running, burning CPU and disk,
// until the passive idle timeout (75s) eventually noticed nobody had
// fetched a segment in a while. Stop() lets the HTTP layer cancel the
// encoder immediately instead of waiting out that window.
func TestHLSStopCancelsRunningEncoderImmediately(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "encoder.sh")
	body := "#!/bin/sh\nexec sleep 30\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.mkv")
	if err := os.WriteFile(source, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewHLSStore(filepath.Join(root, "hls"), script, 1<<20, 2, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{LibraryID: "movies", ID: "stop-me", Revision: "one", Duration: 120, Audio: []AudioStream{{Index: 1}}}
	file, _ := os.Open(source)
	key, err := store.Start(file, item, 1, HLSQualityHigh)
	if err != nil {
		t.Fatal(err)
	}
	waitUntilRunning(t, store)

	started := time.Now()
	if !store.Stop(key) {
		t.Fatal("Stop reported nothing was running")
	}
	waitUntilCanceled(t, store)
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("Stop took %s to take effect; the 30s sleep should have been killed, not waited out", elapsed)
	}
}

// A real encoder can leave a grandchild process alive after its direct
// child is killed (a shell wrapper that forks instead of exec'ing, or a
// hardware-accel helper process): SIGKILL-ing only the direct child does
// not close file descriptors that grandchild still holds, and without
// Cmd.WaitDelay, exec.Cmd.Run() blocks until that straggler exits on its
// own — in production, that is exactly "I stopped the movie but ffmpeg
// kept going". Stop() must not hang waiting for a process tree it only
// partially killed.
func TestHLSStopDoesNotHangOnOrphanedGrandchild(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "encoder.sh")
	// No "exec": the shell forks "sleep 30" as a child instead of replacing
	// itself, so killing the shell leaves the sleep process running,
	// orphaned, still holding the shell's inherited stdout/stderr pipe ends.
	body := "#!/bin/sh\nsleep 30 &\nwait\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.mkv")
	if err := os.WriteFile(source, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewHLSStore(filepath.Join(root, "hls"), script, 1<<20, 2, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{LibraryID: "movies", ID: "orphan", Revision: "one", Duration: 120, Audio: []AudioStream{{Index: 1}}}
	file, _ := os.Open(source)
	key, err := store.Start(file, item, 1, HLSQualityHigh)
	if err != nil {
		t.Fatal(err)
	}
	waitUntilRunning(t, store)

	started := time.Now()
	if !store.Stop(key) {
		t.Fatal("Stop reported nothing was running")
	}
	waitUntilCanceled(t, store)
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("Stop took %s; it must not hang waiting on an orphaned grandchild process", elapsed)
	}
}

func TestHLSStopOnUnknownKeyIsANoop(t *testing.T) {
	root := t.TempDir()
	store, err := NewHLSStore(filepath.Join(root, "hls"), "true", 1<<20, 2, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.Stop(fmt.Sprintf("%032x", 0)) {
		t.Fatal("expected no session to stop")
	}
}

func TestHLSStopOnAlreadyFinishedSessionIsANoop(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "encoder.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.mkv")
	if err := os.WriteFile(source, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewHLSStore(filepath.Join(root, "hls"), script, 1<<20, 2, 1080, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item := Item{LibraryID: "movies", ID: "done", Revision: "one", Duration: 120, Audio: []AudioStream{{Index: 1}}}
	file, _ := os.Open(source)
	key, err := store.Start(file, item, 1, HLSQualityHigh)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range store.Jobs() {
		waitContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = store.pool.Wait(waitContext, state.Key)
		cancel()
	}
	if store.Stop(key) {
		t.Fatal("expected nothing left to stop once the encoder already exited")
	}
}

func waitUntilRunning(t *testing.T, store *HLSStore) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if states := store.Jobs(); len(states) > 0 && states[0].Status == "running" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("encoder never reached running")
}

func waitUntilCanceled(t *testing.T, store *HLSStore) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var finalStatus string
	for time.Now().Before(deadline) {
		for _, state := range store.Jobs() {
			finalStatus = state.Status
		}
		if finalStatus == "canceled" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("job status = %q, want canceled", finalStatus)
}
