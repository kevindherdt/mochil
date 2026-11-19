package net.kdhmedia.mochil.playback

import net.kdhmedia.mochil.audio.AudioTrack
import org.junit.Test
import org.junit.Assert.assertEquals

class PlaybackStartPositionTest {
    private fun track(kind: String) = AudioTrack(id = "1", libraryId = "l", title = "t", artist = "a", album = "b", durationSeconds = 200.0, libraryKind = kind)

    @Test
    fun aSongAlwaysStartsFromTheBeginning() {
        assertEquals(0, playbackStartPositionMs(track("music"), 95_000))
    }

    @Test
    fun audiobooksAndPodcastsContinueWhereYouStopped() {
        assertEquals(95_000, playbackStartPositionMs(track("audiobooks"), 95_000))
        assertEquals(95_000, playbackStartPositionMs(track("podcasts"), 95_000))
    }

    @Test
    fun aNegativePositionIsNeverUsed() {
        assertEquals(0, playbackStartPositionMs(track("audiobooks"), -5))
    }
}
