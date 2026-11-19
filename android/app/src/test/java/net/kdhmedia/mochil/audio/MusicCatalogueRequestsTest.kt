package net.kdhmedia.mochil.audio

import kotlinx.coroutines.*
import org.junit.Assert.*
import org.junit.Test

class MusicCatalogueRequestsTest {
    @Test fun `navigation cancels all old catalogue children but not a download`() = runBlocking {
        val owner = SupervisorJob()
        val scope = CoroutineScope(coroutineContext + owner)
        try {
            val requests = MusicCatalogueRequests(scope)
            val old = requests.capture()
            val page = requests.launch { awaitCancellation() }
            val alphabet = requests.launch { awaitCancellation() }
            val download = scope.launch { awaitCancellation() }
            yield()
            val next = requests.advance()
            page.join(); alphabet.join()
            assertTrue(page.isCancelled)
            assertTrue(alphabet.isCancelled)
            assertTrue(download.isActive)
            assertFalse(requests.isCurrent(old))
            assertTrue(requests.isCurrent(next))
            var loaded = false
            requests.launch { loaded = true }.join()
            assertTrue(loaded)
        } finally { owner.cancelAndJoin() }
    }

    @Test fun `navigation before dispatch prevents obsolete work from starting`() = runBlocking {
        val owner = SupervisorJob()
        val scope = CoroutineScope(coroutineContext + owner)
        try {
            val requests = MusicCatalogueRequests(scope)
            var calls = 0
            val old = requests.launch { calls++ }
            requests.advance()
            old.join()
            assertEquals(0, calls)
        } finally { owner.cancelAndJoin() }
    }

    @Test fun `owner shutdown cancels the current catalogue context`() = runBlocking {
        val owner = SupervisorJob()
        val requests = MusicCatalogueRequests(CoroutineScope(coroutineContext + owner))
        val current = requests.launch { awaitCancellation() }
        yield()
        owner.cancelAndJoin()
        assertTrue(current.isCancelled)
        val late = requests.launch { fail("Disposed owner started work") }
        late.join()
        assertTrue(late.isCancelled)
    }
}
