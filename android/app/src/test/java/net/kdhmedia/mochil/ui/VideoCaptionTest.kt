package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoShow
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class VideoCaptionTest {
    private fun item(episode: Int = 0, resume: Double = 0.0, finished: Boolean = false, year: Int = 2019, duration: Double = 6720.0, stream: String = "/s") =
        VideoItem("i", "lib", "Film", "", 0, episode, duration, "", stream, "", resume, emptyList(), finished = finished, year = year)

    @Test
    fun anUnstartedFilmReadsYearAndLengthWithNoProgress() {
        val caption = item().caption()
        assertEquals("2019 · 1 h 52 min", caption.text)
        assertNull(caption.fraction)
    }

    @Test
    fun aStartedFilmShowsMinutesAndAProgressLine() {
        val caption = item(resume = 1680.0).caption()
        assertEquals("28 min watched", caption.text)
        assertEquals(0.25f, caption.fraction!!, 0.001f)
    }

    @Test
    fun aFinishedFilmIsPlayedAndFull() {
        val caption = item(finished = true).caption()
        assertEquals("Played", caption.text)
        assertTrue(caption.done)
        assertEquals(1f, caption.fraction!!, 0f)
    }

    @Test
    fun anEpisodeNamesItsNumber() {
        assertEquals("Episode 3 · 52 min", item(episode = 3, duration = 3120.0).caption().text)
    }

    @Test
    fun showsCountEpisodesUntilStarted() {
        val show = VideoShow("Mouse", "mouse", "m", 20, listOf(1), "")
        assertEquals("20 episodes", show.caption().text)
        assertNull(show.caption().fraction)
        assertEquals("5 of 20 played", show.copy(watchedCount = 5).caption().text)
    }
}
