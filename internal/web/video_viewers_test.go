package web

import (
	"testing"
	"time"
)

func TestStoppingAnEncodeLeavesAnotherViewersStreamAlone(t *testing.T) {
	var viewers videoViewers
	const key = "video/episode1/0/720p"
	viewers.observe(key, "alice")
	if viewers.remembered(key, "alice") {
		t.Fatal("a stop from the only viewer was refused, so closing the player would never stop the encoder")
	}
	if !viewers.remembered(key, "bob") {
		t.Fatal("bob stopping cancelled the encode alice is watching")
	}
}

// The viewer that closes has just been fetching segments, so relying on the most
// recent observer alone honoured its stop even while another account was watching.
func TestTheViewerThatClosesLastStillSeesTheOtherViewer(t *testing.T) {
	var viewers videoViewers
	const key = "video/episode1/0/720p"
	viewers.observe(key, "bob")
	viewers.observe(key, "alice") // alice fetched most recently, and now closes
	if !viewers.remembered(key, "alice") {
		t.Fatal("alice closing cancelled the encode bob is still watching")
	}
}

func TestAStaleViewerDoesNotHoldTheEncoderOpen(t *testing.T) {
	var viewers videoViewers
	const key = "video/episode1/0/720p"
	viewers.observe(key, "alice")
	viewers.mu.Lock()
	viewers.last[key]["alice"] = time.Now().Add(-2 * videoViewerWindow)
	viewers.mu.Unlock()
	if viewers.remembered(key, "bob") {
		t.Fatal("a viewer from long ago kept the encoder alive")
	}
}
