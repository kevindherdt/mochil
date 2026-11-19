package transcode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestEngineDistantSeekWaitsForCanceledEncoder(t *testing.T) {
	flock, err := exec.LookPath("flock")
	if err != nil {
		t.Skip("flock is unavailable")
	}
	root := t.TempDir()
	script, lock, overlap := filepath.Join(root, "encoder.sh"), filepath.Join(root, "encoder.lock"), filepath.Join(root, "overlap")
	body := fmt.Sprintf("#!/bin/sh\nexec 9>%q\nif ! %q -n 9; then echo overlap >>%q; exit 1; fi\nsleep 0.4\nexit 1\n", lock, flock, overlap)
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.flac")
	if err := os.WriteFile(source, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	engine, err := NewEngine(filepath.Join(root, "hls"), script, 1<<20, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	engine.readyAfter = 2 * time.Second
	request := Request{Source: Source{LibraryID: "music", ItemID: "track", Revision: "one", Kind: "audio"}, Variant: "aac", Arguments: func(int, string) []string { return nil }}
	file, _ := os.Open(source)
	if _, err := engine.Start(file, request); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		states := engine.Jobs()
		if len(states) > 0 && states[len(states)-1].Status == "running" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	file, _ = os.Open(source)
	if err := engine.Ensure(file, request, 20); err != nil {
		t.Fatal(err)
	}
	for _, state := range engine.Jobs() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = engine.pool.Wait(ctx, state.Key)
		cancel()
	}
	if data, err := os.ReadFile(overlap); err == nil {
		t.Fatalf("encoders overlapped: %s", data)
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestEngineOpenAllowsOnlySessionMedia(t *testing.T) {
	root := t.TempDir()
	engine, err := NewEngine(root, "true", 1<<20, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	request := Request{Source: Source{LibraryID: "books", ItemID: "one", Revision: "r1", Kind: "audio"}, Variant: "aac", Arguments: func(int, string) []string { return nil }}
	key := engine.Key(request)
	directory := filepath.Join(root, key)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	engine.sessions[key] = &session{key: key, directory: directory, lastUsed: time.Now()}
	if err := os.WriteFile(filepath.Join(directory, "init.mp4"), []byte("init"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, mediaType, err := engine.Open(key, "init.mp4")
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	if mediaType != "audio/mp4" {
		t.Fatalf("media type = %q", mediaType)
	}
	for _, name := range []string{"../secret", "index.m3u8", "seg1.m4s", "seg00000.ts"} {
		if file, _, err := engine.Open(key, name); err == nil {
			file.Close()
			t.Fatalf("opened %q", name)
		}
	}
}

func TestEngineUsesValidatedDescriptorAfterPathReplacement(t *testing.T) {
	root := t.TempDir()
	captured := filepath.Join(root, "captured")
	script := filepath.Join(root, "encoder.sh")
	body := fmt.Sprintf("#!/bin/sh\nsleep 0.1\ncat <&3 >%q\n", captured)
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source.flac")
	if err := os.WriteFile(source, []byte("validated"), 0o600); err != nil {
		t.Fatal(err)
	}
	engine, err := NewEngine(filepath.Join(root, "hls"), script, 1<<20, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	request := Request{Source: Source{LibraryID: "music", ItemID: "track", Revision: "one", Kind: "audio"}, Variant: "aac", Arguments: func(int, string) []string { return nil }}
	file, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Start(file, request); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, source+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	states := engine.Jobs()
	if len(states) != 1 {
		t.Fatalf("jobs = %#v", states)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	err = engine.pool.Wait(ctx, states[0].Key)
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(captured)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "validated" {
		t.Fatalf("captured = %q", data)
	}
}
