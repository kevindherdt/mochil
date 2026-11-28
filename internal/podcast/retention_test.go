package podcast

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A feed's download cap bounds downloads, not episodes: a hand-downloaded episode from long ago is the only file on disk and
// must survive, even though it is far outside the newest few episodes of the feed.
func TestRetainDownloadsKeepsAHandDownloadedOldEpisode(t *testing.T) {
	root := t.TempDir()
	downloader := NewDownloader(root, nil, 0)
	defer downloader.Close()

	feed := Feed{ID: "feed", Policy: FeedPolicy{RetainDownloads: 1}}
	now := time.Now()
	for index := range 5 {
		id := "ep-" + string(rune('a'+index))
		feed.Episodes = append(feed.Episodes, Episode{ID: id, Published: now.AddDate(0, 0, -index*100)})
	}
	// Only the oldest episode is on disk.
	oldest := feed.Episodes[len(feed.Episodes)-1]
	if err := os.WriteFile(filepath.Join(root, oldest.ID+".media"), []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	downloader.Reconcile(feed, nil)
	if _, ok := downloader.Path(oldest.ID); !ok {
		t.Fatalf("the only download on disk was deleted because it is not in the newest %d episodes", feed.Policy.RetainDownloads)
	}
}

// And the cap still does its job once there are more downloads than it allows.
func TestRetainDownloadsTrimsToTheNewestOnes(t *testing.T) {
	root := t.TempDir()
	downloader := NewDownloader(root, nil, 0)
	defer downloader.Close()

	feed := Feed{ID: "feed", Policy: FeedPolicy{RetainDownloads: 2}}
	now := time.Now()
	for index := range 4 {
		id := "ep-" + string(rune('a'+index))
		feed.Episodes = append(feed.Episodes, Episode{ID: id, Published: now.AddDate(0, 0, -index)})
		if err := os.WriteFile(filepath.Join(root, id+".media"), []byte("audio"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	downloader.Reconcile(feed, nil)
	for index, episode := range feed.Episodes {
		_, ok := downloader.Path(episode.ID)
		if want := index < 2; ok != want {
			t.Fatalf("episode %d (%s) present = %v, want %v", index, episode.ID, ok, want)
		}
	}
}
