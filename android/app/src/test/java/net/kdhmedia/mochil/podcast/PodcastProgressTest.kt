package net.kdhmedia.mochil.podcast

import net.kdhmedia.mochil.audio.AudioTrack
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PodcastProgressTest {
    private val episode = AudioTrack(
        id = "episode", libraryId = "podcasts", title = "Episode",
        artist = "Show", album = "Show", durationSeconds = 600.0,
    )

    @Test fun `latest position replaces stale catalogue position`() {
        val resolved = episode.copy(resumePositionSeconds = 30.0).withPodcastProgress(245.5, false)
        assertEquals(245.5, resolved.resumePositionSeconds, 0.001)
        assertFalse(resolved.finished)
    }

    @Test fun `remote position is bounded to episode duration`() {
        assertEquals(600.0, episode.withPodcastProgress(900.0, false).resumePositionSeconds, 0.001)
    }

    @Test fun `finished progress resolves to the end`() {
        val resolved = episode.withPodcastProgress(12.0, true)
        assertEquals(600.0, resolved.resumePositionSeconds, 0.001)
        assertTrue(resolved.finished)
    }
}
