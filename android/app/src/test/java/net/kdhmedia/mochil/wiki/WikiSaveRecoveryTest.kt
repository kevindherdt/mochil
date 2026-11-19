package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertNull
import org.junit.Assert.assertSame
import org.junit.Test

class WikiSaveRecoveryTest {
    private val key = listOf("server", "device", "vault", "page")
    private fun failure() = WikiSavedButReloadFailedException(IllegalStateException("Index unavailable"))

    @Test fun `read started before acknowledgement cannot clear recovery`() {
        val guard = WikiSaveRecovery()
        val oldRead = guard.ticket(key)
        val failure = failure()
        guard.record(key, failure)
        guard.refreshed(key, oldRead)
        assertSame(failure, runCatching { guard.requireFresh(key) }.exceptionOrNull())
        guard.refreshed(key, guard.ticket(key))
        assertNull(runCatching { guard.requireFresh(key) }.exceptionOrNull())
    }

    @Test fun `replacement acknowledgement survives earlier reads and other identities`() {
        val guard = WikiSaveRecovery()
        guard.record(key, failure())
        val oldRead = guard.ticket(key)
        val latest = failure()
        guard.record(key, latest)
        guard.refreshed(key, oldRead)
        assertSame(latest, runCatching { guard.requireFresh(key) }.exceptionOrNull())
        for (index in key.indices) {
            val other = key.toMutableList().apply { this[index] = "other" }
            assertNull(runCatching { guard.requireFresh(other) }.exceptionOrNull())
            guard.refreshed(other, guard.ticket(key))
            assertSame(latest, runCatching { guard.requireFresh(key) }.exceptionOrNull())
        }
    }
}
