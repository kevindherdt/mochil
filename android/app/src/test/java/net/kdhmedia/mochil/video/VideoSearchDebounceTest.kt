package net.kdhmedia.mochil.video

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.yield
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test

class VideoSearchDebounceTest {
    @Test fun invalidatesTheOldResponseBeforeTheDelayCompletes() = runBlocking {
        val requests = VideoRequestGeneration()
        val oldPage = requests.advance()
        val gate = CompletableDeferred<Unit>()
        val debounce = VideoSearchDebounce(this, requests) { gate.await() }
        var loads = 0
        debounce.submit { loads++ }
        assertFalse(requests.isCurrent(oldPage))
        yield()
        assertEquals(0, loads)
        gate.complete(Unit)
        yield()
        assertEquals(1, loads)
    }

    @Test fun navigationDuringDelayDoesNotReloadTheNewLibrary() = runBlocking {
        val requests = VideoRequestGeneration()
        val gate = CompletableDeferred<Unit>()
        val debounce = VideoSearchDebounce(this, requests) { gate.await() }
        var loads = 0
        debounce.submit { loads++ }
        yield()
        requests.advance() // The new library/show/season starts its own request.
        gate.complete(Unit)
        yield()
        assertEquals(0, loads)
    }

    @Test fun rapidTypingLoadsOnlyTheLatestQuery() = runBlocking {
        val requests = VideoRequestGeneration()
        val gate = CompletableDeferred<Unit>()
        val debounce = VideoSearchDebounce(this, requests) { gate.await() }
        val loads = mutableListOf<String>()
        debounce.submit { loads += "b" }
        yield()
        debounce.submit { loads += "ba" }
        debounce.submit { loads += "baki" }
        yield()
        gate.complete(Unit)
        yield()
        assertEquals(listOf("baki"), loads)
    }
}
