package catalog

import "testing"

func TestNumberedFilenameInference(t *testing.T) {
	books := []Book{
		{Path: "Mistborn/01 - The Final Empire.epub", Title: "01 - The Final Empire", Authors: []string{"Unknown"}},
		{Path: "Mistborn/02 - The Well of Ascension.epub", Title: "02 - The Well of Ascension", Authors: []string{"Unknown"}},
	}
	inferFilesystemMetadata(books, FilesystemOptions{FolderLayout: "auto"})
	if books[0].Series != "Mistborn" || books[0].SeriesIndex == nil || *books[0].SeriesIndex != 1 {
		t.Fatalf("unexpected inference: %+v", books[0])
	}
	if books[0].Title != "The Final Empire" {
		t.Fatalf("unexpected inferred title %q", books[0].Title)
	}
}

func TestBracketedMangaChapterInference(t *testing.T) {
	books := []Book{
		{Path: "Battle_Royale/[0001]_Chapter_1.cbz", Title: "[0001]_Chapter_1"},
		{Path: "Battle_Royale/[0010]_Chapter_10.cbz", Title: "[0010]_Chapter_10"},
	}
	inferFilesystemMetadata(books, FilesystemOptions{FolderLayout: "auto"})
	if books[0].Series != "Battle Royale" || books[0].Title != "Chapter 1" || books[0].SeriesIndex == nil || *books[0].SeriesIndex != 1 {
		t.Fatalf("unexpected first chapter inference: %+v", books[0])
	}
	if books[1].Title != "Chapter 10" || books[1].SeriesIndex == nil || *books[1].SeriesIndex != 10 {
		t.Fatalf("unexpected tenth chapter inference: %+v", books[1])
	}
}

func TestAutoDoesNotTreatSingleNumberedFileAsSeries(t *testing.T) {
	books := []Book{{Path: "Author/01 - Only Book.epub", Title: "01 - Only Book", Authors: []string{"Author"}}}
	inferFilesystemMetadata(books, FilesystemOptions{FolderLayout: "auto"})
	if books[0].Series != "" {
		t.Fatalf("unexpected series %q", books[0].Series)
	}
}

func TestSeriesLayoutUsesFolderWithoutNumberedFilenames(t *testing.T) {
	books := []Book{{Path: "Mistborn/The Final Empire.epub", Title: "The Final Empire", Authors: []string{"Unknown"}}}
	inferFilesystemMetadata(books, FilesystemOptions{FolderLayout: "series"})
	if books[0].Series != "Mistborn" || books[0].SeriesIndex != nil {
		t.Fatalf("unexpected folder inference: %+v", books[0])
	}
}

func TestSeriesLayoutInfersMarkedVolume(t *testing.T) {
	books := []Book{{Path: "Mushoku Tensei/Mushoku Tensei Volume 10.epub", Title: "Mushoku Tensei Volume 10"}}
	inferFilesystemMetadata(books, FilesystemOptions{FolderLayout: "series"})
	if books[0].SeriesIndex == nil || *books[0].SeriesIndex != 10 {
		t.Fatalf("unexpected inference: %+v", books[0])
	}
}

func TestSeriesLayoutPrefersVolumeOverTitleNumber(t *testing.T) {
	books := []Book{{Path: "86--EIGHTY-SIX/86 - Volume 10.epub", Title: "86—EIGHTY-SIX, Vol. 10"}}
	inferFilesystemMetadata(books, FilesystemOptions{FolderLayout: "series"})
	if books[0].SeriesIndex == nil || *books[0].SeriesIndex != 10 {
		t.Fatalf("title number won over volume number: %+v", books[0])
	}
}

func TestSeriesTitleBeginningWithNumberIsNotVolumePrefix(t *testing.T) {
	books := []Book{{Path: "5 Centimeters per Second/5 Centimeters per Second + Children Who Chase Lost Voices.pdf", Title: "5 Centimeters per Second + Children Who Chase Lost Voices", Authors: []string{"Writer"}}}
	inferFilesystemMetadata(books, FilesystemOptions{FolderLayout: "series"})
	if books[0].Series != "5 Centimeters per Second" || books[0].SeriesIndex != nil {
		t.Fatalf("title number was mistaken for volume metadata: %+v", books[0])
	}
}

func TestBareNumberRequiresDelimiter(t *testing.T) {
	if inferred := parseNumberedFilename("5 Centimeters per Second"); inferred.ok {
		t.Fatalf("plain title number was parsed as a volume: %+v", inferred)
	}
	if inferred := parseNumberedFilename("05 - The Fifth Book"); !inferred.ok || inferred.index != 5 {
		t.Fatalf("delimited volume was not parsed: %+v", inferred)
	}
}

func TestSpecialPublicationsKeepSeriesWithoutInventedOrder(t *testing.T) {
	books := []Book{
		{Path: "Saga/01 - Side Stories.epub", Title: "01 - Side Stories"},
		{Path: "Saga/02 - Omnibus Edition.epub", Title: "02 - Omnibus Edition"},
		{Path: "Saga/Volume 03.epub", Title: "Volume 03"},
	}
	inferFilesystemMetadata(books, FilesystemOptions{FolderLayout: "series"})
	if books[0].Series != "Saga" || books[0].SeriesIndex != nil || books[0].PublicationRole != PublicationSideStory {
		t.Fatalf("side story inference = %+v", books[0])
	}
	if books[1].SeriesIndex != nil || books[1].PublicationRole != PublicationOmnibus {
		t.Fatalf("omnibus inference = %+v", books[1])
	}
	if books[2].SeriesIndex == nil || *books[2].SeriesIndex != 3 || books[2].PublicationRole != "" {
		t.Fatalf("ordinary volume inference = %+v", books[2])
	}
}

// Sandbox layout: every book sits in its own "Author/Title (id)/" folder. The folder is storage, not a series.
func TestPerBookFolderIsNotASeries(t *testing.T) {
	books := []Book{
		{Path: "George Orwell/1984 (282)/1984 - George Orwell.epub", Title: "1984", Authors: []string{"George Orwell"}},
		{Path: "S. M. Stirling/01 Dies the Fire (318)/01 Dies the Fire - S. M. Stirling.epub", Title: "01 Dies the Fire", Authors: []string{"S. M. Stirling"}},
	}
	inferFilesystemMetadata(books, FilesystemOptions{FolderLayout: "series"})
	if books[0].Series != "" || books[0].SeriesIndex != nil {
		t.Fatalf("a per-book folder became a series: %+v", books[0])
	}
	if books[1].Series != "" {
		t.Fatalf("a per-book folder became a series: %+v", books[1])
	}
	if books[1].SeriesIndex != nil {
		t.Fatalf("a per-book folder should not invent a position: %+v", books[1])
	}
}

// The folder still names the series when the folder says something the book does not.
func TestFolderThatIsNotTheTitleStillNamesTheSeries(t *testing.T) {
	books := []Book{{Path: "Mistborn/The Final Empire.epub", Title: "The Final Empire", Authors: []string{"Unknown"}}}
	inferFilesystemMetadata(books, FilesystemOptions{FolderLayout: "series"})
	if books[0].Series != "Mistborn" {
		t.Fatalf("expected the folder to name the series, got %+v", books[0])
	}
}
