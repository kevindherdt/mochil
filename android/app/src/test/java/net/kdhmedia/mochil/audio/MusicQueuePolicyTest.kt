package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Test

class MusicQueuePolicyTest {
    private fun track(id: String) = AudioTrack(id, "music", id, "Artist", "Album", 180.0)

    @Test fun appendMovesAnExistingTrackToTheEndWithoutDuplicatingIt() {
        val first = track("first")
        val second = track("second")
        assertEquals(listOf(second, first), queueWithAppended(listOf(first, second), first))
    }

    @Test fun playNextInsertsAfterCurrentAndDoesNotDuplicate() {
        val first = track("first")
        val second = track("second")
        val third = track("third")
        assertEquals(listOf(first, third, second), queueWithNext(listOf(first, second, third), third, first))
    }

    @Test fun playNextStartsQueueWhenNoCurrentTrackExists() {
        val next = track("next")
        assertEquals(next, queueWithNext(listOf(track("old")), next, null).first())
    }

    @Test fun groupQueueActionsPreserveAlbumOrderAndRemoveDuplicates() {
        val current = track("current")
        val one = track("one")
        val two = track("two")
        assertEquals(listOf(current, one, two), queueWithNextAll(listOf(current, two), listOf(one, two), current))
        assertEquals(listOf(current, one, two), queueWithAppendedAll(listOf(current, two), listOf(one, two)))
    }

}
