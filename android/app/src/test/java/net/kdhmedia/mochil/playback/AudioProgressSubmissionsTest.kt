package net.kdhmedia.mochil.playback

import org.junit.Assert.*
import org.junit.Test

class AudioProgressSubmissionsTest {
    private val first = PendingAudioProgress("podcasts", "one", 30.0, 600.0, owner = "owner")

    @Test fun `newer intent supersedes same item but not other item or owner`() {
        val submissions = AudioProgressSubmissions()
        val old = submissions.submit(first)!!
        val other = submissions.submit(first.copy(itemId = "two"))!!
        val owner = submissions.submit(first.copy(owner = "another"))!!
        val newest = submissions.submit(first.copy(positionSeconds = 40.0))!!
        assertFalse(submissions.isCurrent(old))
        assertTrue(submissions.isCurrent(newest))
        assertTrue(submissions.isCurrent(other))
        assertTrue(submissions.isCurrent(owner))
        assertNull(submissions.submit(first, replay = true))
    }

    @Test fun `passive replacement preserves pending seek intent and its original expiry`() {
        val submissions = AudioProgressSubmissions()
        submissions.submit(first.copy(explicitSeek = true, explicitSeekAtEpochMs = 1_000))
        val newer = submissions.submit(first.copy(positionSeconds = 35.0))!!
        assertEquals(35.0, newer.value.positionSeconds, 0.0)
        assertTrue(newer.value.explicitSeek)
        assertEquals(1_000L, newer.value.explicitSeekAtEpochMs)
        assertFalse(newer.value.explicitSeekForReplay(EXPLICIT_SEEK_REPLAY_WINDOW_MS + 1_001))
    }

    @Test fun `bounded ticket retention invalidates evicted work`() {
        val submissions = AudioProgressSubmissions(2)
        val old = submissions.submit(first)!!
        val second = submissions.submit(first.copy(itemId = "two"))!!
        val third = submissions.submit(first.copy(itemId = "three"))!!
        assertFalse(submissions.isCurrent(old))
        assertTrue(submissions.isCurrent(second))
        assertTrue(submissions.isCurrent(third))
    }

    @Test fun `acknowledged seek does not grant future passive updates explicit ownership`() {
        val submissions = AudioProgressSubmissions()
        val seek = submissions.submit(first.copy(explicitSeek = true, explicitSeekAtEpochMs = 1_000))!!
        submissions.acknowledge(seek)
        val passive = submissions.submit(first.copy(positionSeconds = 40.0))!!
        assertFalse(passive.value.explicitSeek)
        assertNull(submissions.submit(first, replay = true))
    }

    @Test fun `coalesced passive send retains exact durable acknowledgment without erasing newer commits`() {
        val submissions = AudioProgressSubmissions()
        submissions.submit(first, committed = first)
        val later = first.copy(positionSeconds = 40.0)
        val passive = submissions.submit(later)!!
        assertEquals(first, passive.committed)
        assertEquals(later, passive.value)
        val newest = submissions.submit(later, committed = later)!!
        assertEquals(later, newest.committed)
        assertFalse(submissions.isCurrent(passive))
    }
}
