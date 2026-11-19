package net.kdhmedia.mochil.video

import org.junit.Assert.*
import org.junit.Test

class VideoLocalResumeTest {
    private val owner = "a".repeat(64)
    private val entry = VideoPlaybackEntry("anime", "one", "Episode", "stream", "fallback", 10_000, 600.0, "subs", revision = "file-r1")
    private val pending = PendingVideoProgress(owner, "anime", "one", 20.0, 600.0, 3, false, mediaRevision = "file-r1")

    @Test fun `restores selected position without altering URLs metadata or other queue entries`() {
        val queue = VideoPlaybackQueue(listOf(entry.copy(itemId = "other"), entry), 1)
        val restored = restorePendingVideoPosition(queue, listOf(pending), owner, 2000)
        assertEquals(queue.entries, restored.entries)
        assertEquals(20_000L, restored.startPositionMs())
    }

    @Test fun `foreign and changed media checkpoints cannot affect resume`() {
        val queue = VideoPlaybackQueue(listOf(entry), 0)
        for (value in listOf(pending.copy(owner = "b".repeat(64)), pending.copy(libraryId = "movies"),
            pending.copy(itemId = "other"), pending.copy(mediaRevision = "changed"), pending.copy(finished = true))) {
            assertSame(queue, restorePendingVideoPosition(queue, listOf(value), owner, 2000))
        }
    }

    @Test fun `newer remote progress wins unless recent seek intent remains pending`() {
        val queue = VideoPlaybackQueue(listOf(entry.copy(positionMs = 30_000)), 0)
        assertSame(queue, restorePendingVideoPosition(queue, listOf(pending), owner, 2000))
        val seek = pending.copy(explicitSeekAtEpochMs = 1000)
        assertEquals(20_000L, restorePendingVideoPosition(queue, listOf(seek), owner, 2000).startPositionMs())
        assertSame(queue, restorePendingVideoPosition(queue, listOf(seek), owner, 601_001))
        val finished = queue.copy(entries = listOf(entry.copy(finished = true)))
        assertSame(finished, restorePendingVideoPosition(finished, listOf(pending), owner, 2000))
    }

    @Test fun `recovered position respects known duration and invalid selection is unchanged`() {
        val queue = VideoPlaybackQueue(listOf(entry.copy(durationSeconds = 15.0)), 0)
        assertEquals(15_000L, restorePendingVideoPosition(queue, listOf(pending), owner, 2000).startPositionMs())
        val invalid = queue.copy(currentIndex = 4)
        assertSame(invalid, restorePendingVideoPosition(invalid, listOf(pending), owner, 2000))
    }

    @Test fun `seek intent is not inherited when the underlying media changes`() {
        val prior = pending.copy(explicitSeekAtEpochMs = 1000)
        val changed = pending.copy(mediaRevision = "file-r2")
        assertEquals(0L, mergePendingVideoProgress(listOf(prior), changed).single().explicitSeekAtEpochMs)
        assertEquals(listOf(changed), decodePendingVideoProgressStrict(encodePendingVideoProgress(listOf(changed))))
    }
}
