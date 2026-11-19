package web

import (
	"os"
	"path/filepath"
	"testing"

	"mochil/internal/podcast"
)

func TestAnEpisodeTheServerDownloadedIsMarkedAsSavedOnTheServer(t *testing.T) {
	root := t.TempDir()
	downloader := podcast.NewDownloader(root, nil, 1<<20)
	defer downloader.Close()
	m := &multiServer{podcastDownloads: downloader}
	if m.podcastSavedOnServer("ep1") {
		t.Fatal("nothing was downloaded yet")
	}
	if err := os.WriteFile(filepath.Join(root, "ep1.media"), []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !m.podcastSavedOnServer("ep1") || m.podcastSavedOnServer("ep2") {
		t.Fatal("only the downloaded episode is on the server")
	}
	if (&multiServer{}).podcastSavedOnServer("ep1") {
		t.Fatal("without a downloader nothing is saved on the server")
	}
}
