package podcast

import (
	"net/http"
	"testing"
)

func TestValidatePodcastURLRejectsNonPublicTargets(t *testing.T) {
	invalid := []string{
		"http://localhost/feed", "http://name.localhost/feed", "http://127.0.0.1/feed",
		"http://[::1]/feed", "http://169.254.169.254/latest/meta-data", "http://10.0.0.1/feed",
		"http://100.64.0.1/feed", "http://192.0.0.8/feed", "file:///tmp/feed",
		"https://user:password@example.com/feed", "//example.com/feed",
	}
	for _, raw := range invalid {
		if _, err := validatePodcastURL(raw, true); err == nil {
			t.Errorf("validatePodcastURL(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestValidatePodcastURLAllowsPublicTargets(t *testing.T) {
	valid := []string{"https://example.com/feed.xml", "http://8.8.8.8/feed", "https://[2606:4700:4700::1111]/feed"}
	for _, raw := range valid {
		if _, err := validatePodcastURL(raw, true); err != nil {
			t.Errorf("validatePodcastURL(%q): %v", raw, err)
		}
	}
}

func TestPublicHTTPClientRejectsPrivateRedirects(t *testing.T) {
	client := NewPublicHTTPClient(0)
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = client.CheckRedirect(request, []*http.Request{{}}); err == nil {
		t.Fatal("private redirect unexpectedly accepted")
	}
}

func TestPublicHTTPClientAllowsBoundedPublicRedirectChain(t *testing.T) {
	client := NewPublicHTTPClient(0)
	request, err := http.NewRequest(http.MethodGet, "https://media.example.com/episode.mp3", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = client.CheckRedirect(request, make([]*http.Request, 9)); err != nil {
		t.Fatalf("nine-hop public redirect chain rejected: %v", err)
	}
	if err = client.CheckRedirect(request, make([]*http.Request, 10)); err == nil {
		t.Fatal("unbounded redirect chain unexpectedly accepted")
	}
}
