package net.kdhmedia.mochil.playback

import org.junit.Assert.assertEquals
import org.junit.Test

class PlaybackMediaItemTest {
    @Test fun `queue metadata supplies outgoing duration after transition`() {
        assertEquals(1_234_500L, resolvedPlaybackDurationMs(1_234.5, 0L))
    }

    @Test fun `active player duration remains the fallback`() {
        assertEquals(90_000L, resolvedPlaybackDurationMs(null, 90_000L))
        assertEquals(0L, resolvedPlaybackDurationMs(Double.NaN, -1L))
    }
}
