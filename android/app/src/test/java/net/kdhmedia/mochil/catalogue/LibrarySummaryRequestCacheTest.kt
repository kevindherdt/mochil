package net.kdhmedia.mochil.catalogue

import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicInteger
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Test

class LibrarySummaryRequestCacheTest {
    private val credential = DeviceCredential("https://example.test", "device", "secret")
    private val libraries = listOf(LibrarySummary("books", "Books", "books", 10))

    @Test fun `coalesces concurrent reads for one device`() {
        val cache = LibrarySummaryRequestCache(ttlMillis = 5_000)
        val requests = AtomicInteger()
        val entered = CountDownLatch(1)
        val release = CountDownLatch(1)
        val pool = Executors.newFixedThreadPool(2)
        val first = pool.submit<List<LibrarySummary>> { cache.load(credential) {
            requests.incrementAndGet(); entered.countDown(); release.await(); libraries
        } }
        entered.await()
        val second = pool.submit<List<LibrarySummary>> { cache.load(credential) { requests.incrementAndGet(); libraries } }
        release.countDown()
        assertEquals(libraries, first.get())
        assertEquals(libraries, second.get())
        assertEquals(1, requests.get())
        pool.shutdownNow()
    }

    @Test fun `short cache expires and explicit refresh bypasses it`() {
        var now = 100L
        val cache = LibrarySummaryRequestCache(nowMillis = { now }, ttlMillis = 10)
        val requests = AtomicInteger()
        fun load(refresh: Boolean = false) = cache.load(credential, refresh) { requests.incrementAndGet(); libraries }
        load(); load()
        assertEquals(1, requests.get())
        load(refresh = true)
        assertEquals(2, requests.get())
        now += 11
        load()
        assertEquals(3, requests.get())
    }
}
