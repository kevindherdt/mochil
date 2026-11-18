package net.kdhmedia.mochil.playback

import java.io.IOException
import java.util.Collections
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import org.junit.Assert.*
import org.junit.Test

class PlaybackProgressDeliveryQueueTest {
    @Test fun `late stale producer cannot replace a newer queued submission`() = runBlocking {
        withTimeout(5_000) {
            val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
            try {
                val queue = PlaybackProgressDeliveryQueue(scope)
                val submissions = AudioProgressSubmissions()
                val entered = CompletableDeferred<Unit>()
                val release = CompletableDeferred<Unit>()
                val done = CompletableDeferred<Unit>()
                val sent = Collections.synchronizedList(mutableListOf<Double>())
                queue.submit("active") { entered.complete(Unit); release.await() }
                entered.await()
                val value = PendingAudioProgress("podcasts", "one", 30.0, 600.0)
                val stale = submissions.submit(value, replay = true)!!
                val fresh = submissions.submit(value.copy(positionSeconds = 40.0))!!
                assertTrue(queue.submit(fresh.key, { submissions.isCurrent(fresh) }) { sent += fresh.value.positionSeconds })
                assertFalse(queue.submit(stale.key, { submissions.isCurrent(stale) }) { sent += stale.value.positionSeconds })
                queue.submit("done") { done.complete(Unit) }
                release.complete(Unit)
                done.await()
                assertEquals(listOf(40.0), sent.toList())
            } finally { scope.cancel() }
        }
    }

    @Test fun `slow sender retains only latest queued work per item`() = runBlocking {
        withTimeout(5_000) {
            val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
            try {
                val queue = PlaybackProgressDeliveryQueue(scope)
                val entered = CompletableDeferred<Unit>()
                val release = CompletableDeferred<Unit>()
                val done = CompletableDeferred<Unit>()
                val sent = Collections.synchronizedList(mutableListOf<Int>())
                queue.submit("active") { entered.complete(Unit); release.await(); sent += -1 }
                entered.await()
                repeat(1_000) { value -> assertTrue(queue.submit("same-item") { sent += value }) }
                queue.submit("other") { sent += 2_000; done.complete(Unit) }
                release.complete(Unit)
                done.await()
                assertEquals(listOf(-1, 999, 2_000), sent.toList())
            } finally { scope.cancel() }
        }
    }

    @Test fun `pending capacity is bounded and work continues after a send exception`() = runBlocking {
        withTimeout(5_000) {
            val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
            try {
                val failures = Collections.synchronizedList(mutableListOf<Exception>())
                val queue = PlaybackProgressDeliveryQueue(scope, capacity = 2, onFailure = { failures += it })
                val entered = CompletableDeferred<Unit>()
                val release = CompletableDeferred<Unit>()
                val done = CompletableDeferred<Unit>()
                queue.submit("active") { entered.complete(Unit); release.await() }
                entered.await()
                queue.submit("evicted") { error("Must be evicted") }
                queue.submit("failing") { throw IOException("Offline") }
                queue.submit("last") { done.complete(Unit) }
                release.complete(Unit)
                done.await()
                assertEquals(1, failures.size)
                assertTrue(failures.single() is IOException)
            } finally { scope.cancel() }
        }
    }

    @Test fun `service cancellation rejects new work and never starts queued work`() = runBlocking {
        withTimeout(5_000) {
            val job = SupervisorJob()
            val scope = CoroutineScope(job + Dispatchers.Default)
            try {
                val queue = PlaybackProgressDeliveryQueue(scope)
                val entered = CompletableDeferred<Unit>()
                val release = CompletableDeferred<Unit>()
                var queuedRan = false
                queue.submit("active") { entered.complete(Unit); release.await() }
                entered.await()
                queue.submit("pending") { queuedRan = true }
                job.cancel()
                job.join()
                assertFalse(queue.submit("late") { error("Must not run") })
                assertFalse(queuedRan)
            } finally { scope.cancel() }
        }
    }
}
