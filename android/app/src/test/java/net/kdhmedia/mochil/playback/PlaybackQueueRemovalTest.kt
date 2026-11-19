package net.kdhmedia.mochil.playback

import org.junit.Assert.*
import org.junit.Test

class PlaybackQueueRemovalTest {
    @Test fun `adjacent duplicates use two updates around the playing entry`() {
        val queue = (0..199).map { "$it" }.toMutableList()
        val ranges = queuedCopyRemovalRanges(queue.size, 100, queue.toSet(), queue::get)
        assertEquals(listOf(QueueRemovalRange(101, 200), QueueRemovalRange(0, 100)), ranges)
        ranges.forEach { queue.subList(it.start, it.endExclusive).clear() }
        assertEquals(listOf("100"), queue)
    }

    @Test fun `ranges preserve unrelated entries and handle queue edges`() {
        for (current in -1..6) {
            val original = listOf("a", "a", "keep", "a", "keep", "a")
            val queue = original.toMutableList()
            val ranges = queuedCopyRemovalRanges(queue.size, current, setOf("a"), queue::get)
            ranges.forEach { queue.subList(it.start, it.endExclusive).clear() }
            assertEquals(original.filterIndexed { index, key -> index == current || key != "a" }, queue)
        }
        assertTrue(queuedCopyRemovalRanges(0, -1, setOf("a")) { error("Empty queue") }.isEmpty())
        assertTrue(queuedCopyRemovalRanges(6, 0, emptySet()) { error("No additions") }.isEmpty())
    }

    @Test fun `duplicates are removed backwards without removing the playing occurrence`() {
        val queue = mutableListOf("one", "two", "one", "three", "two")
        val indices = queuedCopyRemovalIndices(queue.size, 2, setOf("one", "two"), queue::get)
        assertEquals(listOf(4, 1, 0), indices)
        indices.forEach { queue.removeAt(it) }
        assertEquals(listOf("one", "three"), queue)
    }

    @Test fun `bulk duplicate lookup visits each non playing entry only once`() {
        var visits = 0
        val indices = queuedCopyRemovalIndices(200, 100, (0..199).mapTo(mutableSetOf()) { "$it" }) { visits++; "$it" }
        assertEquals(199, visits)
        assertEquals(199, indices.size)
        assertFalse(100 in indices)
        assertTrue(queuedCopyRemovalIndices(200, 100, emptySet()) { error("No work needed") }.isEmpty())
        assertTrue(queuedCopyRemovalIndices(0, -1, setOf("one")) { error("Empty queue") }.isEmpty())
    }
}
