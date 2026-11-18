package net.kdhmedia.mochil.video

import org.junit.Assert.*
import org.junit.Test

class VideoProgressJournalTest {
    private val owner = "a".repeat(64)
    private fun checkpoint() = PendingVideoProgress(owner, "movies", "one", 10.0, 100.0, 2, false)

    @Test fun `unreadable persisted progress is not overwritten or acknowledged`() {
        val malformed = "[{broken".encodeToByteArray()
        val oversized = ByteArray(MAX_VIDEO_PROGRESS_JOURNAL_BYTES + 1)
        for (read in listOf<() -> ByteArray?>(
            { error("Keystore unavailable") }, { malformed }, { oversized },
        )) {
            var writes = 0
            val journal = VideoProgressJournal(owner, read, { writes++; true })
            // Reporter catches this read failure and emits its persistence diagnostic.
            assertThrows(Exception::class.java) { journal.entries() }
            assertNull(journal.put(checkpoint()))
            assertFalse(journal.removeIfCurrent(checkpoint()))
            assertEquals(0, writes)
        }
    }

    @Test fun `recreated journal retains merged checkpoint and exact acknowledgement`() {
        var bytes: ByteArray? = null
        fun journal() = VideoProgressJournal(owner, { bytes }, { bytes = it; true })
        val first = journal()
        val original = first.put(checkpoint().copy(explicitSeekAtEpochMs = 100))!!
        val reopened = journal()
        assertEquals(listOf(original), reopened.entries())
        val latest = reopened.put(checkpoint().copy(positionSeconds = 20.0))!!
        assertEquals(100L, latest.explicitSeekAtEpochMs)
        assertFalse(first.removeIfCurrent(original))
        assertEquals(listOf(latest), first.entries())
        assertTrue(first.removeIfCurrent(latest))
        assertTrue(reopened.entries().isEmpty())
    }

    @Test fun `failed persistence neither claims durability nor discards existing checkpoint`() {
        var bytes: ByteArray? = null
        var succeeds = true
        val journal = VideoProgressJournal(owner, { bytes }, { if (succeeds) { bytes = it; true } else false })
        val original = journal.put(checkpoint())!!
        succeeds = false
        assertNull(journal.put(checkpoint().copy(positionSeconds = 20.0)))
        assertFalse(journal.removeIfCurrent(original))
        assertEquals(listOf(original), journal.entries())
        val throwing = VideoProgressJournal(owner, { bytes }, { error("Storage unavailable") })
        assertNull(throwing.put(checkpoint()))
        assertFalse(throwing.removeIfCurrent(original))
    }

    @Test fun `foreign owners and malformed entries cannot be replayed or acknowledged`() {
        val foreign = checkpoint().copy(owner = "b".repeat(64))
        val bytes = encodePendingVideoProgress(listOf(foreign))
        var writes = 0
        val journal = VideoProgressJournal(owner, { bytes }, { writes++; true })
        assertTrue(journal.entries().isEmpty())
        assertNull(journal.put(foreign))
        assertFalse(journal.removeIfCurrent(foreign))
        assertNull(journal.put(checkpoint().copy(durationSeconds = 0.0)))
        assertEquals(0, writes)
    }
}
