package net.kdhmedia.mochil.playback

import net.kdhmedia.mochil.audio.audioTrackIdentityKey
import org.junit.Assert.*
import org.junit.Test

class PlaybackQueueIdentityTest {
    @Test fun `captured item follows reordering and never falls back to an old row`() {
        val queue = mutableListOf("one", "two", "three")
        assertEquals(1, playbackQueueIndex(queue.size, "two", queue::get))
        queue.removeAt(0)
        assertEquals(0, playbackQueueIndex(queue.size, "two", queue::get))
        queue.removeAt(0)
        assertEquals(-1, playbackQueueIndex(queue.size, "two", queue::get))
        assertEquals(-1, playbackQueueIndex(0, "two") { error("Empty queue lookup") })
    }

    @Test fun `same item id in different libraries remains distinct`() {
        val first = audioTrackIdentityKey("first", "track")
        val second = audioTrackIdentityKey("second", "track")
        val queue = listOf(first, second)
        assertEquals(1, playbackQueueIndex(queue.size, second, queue::get))
    }
}
