package video

import "testing"

func TestInferEpisodeFromShowFolderAndFilename(t *testing.T) {
	item := Item{Path: "Baki Hanma (2021) [tvdbid=406152]/Season 2/Baki Hanma - S02E07 - Moment of the Heat WEBDL-1080p.mkv", Title: "Baki Hanma - S02E07 - Moment of the Heat WEBDL-1080p"}
	InferEpisode(&item)
	if item.Show != "Baki Hanma (2021)" || item.Season != 2 || item.Episode != 7 || item.EpisodeTitle != "Moment of the Heat" {
		t.Fatalf("episode inference = %#v", item)
	}
}

func TestInferEpisodeDoesNotGuessFromMovieNumbers(t *testing.T) {
	item := Item{Path: "Movies/2001 A Space Odyssey 2160p.mkv", Title: "2001 A Space Odyssey 2160p"}
	InferEpisode(&item)
	if item.Show != "" || item.Season != 0 || item.Episode != 0 || item.EpisodeTitle != "" {
		t.Fatalf("movie was treated as episode: %#v", item)
	}
}

func TestInferEpisodeFromFansubNameInsideSeasonFolder(t *testing.T) {
	item := Item{Path: "When They Cry - Higurashi (2006) [tvdbid=79682]/Season 02/[Okay-Subs] Higurashi no Naku Koro ni S2 - 01 (BD 576p) [1FA38EDE].mkv", Title: "[Okay-Subs] Higurashi no Naku Koro ni S2 - 01 (BD 576p) [1FA38EDE]"}
	InferEpisode(&item)
	if item.Show != "When They Cry - Higurashi (2006)" || item.Season != 2 || item.Episode != 1 {
		t.Fatalf("show=%q season=%d episode=%d", item.Show, item.Season, item.Episode)
	}
	loose := Item{Path: "Movies/Blade Runner - 2049.mkv", Title: "Blade Runner - 2049"}
	InferEpisode(&loose)
	if loose.Show != "" || loose.Episode != 0 {
		t.Fatalf("a numbered title outside a season folder became an episode: %+v", loose)
	}
}
