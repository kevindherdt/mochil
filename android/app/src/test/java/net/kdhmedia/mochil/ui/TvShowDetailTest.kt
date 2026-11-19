package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoShow
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class TvShowDetailTest {
    private fun episode(n: Int, finished: Boolean = false, resume: Double = 0.0, duration: Double = 2_520.0) = VideoItem(
        id = "e$n", libraryId = "tv", title = "T", show = "S", season = 1, episode = n, durationSeconds = duration,
        artworkPath = "", streamPath = "", hlsPath = "", resumeSeconds = resume, subtitles = emptyList(), finished = finished,
    )

    @Test fun playButtonResumesTheEpisodeInProgress() {
        val items = listOf(episode(1, finished = true), episode(2, resume = 600.0), episode(3))
        assertEquals("e2", tvShowPlayTarget(items)?.id)
    }

    @Test fun playButtonMovesOnToTheNextUnwatchedEpisode() {
        assertEquals("e3", tvShowPlayTarget(listOf(episode(1, finished = true), episode(2, finished = true), episode(3)))?.id)
    }

    @Test fun playButtonStartsOverWhenEverythingIsWatched() {
        assertEquals("e1", tvShowPlayTarget(listOf(episode(1, finished = true), episode(2, finished = true)))?.id)
        assertNull(tvShowPlayTarget(emptyList()))
    }

    @Test fun showAndEpisodeLinesAreCompact() {
        val show = VideoShow("S", "S", "i", 10, listOf(1), "", year = 2020)
        assertEquals("2020 · 1 season · 10 episodes", tvShowMeta(show))
        assertEquals("S1 · E4 · 42 min", tvEpisodeLabel(episode(4)))
        assertEquals("2020 · 1 h 52 min", tvMovieMeta(episode(1, duration = 6_720.0).copy(year = 2020)))
    }

    /**
     * Season 0 is the extras folder on the server. The phone has always called it Specials; the TV row used to print
     * "Season 0", so the same show read as two different things on the two screens.
     */
    @Test fun seasonsCarryTheSameNamesAsThePhone() {
        assertEquals("Specials", tvSeasonLabel(0))
        assertEquals("Season 1", tvSeasonLabel(1))
        assertEquals("Season 20", tvSeasonLabel(20))
    }
}
