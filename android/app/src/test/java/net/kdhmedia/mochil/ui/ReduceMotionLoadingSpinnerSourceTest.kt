package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * A prior audit found ~12 call sites that construct `CircularProgressIndicator()` directly,
 * bypassing the shared `AccessibleLoadingIndicator` helper, and still spin unconditionally even
 * when e-ink's "Reduce animations" setting (`LocalReduceMotion`) is on. Each of these sites must
 * gate on `LocalReduceMotion.current` and render a static full ring (`progress = { 1f }`) instead
 * of the always-spinning indeterminate `CircularProgressIndicator()`, the same pattern
 * `AccessibleLoadingIndicator.kt` itself uses (see `AccessibleLoadingIndicatorMotionEfficiencySourceTest`).
 *
 * This is a single source-scan test covering every touched file, matching that existing
 * convention, since swapping these call sites to the shared composable would risk changing their
 * exact size/color/modifier (the shared helper defaults to a 32dp ring, which does not
 * necessarily match these call sites' existing sizing) — so each site keeps its own
 * `CircularProgressIndicator` call, just gated inline.
 */
class ReduceMotionLoadingSpinnerSourceTest {

    private fun source(relativePath: String): String =
        Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/$relativePath"))

    private fun assertSiteIsGated(relativePath: String, expectedCount: Int) {
        val text = source(relativePath)
        assertTrue(
            "$relativePath should import LocalReduceMotion",
            text.contains("import net.kdhmedia.mochil.ui.theme.LocalReduceMotion"),
        )
        val gatedCount = Regex("LocalReduceMotion\\.current").findAll(text).count()
        assertTrue(
            "$relativePath should gate $expectedCount loading-spinner site(s) on LocalReduceMotion.current, found $gatedCount",
            gatedCount >= expectedCount,
        )
        val staticRingCount = Regex("progress = \\{ ?1f ?\\}").findAll(text).count()
        assertTrue(
            "$relativePath should render a static full ring (progress = { 1f }) for $expectedCount site(s), found $staticRingCount",
            staticRingCount >= expectedCount,
        )
    }

    @Test fun pdfReaderScreenPageLoadingSpinnerIsGated() =
        assertSiteIsGated("reader/pdf/PdfReaderScreen.kt", 1)

    @Test fun comicReaderScreenPageLoadingSpinnerIsGated() =
        assertSiteIsGated("reader/comic/ComicReaderScreen.kt", 1)

    @Test fun hermesChatScreenLoadingSpinnersAreGated() =
        assertSiteIsGated("hermes/HermesChatScreen.kt", 2)

    @Test fun collectionPickerSheetLoadingSpinnersAreGated() =
        assertSiteIsGated("ui/CollectionPickerSheet.kt", 2)

    @Test fun wikiScreenLoadingSpinnersAreGated() =
        assertSiteIsGated("ui/WikiScreen.kt", 2)

    @Test fun musicGroupBrowserBusySpinnerIsGated() =
        assertSiteIsGated("ui/MusicGroupBrowser.kt", 1)

    @Test fun musicExpandableGroupListLoadingSpinnersAreGated() =
        assertSiteIsGated("ui/MusicExpandableGroupList.kt", 2)

    @Test fun musicPlaylistLibrarySpinnersAreGated() =
        assertSiteIsGated("ui/MusicPlaylistLibrary.kt", 3)

    @Test fun opdsScreenLoadingSpinnersAreGated() =
        assertSiteIsGated("ui/OpdsScreen.kt", 2)

    @Test fun webDavScreenLoadingSpinnersAreGated() =
        assertSiteIsGated("ui/WebDavScreen.kt", 2)

    @Test fun localMediaLibraryScanningSpinnerIsGated() =
        assertSiteIsGated("ui/LocalMediaLibrary.kt", 1)

    @Test fun authorBrowseLoadingSpinnerIsGated() =
        assertSiteIsGated("ui/AuthorBrowse.kt", 1)

    @Test fun publicationPagingIndicatorIsGated() =
        assertSiteIsGated("ui/PublicationItems.kt", 1)

    @Test fun podcastSubscribeSearchSpinnerIsGated() =
        assertSiteIsGated("ui/PodcastSubscribeDialog.kt", 1)

    @Test fun podcastEpisodeActionSpinnersAreGated() =
        assertSiteIsGated("ui/PodcastEpisodes.kt", 3)

    @Test fun musicPlaylistLibraryActionSpinnersAreGated() =
        assertSiteIsGated("ui/MusicPlaylistLibrary.kt", 5)

    @Test fun musicPlaylistsPanelSpinnersAreGated() =
        assertSiteIsGated("ui/MusicPlaylistsPanel.kt", 3)

    @Test fun musicTrackDownloadSpinnerIsGated() =
        assertSiteIsGated("ui/MusicTrackRows.kt", 1)

    @Test fun musicQueueSavingSpinnerIsGated() =
        assertSiteIsGated("ui/MusicQueuePanel.kt", 1)

    @Test fun loginSpinnerIsGated() =
        assertSiteIsGated("ui/LoginScreen.kt", 1)
}
