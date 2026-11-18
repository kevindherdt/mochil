package net.kdhmedia.mochil.podcast

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PodcastPlaybackResolutionTest {
    @Test fun onlyNewestRequestMayStartPlayback() {
        val resolution = PodcastPlaybackResolution()
        val first = resolution.begin()
        val second = resolution.begin()
        assertFalse(resolution.isCurrent(first))
        assertTrue(resolution.isCurrent(second))
    }

    @Test fun navigationCancelsPendingPlayback() {
        val resolution = PodcastPlaybackResolution()
        val pending = resolution.begin()
        resolution.cancel()
        assertFalse(resolution.isCurrent(pending))
    }
}
