package net.kdhmedia.mochil.playback

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class AudioProgressCommitTest {
    private val seek = PendingAudioProgress("podcasts", "episode", 30.0, 600.0, true, 1_000, "owner")

    @Test fun `pause submission returns exact persisted merged seek record for send and acknowledgment`() {
        val pause = seek.copy(positionSeconds = 35.0, explicitSeek = false, explicitSeekAtEpochMs = 0)
        var persisted = emptyList<PendingAudioProgress>()
        val committed = commitPendingAudioProgress(listOf(seek), pause, 64) { persisted = it; true }!!
        assertEquals(persisted.single(), committed)
        assertTrue(committed.explicitSeek)
        assertEquals(1_000L, committed.explicitSeekAtEpochMs)
        assertEquals(35.0, committed.positionSeconds, 0.0)
        assertTrue(persisted.filterNot { it == committed }.isEmpty())
        // Original unmerged pause would fail the journal's equality acknowledgment.
        assertEquals(1, persisted.filterNot { it == pause }.size)
    }

    @Test fun `failed persistence does not produce a committed record`() {
        assertNull(commitPendingAudioProgress(listOf(seek), seek, 64) { false })
    }

    @Test fun `commit remains owner isolated and never inherits another item seek`() {
        val other = seek.copy(itemId = "other", explicitSeek = false)
        var persisted = emptyList<PendingAudioProgress>()
        val committed = commitPendingAudioProgress(listOf(seek.copy(owner = "someone-else")), other, 64) { persisted = it; true }
        assertEquals(other, committed)
        assertEquals(listOf(other), persisted)
    }

    @Test fun `older acknowledgment cannot match a newer pause`() {
        val committed = commitPendingAudioProgress(listOf(seek), seek, 64) { true }!!
        var persisted = emptyList<PendingAudioProgress>()
        val next = seek.copy(positionSeconds = 40.0, explicitSeek = false)
        commitPendingAudioProgress(listOf(committed), next, 64) { persisted = it; true }
        assertEquals(1, persisted.filterNot { it == committed }.size)
    }
}
