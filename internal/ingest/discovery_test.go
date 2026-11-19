package ingest

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDiscoverNamedReusesTheWalkAcrossConsecutivePages proves (and, once fixed,
// guards against) the bug where every pagination call re-walked the whole
// source tree from scratch just to skip past entries already returned.
func TestDiscoverNamedReusesTheWalkAcrossConsecutivePages(t *testing.T) {
	executor, incoming, _, _ := newTestExecutor(t, 1<<20)
	for i := 0; i < 40; i++ {
		name := fmt.Sprintf("book-%02d.epub", i)
		if err := os.WriteFile(filepath.Join(incoming, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var visits int
	executor.discoveryVisitHook = func() { visits++ }

	first, err := executor.Discover(context.Background(), Publication, "", 5)
	if err != nil || len(first.Items) != 5 || first.NextCursor == "" {
		t.Fatalf("first page = %#v, %v", first, err)
	}
	if visits == 0 {
		t.Fatal("expected the first page to walk the source tree at least once")
	}

	visits = 0
	second, err := executor.Discover(context.Background(), Publication, first.NextCursor, 5)
	if err != nil || len(second.Items) != 5 {
		t.Fatalf("second page = %#v, %v", second, err)
	}
	if visits != 0 {
		t.Fatalf("second page re-walked the source tree instead of reusing the cached listing: %d directory-entry visits", visits)
	}
}

// TestDiscoverNamedCacheRefreshesAfterItsTTL proves the cached listing is not
// stale forever: it still picks up files added after the cache window elapses,
// while serving consecutive pages inside the window from the same snapshot.
func TestDiscoverNamedCacheRefreshesAfterItsTTL(t *testing.T) {
	executor, incoming, _, _ := newTestExecutor(t, 1<<20)
	if err := os.WriteFile(filepath.Join(incoming, "a.epub"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixed := time.Now()
	executor.now = func() time.Time { return fixed }

	first, err := executor.Discover(context.Background(), Publication, "", 10)
	if err != nil || len(first.Items) != 1 {
		t.Fatalf("first page = %#v, %v", first, err)
	}

	if err := os.WriteFile(filepath.Join(incoming, "b.epub"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	stillWarm, err := executor.Discover(context.Background(), Publication, "", 10)
	if err != nil || len(stillWarm.Items) != 1 {
		t.Fatalf("expected the warm cache to still serve the old snapshot: %#v %v", stillWarm, err)
	}

	executor.now = func() time.Time { return fixed.Add(discoveryCacheTTL + time.Second) }
	refreshed, err := executor.Discover(context.Background(), Publication, "", 10)
	if err != nil || len(refreshed.Items) != 2 {
		t.Fatalf("expected the cache to refresh past its TTL and see the new file: %#v %v", refreshed, err)
	}
}

func TestDiscoveryIsBoundedStableAndProposesStructuredTargets(t *testing.T) {
	executor, incoming, _, _ := newTestExecutor(t, 1<<20)
	files := map[string]string{
		"Brandon Sanderson/Mistborn/Mistborn - Vol 01 - The Final Empire.epub": "book",
		"Daft Punk/Discovery/03 - Digital Love.flac":                           "music",
		"The Expanse/Season 02/The.Expanse.S02E03.Static.mkv":                  "video",
		"ignore.txt": "ignored",
	}
	for relative, body := range files {
		path := filepath.Join(incoming, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	first, err := executor.Discover(context.Background(), "", "", 2)
	if err != nil || len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("first page = %#v, %v", first, err)
	}
	second, err := executor.Discover(context.Background(), "", first.NextCursor, 2)
	if err != nil || len(second.Items) != 1 || second.NextCursor != "" {
		t.Fatalf("second page = %#v, %v", second, err)
	}
	all := append(first.Items, second.Items...)
	want := map[string]string{
		"publication": "Brandon Sanderson/Mistborn/Mistborn - Vol 1 - The Final Empire.epub",
		"music":       "Daft Punk/Discovery/03 - Digital Love.flac",
		"video":       "The Expanse/Season 02/The Expanse - S02E03 - Static.mkv",
	}
	for _, item := range all {
		if item.SuggestedTarget != want[string(item.Kind)] || len(item.Proposal.Evidence) == 0 {
			t.Fatalf("unexpected discovery item: %#v", item)
		}
	}
}

func TestDiscoveryFiltersKindAndNeverFollowsSymlinks(t *testing.T) {
	executor, incoming, _, _ := newTestExecutor(t, 1<<20)
	outside := filepath.Join(t.TempDir(), "outside.epub")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(incoming, "linked.epub")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(incoming, "song.mp3"), []byte("song"), 0o600); err != nil {
		t.Fatal(err)
	}
	page, err := executor.Discover(context.Background(), Music, "", 10)
	if err != nil || len(page.Items) != 1 || page.Items[0].SourcePath != "song.mp3" {
		t.Fatalf("filtered discovery = %#v, %v", page, err)
	}
	if _, err = executor.Discover(context.Background(), Kind("archive"), "", 10); err == nil {
		t.Fatal("expected unsupported kind error")
	}
	if _, err = executor.Discover(context.Background(), "", "../escape", 10); err == nil {
		t.Fatal("expected unsafe cursor error")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = executor.Discover(cancelled, "", "", 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled discovery = %v", err)
	}
}

func TestDiscoveryReadsTheBooksOwnMetadataWhenItsNameSaysNothing(t *testing.T) {
	executor, incoming, _, _ := newTestExecutor(t, 1<<20)
	file := filepath.Join(incoming, "misc", "download (7).epub")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	out, _ := os.Create(file)
	zw := zip.NewWriter(out)
	w, _ := zw.Create("META-INF/container.xml")
	w.Write([]byte(`<container><rootfiles><rootfile full-path="content.opf"/></rootfiles></container>`))
	w, _ = zw.Create("content.opf")
	w.Write([]byte(calibreOPF))
	zw.Close()
	out.Close()
	// a broken book next to it keeps the guess from its name
	if err := os.WriteFile(filepath.Join(incoming, "misc", "broken.epub"), []byte("not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	page, err := executor.Discover(context.Background(), Publication, "", 10)
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("%#v %v", page, err)
	}
	for _, item := range page.Items {
		switch item.SourcePath {
		case "misc/download (7).epub":
			if item.Proposal.Series != "Mushoku Tensei" || item.Proposal.Authors[0] != "Rifujin na Magonote" {
				t.Fatalf("embedded metadata was not used: %+v", item.Proposal)
			}
		case "misc/broken.epub":
			if item.Proposal.Title != "broken" {
				t.Fatalf("broken book lost its guess: %+v", item.Proposal)
			}
		}
	}
}

func TestDiscoveryFollowsTheChosenNamingPreset(t *testing.T) {
	executor, incoming, _, _ := newTestExecutor(t, 1<<20)
	path := filepath.Join(incoming, "Rifujin", "Mushoku Tensei", "Mushoku Tensei - Vol 3 - Title.epub")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	kavita, err := executor.DiscoverNamed(context.Background(), Publication, "", 10, NamingKavita)
	if err != nil || len(kavita.Items) != 1 || kavita.Items[0].SuggestedTarget != "Mushoku Tensei/Mushoku Tensei - Vol 03.epub" {
		t.Fatalf("%#v %v", kavita, err)
	}
	plain, _ := executor.Discover(context.Background(), Publication, "", 10)
	if plain.Items[0].SuggestedTarget != "Rifujin/Mushoku Tensei/Mushoku Tensei - Vol 3 - Title.epub" {
		t.Fatalf("the default layout changed: %q", plain.Items[0].SuggestedTarget)
	}
	if _, err := executor.DiscoverNamed(context.Background(), Publication, "", 10, "../etc"); err == nil {
		t.Fatal("an unknown preset was accepted")
	}
}

func TestADefaultNameDoesNotRepeatTheSeriesAndVolumeInTheTitle(t *testing.T) {
	index := 1.0
	for title, want := range map[string]string{
		"7th Time Loop - Volume 01":                  "A/7th Time Loop/7th Time Loop - Vol 1.epub",
		"7th Time Loop, Vol. 1":                      "A/7th Time Loop/7th Time Loop - Vol 1.epub",
		"7th Time Loop - Volume 01 - The Villainess": "A/7th Time Loop/7th Time Loop - Vol 1 - The Villainess.epub",
		"The Final Empire":                           "A/7th Time Loop/7th Time Loop - Vol 1 - The Final Empire.epub",
	} {
		got := suggestedTarget("x.epub", Proposal{Kind: Publication, Title: title, Authors: []string{"A"}, Series: "7th Time Loop", SeriesIndex: &index})
		if got != want {
			t.Errorf("%q -> %q, want %q", title, got, want)
		}
	}
}

func TestABatchFolderHoldingManySeriesIsNotAnAuthor(t *testing.T) {
	executor, incoming, _, _ := newTestExecutor(t, 1<<20)
	write := func(relative string) {
		path := filepath.Join(incoming, filepath.FromSlash(relative))
		os.MkdirAll(filepath.Dir(path), 0o700)
		os.WriteFile(path, []byte("x"), 0o600)
	}
	for i := 0; i < 20; i++ {
		write("batch25/Series " + string(rune('A'+i)) + "/Series " + string(rune('A'+i)) + " - Volume 01.pdf")
	}
	write("Brandon Sanderson/Mistborn/Mistborn - Volume 01.pdf")
	page, err := executor.Discover(context.Background(), Publication, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range page.Items {
		switch {
		case strings.HasPrefix(item.SourcePath, "batch25/"):
			if len(item.Proposal.Authors) != 0 || item.Proposal.Series == "" {
				t.Fatalf("batch folder taken as an author: %+v", item.Proposal)
			}
		case strings.HasPrefix(item.SourcePath, "Brandon Sanderson/"):
			if len(item.Proposal.Authors) != 1 || item.Proposal.Authors[0] != "Brandon Sanderson" {
				t.Fatalf("a real author folder lost its author: %+v", item.Proposal)
			}
		}
	}
}

func TestAReleaseTagInAFilenameIsNotPartOfTheTitle(t *testing.T) {
	got := InferPath(Publication, "A/Battle Royale/Battle Royale - Complete [Nathan Collins, VIZ].epub")
	if got.Title != "Complete" {
		t.Fatalf("%q", got.Title)
	}
	if got := InferPath(Publication, "A/Spica/[Oshi No Ko] Spica the First Star.epub"); got.Title != "[Oshi No Ko] Spica the First Star" {
		t.Fatalf("a leading bracket belongs to the title: %q", got.Title)
	}
}

// TestDiscoverPaginationNeverDropsASiblingThatSortsBeforeASubdirectory guards
// the cursor contract: DiscoverNamed resumes with a lexicographic comparison on
// the slash-joined relative path, but filepath.WalkDir visits a directory's
// contents before a sibling file whose name is that directory plus a byte below
// '/' (space, '!', '(' ...). Walk order and string order disagree there, so a
// cursor taken from "A/x.mp3" used to skip "A!.mp3" on every later page.
func TestDiscoverPaginationNeverDropsASiblingThatSortsBeforeASubdirectory(t *testing.T) {
	executor, incoming, _, _ := newTestExecutor(t, 1<<20)
	if err := os.MkdirAll(filepath.Join(incoming, "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []string{"A!.mp3", "A (live).mp3", "A/x.mp3", "A/y.mp3", "B.mp3"}
	for _, name := range want {
		if err := os.WriteFile(filepath.Join(incoming, filepath.FromSlash(name)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, limit := range []int{1, 2, 3} {
		seen := map[string]int{}
		after := ""
		for page := 0; page < 20; page++ {
			result, err := executor.Discover(context.Background(), "", after, limit)
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range result.Items {
				seen[item.SourcePath]++
			}
			if result.NextCursor == "" {
				break
			}
			after = result.NextCursor
		}
		for _, name := range want {
			if seen[name] != 1 {
				t.Fatalf("limit %d: %q returned %d times across pages, want exactly once (seen %v)", limit, name, seen[name], seen)
			}
		}
	}
}
