package net.kdhmedia.mochil.webdav

import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Test

/**
 * WebDavPageCache decrypts/parses its whole bounded blob through [DecodedWebDavCacheMemory]. A
 * page navigation must not pay for that round trip when nothing has changed since the last read.
 */
class DecodedWebDavCacheMemoryTest {
    private val samplePage = CachedWebDavPage("conn", "cred", WebDavPage("https://example/dav/", emptyList(), "v1"), 1000L)

    @Test fun decodesOnlyOnceAcrossRepeatedReads() {
        var loadCalls = 0
        val memory = DecodedWebDavCacheMemory(
            load = { loadCalls++; encodeWebDavCache(listOf(samplePage)) },
            save = { true },
        )
        val first = memory.decoded()
        val second = memory.decoded()
        val third = memory.decoded()

        assertEquals(1, loadCalls)
        assertSame("Repeated reads must return the same in-memory list, not a fresh decode.", first, second)
        assertSame(second, third)
        assertEquals(1, first.size)
    }

    @Test fun replaceAdoptsTheNewListWithoutReReadingTheStore() {
        var loadCalls = 0
        val saved = mutableListOf<ByteArray>()
        val memory = DecodedWebDavCacheMemory(
            load = { loadCalls++; encodeWebDavCache(listOf(samplePage)) },
            save = { payload -> saved += payload; true },
        )
        memory.decoded() // warms the cache from the "store"
        val updated = listOf(samplePage, samplePage.copy(connectionId = "conn2"))
        memory.replace(updated, encodeWebDavCache(updated))

        assertEquals(1, loadCalls)
        assertEquals(2, memory.decoded().size)
        assertEquals(1, loadCalls) // decoded() after replace() still must not call load() again
        assertEquals(1, saved.size)
    }

    @Test fun replaceIsDiscardedWhenTheUnderlyingSaveFails() {
        val memory = DecodedWebDavCacheMemory(
            load = { encodeWebDavCache(listOf(samplePage)) },
            save = { false },
        )
        memory.decoded()
        memory.replace(listOf(samplePage, samplePage), ByteArray(0))

        assertEquals("A failed save must not leave the in-memory cache out of sync with disk.", 1, memory.decoded().size)
    }
}
