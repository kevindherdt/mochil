package net.kdhmedia.mochil.video

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class VideoModelsTest {
    @Test fun showWatchStateIsConciseAndBounded() {
        val active = VideoShow("Show", "Show", "one", 10, listOf(1), "", watchedCount = 3, inProgressCount = 1)
        val complete = active.copy(watchedCount = 10, inProgressCount = 0)

        assertEquals("3 of 10 played", active.watchLabel)
        assertEquals(.3f, active.watchFraction)
        assertEquals("Played", complete.watchLabel)
    }

    @Test fun appendQueryPreservesExistingArtworkParameters() {
        assertEquals("/poster?id=one&client_artwork=2", appendQuery("/poster?id=one", "client_artwork", "2"))
    }

    @Test
    fun videoQueryKeepsGroupingAndEncodesSearch() {
        val show = VideoShow("Example Show", "Example Show", "one", 3, listOf(1), "")
        assertEquals("?library=tv&limit=60&offset=0&show=Example+Show&season=1&q=finale+part", videoQuery("tv", show, 1, " finale part "))
    }

    @Test
    fun otherVideosUsesExplicitServerFilter() {
        val show = VideoShow("Other videos", "Other videos", "one", 1, emptyList(), "")
        assertEquals("?library=tv&limit=60&offset=0&other=1", videoQuery("tv", show, null, ""))
    }

    @Test fun videoPagingCarriesABoundedOffset() {
        assertEquals("?library=movies&limit=60&offset=120", videoQuery("movies", null, null, "", 120))
        assertEquals(120, queryOffset("/api/videos?library=movies&offset=120&limit=60"))
        assertEquals(0, queryOffset("/api/videos?offset=broken"))
    }

    @Test fun showAndEpisodePagingRemainIndependentInTwoPaneLayouts() {
        val show = VideoShow("Show", "Show", "one", 3, listOf(1), "")
        val root = VideoState(libraryKind = "tv", showNextOffset = 60, itemNextOffset = 120)
        assertEquals(60, root.nextOffset)
        assertEquals(120, root.copy(selectedShow = show).nextOffset)
        assertEquals(120, root.copy(libraryKind = "movies").nextOffset)
    }

    @Test
    fun playbackCapabilityAndProgressAreBounded() {
        val playable = item(stream = "/videos/one", resume = 90.0, duration = 100.0)
        assertTrue(playable.playable)
        assertEquals(.9f, playable.progressFraction)
        assertFalse(item().playable)
        assertEquals(0f, item(resume = 50.0).progressFraction)
    }

    private fun item(stream: String = "", resume: Double = 0.0, duration: Double = 0.0) = VideoItem(
        "id", "movies", "Title", "", 0, 0, duration, "", stream, "", resume, emptyList(),
    )
}
