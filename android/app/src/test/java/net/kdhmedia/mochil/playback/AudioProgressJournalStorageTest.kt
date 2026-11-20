package net.kdhmedia.mochil.playback

import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicReference
import org.junit.Assert.*
import org.junit.Test

class AudioProgressJournalStorageTest {
    private val owner = audioProgressOwnerIdentity("device")
    private fun checkpoint(id: String) = PendingAudioProgress("podcasts", id, 20.0, 200.0, owner = owner)

    @Test fun `unreadable journal is not overwritten or cleared`() {
        for (read in listOf<() -> ByteArray?>(
            { error("Keystore unavailable") },
            { "[{broken".encodeToByteArray() },
        )) {
            var writes = 0
            val journal = AudioProgressJournal(read, { writes++; true }, { writes++; true })
            assertTrue(journal.entries("device").isEmpty())
            assertNull(journal.put(checkpoint("new")))
            journal.removeIfCurrent(checkpoint("old"))
            assertEquals(0, writes)
        }
    }

    @Test fun `two overlapping journal instances preserve both checkpoints`() {
        val bytes = AtomicReference<ByteArray?>(null)
        val reading = CountDownLatch(1)
        val release = CountDownLatch(1)
        val secondStarted = CountDownLatch(1)
        val secondRead = CountDownLatch(1)
        val executor = Executors.newFixedThreadPool(2)
        val first = AudioProgressJournal({
            reading.countDown()
            check(release.await(5, TimeUnit.SECONDS))
            bytes.get()
        }, { bytes.set(it); true }, { bytes.set(null); true })
        val second = AudioProgressJournal({ secondRead.countDown(); bytes.get() },
            { bytes.set(it); true }, { bytes.set(null); true })
        try {
            val firstWrite = executor.submit<PendingAudioProgress?> { first.put(checkpoint("one")) }
            assertTrue(reading.await(5, TimeUnit.SECONDS))
            val secondWrite = executor.submit<PendingAudioProgress?> {
                secondStarted.countDown()
                second.put(checkpoint("two"))
            }
            assertTrue(secondStarted.await(5, TimeUnit.SECONDS))
            assertFalse("Second instance must wait for the first transaction", secondRead.await(150, TimeUnit.MILLISECONDS))
            release.countDown()
            assertNotNull(firstWrite.get(5, TimeUnit.SECONDS))
            assertNotNull(secondWrite.get(5, TimeUnit.SECONDS))
            assertEquals(setOf("one", "two"), second.entries("device").map { it.itemId }.toSet())
        } finally {
            release.countDown()
            executor.shutdownNow()
        }
    }

    @Test fun `old acknowledgement cannot remove a checkpoint saved by another instance`() {
        var bytes: ByteArray? = null
        var writes = 0
        fun journal() = AudioProgressJournal({ bytes }, { bytes = it; writes++; true }, { bytes = null; writes++; true })
        val first = journal()
        val old = first.put(checkpoint("one"))!!
        val second = journal()
        val newer = second.put(old.copy(positionSeconds = 30.0))!!
        first.removeIfCurrent(old)
        assertEquals(listOf(newer), first.entries("device"))
        assertEquals(2, writes)
        second.removeIfCurrent(newer)
        assertTrue(first.entries("device").isEmpty())
        assertEquals(3, writes)
    }
}
