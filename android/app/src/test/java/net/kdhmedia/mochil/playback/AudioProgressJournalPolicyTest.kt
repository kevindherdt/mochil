package net.kdhmedia.mochil.playback

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AudioProgressJournalPolicyTest {
    @Test fun `replay reads avoid redundant persistence but still prune foreign owners`() {
        val own = PendingAudioProgress("podcasts", "one", 30.0, 600.0,
            owner = audioProgressOwnerIdentity("alice-device"))
        val foreign = own.copy(owner = audioProgressOwnerIdentity("bob-device"))
        val writes = mutableListOf<List<PendingAudioProgress>>()
        val persist: (List<PendingAudioProgress>) -> Boolean = { writes.add(it); true }
        repeat(3) {
            assertEquals(listOf(own), readAudioProgressForOwner(listOf(own), "alice-device", persist))
            assertTrue(readAudioProgressForOwner(emptyList(), "alice-device", persist).isEmpty())
        }
        assertTrue(writes.isEmpty())
        assertEquals(listOf(own), readAudioProgressForOwner(listOf(own, foreign), "alice-device", persist))
        assertEquals(listOf(listOf(own)), writes)
        // Persistence failure must never expose another account's entries for replay.
        assertTrue(readAudioProgressForOwner(listOf(foreign), "alice-device") { false }.isEmpty())
    }

    @Test fun `durable outbox is isolated by paired account and rejects legacy rows`() {
        val aliceOwner = audioProgressOwnerIdentity("alice-device")
        val alice = PendingAudioProgress("podcasts", "one", 30.0, 600.0, owner = aliceOwner)
        val bob = alice.copy(itemId = "two", owner = audioProgressOwnerIdentity("bob-device"))
        val legacy = alice.copy(itemId = "legacy", owner = "")
        assertEquals(listOf(alice), audioProgressEntriesForOwner(listOf(alice, bob, legacy), "alice-device"))
        assertTrue(audioProgressEntriesForOwner(listOf(alice, bob, legacy), "new-device").isEmpty())
        assertFalse(aliceOwner.contains("alice"))
    }

    @Test fun `a later pause preserves an unsent explicit seek`() {
        val seek = PendingAudioProgress("podcasts", "episode", 30.0, 600.0, explicitSeek = true, explicitSeekAtEpochMs = 1_000)
        val pause = seek.copy(positionSeconds = 35.0, explicitSeek = false)
        val merged = mergePendingAudioProgress(listOf(seek), pause, 64)
        assertEquals(1, merged.size)
        assertEquals(35.0, merged.single().positionSeconds, 0.0)
        assertTrue(merged.single().explicitSeek)
        assertEquals(1_000, merged.single().explicitSeekAtEpochMs)
    }

    @Test fun `unrelated passive entries do not acquire seek ownership`() {
        val first = PendingAudioProgress("podcasts", "one", 30.0, 600.0, explicitSeek = true, explicitSeekAtEpochMs = 1_000)
        val second = PendingAudioProgress("podcasts", "two", 20.0, 600.0)
        val merged = mergePendingAudioProgress(listOf(first), second, 64)
        assertTrue(merged.first().explicitSeek)
        assertFalse(merged.last().explicitSeek)
    }

    @Test fun `fresh explicit seek replay retains bounded backward ownership`() {
        val now = 1_000_000L
        val pending = PendingAudioProgress("podcasts", "episode", 30.0, 600.0, explicitSeek = true, explicitSeekAtEpochMs = now - 30_000)

        assertTrue(pending.explicitSeekForReplay(now))
    }

    @Test fun `stale explicit seek replay is downgraded to passive progress`() {
        val now = 1_000_000L
        val pending = PendingAudioProgress(
            "podcasts",
            "episode",
            30.0,
            600.0,
            explicitSeek = true,
            explicitSeekAtEpochMs = now - EXPLICIT_SEEK_REPLAY_WINDOW_MS - 1,
        )

        assertFalse(pending.explicitSeekForReplay(now))
        assertFalse(pending.copy(explicitSeekAtEpochMs = 0).explicitSeekForReplay(now))
    }
}
