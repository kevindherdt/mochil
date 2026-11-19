package net.kdhmedia.mochil.auth

import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicReference
import org.junit.Assert.*
import org.junit.Test

class PersistentKeyInitializationTest {
    @Test fun `concurrent store instances create one persistent key`() {
        val executor = Executors.newFixedThreadPool(8)
        val start = CountDownLatch(1)
        val created = AtomicInteger()
        val stored = AtomicReference<Any?>()
        try {
            val requests = (0 until 32).map {
                executor.submit<Any> {
                    assertTrue(start.await(2, TimeUnit.SECONDS))
                    getOrCreatePersistentKey(stored::get) {
                        created.incrementAndGet()
                        Any().also(stored::set)
                    }
                }
            }
            start.countDown()
            val keys = requests.map { it.get(3, TimeUnit.SECONDS) }
            assertEquals(1, created.get())
            keys.forEach { assertSame(stored.get(), it) }
        } finally { start.countDown(); executor.shutdownNow() }
    }

    @Test fun `existing keys are retained and failed creation can be retried`() {
        val existing = Any()
        assertSame(existing, getOrCreatePersistentKey({ existing }) { error("Must not replace a key") })
        assertTrue(runCatching { getOrCreatePersistentKey<Any>({ null }) { error("Keystore unavailable") } }.isFailure)
        assertSame(existing, getOrCreatePersistentKey({ null }) { existing })
    }
}
