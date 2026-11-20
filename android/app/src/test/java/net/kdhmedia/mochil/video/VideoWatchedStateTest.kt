package net.kdhmedia.mochil.video

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class VideoWatchedStateTest {
    private val episode = VideoItem(
        id = "episode-1", libraryId = "anime", title = "Episode 1", show = "Example",
        season = 1, episode = 1, durationSeconds = 1_440.0, artworkPath = "", streamPath = "/stream",
        hlsPath = "", resumeSeconds = 320.0, subtitles = emptyList(), dismissed = true,
    )

    @Test
    fun watchedStateUsesTheEndAndRestoresTheItemToNormalVisibility() {
        val watched = episode.withWatched(true)
        assertTrue(watched.finished)
        assertFalse(watched.dismissed)
        assertEquals(1_440.0, watched.resumeSeconds, 0.0)
        assertEquals(1, videoWatchedDelta(previous = false, watched = true))
    }

    @Test
    fun unwatchedStateClearsProgressWithoutChangingMediaIdentity() {
        val unwatched = episode.copy(finished = true).withWatched(false)
        assertFalse(unwatched.finished)
        assertEquals(0.0, unwatched.resumeSeconds, 0.0)
        assertEquals("episode-1", unwatched.id)
        assertEquals(-1, videoWatchedDelta(previous = true, watched = false))
        assertEquals(0, videoWatchedDelta(previous = false, watched = false))
    }
}
