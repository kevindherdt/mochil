package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.launch
import kotlinx.coroutines.cancel
import org.junit.Assert.*
import org.junit.Test

class WikiStructureRecoveryTest {
    private val owner = listOf("server", "device", "vault")
    private val key = owner + listOf("page", "old")
    private val old = WikiEntry("page", "old", "Old.md", "Old", "md")
    private val moved = WikiEntry("page", "new", "New.md", "New", "md")

    @Test fun `acknowledged writes never repeat while tree recovery remains pending`() = runBlocking {
        val scenarios = listOf(
            WikiStructureChange(targetPath = "New.md") to listOf(moved),
            WikiStructureChange(targetPath = "Folder", folder = true) to listOf(WikiEntry("folder", path = "Folder")),
            WikiStructureChange(targetPath = "New.md", removedId = "old") to listOf(moved),
            WikiStructureChange(removedId = "old") to emptyList(),
        )
        for ((change, reflected) in scenarios) for (serverFailure in listOf(false, true)) {
            val guard = WikiStructureRecovery()
            var writes = 0
            var reads = 0
            suspend fun perform(): Unit = guard.mutate(key, change, "The change was saved", write = {
                writes++
                if (serverFailure) throw WikiException(500, "Index failed", mutationCommitted = true)
            }) { reads++; error("Read failed") }
            val failure = runCatching { perform() }.exceptionOrNull()
            assertTrue(failure is WikiMutationRefreshException)
            assertSame(failure, runCatching { perform() }.exceptionOrNull())
            assertEquals(1, writes)
            assertEquals(if (serverFailure) 0 else 1, reads)
            guard.refreshed(guard.tickets(owner), listOf(old))
            assertSame(failure, runCatching { guard.requireFresh(key) }.exceptionOrNull())
            guard.refreshed(guard.tickets(owner), reflected)
            assertNull(runCatching { guard.requireFresh(key) }.exceptionOrNull())
        }
    }

    @Test fun `old and unrelated tree requests cannot acknowledge newer recovery`() {
        val guard = WikiStructureRecovery()
        val change = WikiStructureChange(removedId = "old")
        val before = guard.tickets(owner)
        val failure = IllegalStateException("Saved")
        guard.record(key, change, failure)
        guard.refreshed(before, emptyList())
        assertSame(failure, runCatching { guard.requireFresh(key) }.exceptionOrNull())
        val earlier = guard.tickets(owner)
        guard.record(key, change, failure)
        guard.refreshed(earlier, emptyList())
        assertSame(failure, runCatching { guard.requireFresh(key) }.exceptionOrNull())
        assertTrue(guard.tickets(listOf("other", "device", "vault")).isEmpty())
    }

    @Test fun `move requires both target presence and old identity absence`() {
        val change = WikiStructureChange(targetPath = "New.md", removedId = "old")
        assertFalse(change.reflectedBy(listOf(old, moved)))
        assertFalse(change.reflectedBy(emptyList()))
        assertTrue(change.reflectedBy(listOf(moved)))
    }

    @Test fun `unacknowledged errors and cancellation retain their identity`() = runBlocking {
        for (failure in listOf(WikiException(500, "Failed"), CancellationException("Closing"))) {
            val guard = WikiStructureRecovery()
            val result = runCatching {
                guard.mutate(key, WikiStructureChange(removedId = "old"), "Removed", write = { throw failure }) { error("Unexpected read") }
            }.exceptionOrNull()
            assertSame(failure, result)
            assertNull(runCatching { guard.requireFresh(key) }.exceptionOrNull())
        }
    }

    @Test fun `cancellation after acknowledged write preserves barrier without starting refresh`() = runBlocking {
        val guard = WikiStructureRecovery()
        var writes = 0
        var reads = 0
        val job = launch {
            guard.mutate(key, WikiStructureChange(removedId = "old"), "Removed", write = {
                writes++
                cancel()
            }) { reads++ }
        }
        job.join()
        assertTrue(job.isCancelled)
        assertEquals(1, writes)
        assertEquals(0, reads)
        assertTrue(runCatching { guard.requireFresh(key) }.exceptionOrNull() is WikiMutationRefreshException)
        guard.refreshed(guard.tickets(owner), emptyList())
        assertNull(runCatching { guard.requireFresh(key) }.exceptionOrNull())
    }

    @Test fun `refresh cancellation keeps original cancellation and acknowledged mutation`() = runBlocking {
        val guard = WikiStructureRecovery()
        val cancelled = CancellationException("Leaving")
        val failure = runCatching {
            guard.mutate(key, WikiStructureChange(removedId = "old"), "Removed", write = {}) { throw cancelled }
        }.exceptionOrNull()
        assertSame(cancelled, failure)
        assertTrue(runCatching { guard.requireFresh(key) }.exceptionOrNull() is WikiMutationRefreshException)
    }
}
