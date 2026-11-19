package net.kdhmedia.mochil.audio

import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class OfflineAudioMutationGateTest {
    @Test fun `index remains readable during transfer and removal follows publication`() {
        val index = Any()
        val gate = OfflineAudioMutationGate(index)
        val started = CountDownLatch(1)
        val finish = CountDownLatch(1)
        val pool = Executors.newFixedThreadPool(3)
        var saved = "existing"
        try {
            val transfer = pool.submit {
                gate.transfer(
                    fetch = {
                        started.countDown()
                        check(finish.await(5, TimeUnit.SECONDS))
                        "downloaded"
                    },
                    publish = { assertTrue(Thread.holdsLock(index)); saved = it },
                )
            }
            assertTrue(started.await(5, TimeUnit.SECONDS))
            val read = pool.submit<String> { synchronized(index) { saved } }
            assertEquals("existing", read.get(2, TimeUnit.SECONDS))
            val removal = pool.submit<String> {
                gate.mutate {
                    assertTrue(Thread.holdsLock(index))
                    val previous = saved
                    saved = "removed"
                    previous
                }
            }
            finish.countDown()
            transfer.get(5, TimeUnit.SECONDS)
            assertEquals("downloaded", removal.get(5, TimeUnit.SECONDS))
            assertEquals("removed", synchronized(index) { saved })
        } finally {
            finish.countDown()
            pool.shutdownNow()
        }
    }

    @Test fun `failed transfer never publishes and releases mutation lock`() {
        val gate = OfflineAudioMutationGate(Any())
        var published = false
        val failure = runCatching {
            gate.transfer(fetch = { throw java.io.IOException("disconnected") }, publish = { published = true })
        }.exceptionOrNull()
        assertTrue(failure is java.io.IOException)
        assertEquals(false, published)
        val pool = Executors.newSingleThreadExecutor()
        try {
            assertEquals("retry", pool.submit<String> { gate.mutate { "retry" } }.get(2, TimeUnit.SECONDS))
        } finally {
            pool.shutdownNow()
        }
    }
}
