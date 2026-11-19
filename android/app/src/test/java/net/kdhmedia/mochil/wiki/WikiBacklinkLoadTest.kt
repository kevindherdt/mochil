package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.cancel
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import org.junit.Assert.*
import org.junit.Test

class WikiBacklinkLoadTest {
    @Test fun `successful empty results clear backlinks but failures do not`() = runBlocking {
        var deliveries = 0
        loadWikiBacklinks({ emptyList() }) { assertTrue(it.isEmpty()); deliveries++ }
        loadWikiBacklinks({ error("Offline") }) { deliveries++ }
        assertEquals(1, deliveries)
    }

    @Test fun `cancelled load never delivers even if underlying operation returns`() = runBlocking {
        var delivered = false
        val job = launch {
            loadWikiBacklinks({
                currentCoroutineContext().cancel()
                emptyList()
            }) { delivered = true }
        }
        job.join()
        assertTrue(job.isCancelled)
        assertFalse(delivered)
    }

    @Test fun `explicit cancellation is propagated without delivering empty fallback`() = runBlocking {
        val cancellation = CancellationException("Left page")
        try {
            loadWikiBacklinks({ throw cancellation }) { fail("Unexpected delivery") }
            fail("Cancellation swallowed")
        } catch (actual: CancellationException) {
            assertSame(cancellation, actual)
        }
    }
}
