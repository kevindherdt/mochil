package net.kdhmedia.mochil.playback

import net.kdhmedia.mochil.audio.AudioTrack
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class PlaybackQueueTest {
    @Test fun boundedQueueKeepsSelectedTrackAndNearbyContext() {
        val tracks = List(500) { index -> track(index) }
        val bounded = boundedPlaybackQueue(tracks, 410)
        assertEquals(PlaybackSnapshotStore.MAX_ITEMS, bounded.size)
        assertTrue(bounded.any { it.id == "410" })
        assertEquals("300", bounded.first().id)
        assertEquals("499", bounded.last().id)
    }

    @Test fun boundedQueueLeavesSmallQueuesUntouched() {
        val tracks = listOf(track(1), track(2))
        assertTrue(boundedPlaybackQueue(tracks, 1) === tracks)
    }

    @Test fun refreshedSelectedMetadataStillCentersTheQueueOnMediaIdentity() {
        val tracks = List(500) { index -> track(index) }
        val refreshed = track(410).copy(title = "Refreshed title", resumePositionSeconds = 27.0)

        val bounded = playbackQueueForSelection(tracks, refreshed)

        assertEquals(PlaybackSnapshotStore.MAX_ITEMS, bounded.size)
        assertTrue(bounded.any { it.id == "410" })
        assertEquals("300", bounded.first().id)
        assertEquals("499", bounded.last().id)
    }

    @Test fun selectedTrackRemainsPlayableWhenTheVisibleSourceWasReplaced() {
        val selected = track(99)

        assertEquals(listOf(selected), playbackQueueForSelection(emptyList(), selected))
        assertEquals(listOf(selected), playbackQueueForSelection(listOf(track(1), track(2)), selected))
    }

    @Test fun emptyPlaybackClearsItsRecoverySnapshot() {
        assertEquals(PlaybackSnapshotAction.Clear, playbackSnapshotAction(0))
        assertEquals(PlaybackSnapshotAction.Clear, playbackSnapshotAction(-1))
        assertEquals(PlaybackSnapshotAction.Save, playbackSnapshotAction(1))
    }

    private fun track(index: Int) = AudioTrack(index.toString(), "music", "Track $index", "Artist", "Album", 60.0)
}
