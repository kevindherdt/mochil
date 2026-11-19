package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import kotlinx.coroutines.*
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class VideoProgressReporterRebaseRegressionTest {
    @Test fun `failed rebase persistence retains sent checkpoint`() = runBlocking {
        val failure = CountDownLatch(1)
        val release = CountDownLatch(1)
        val secondRequest = CountDownLatch(1)
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                exchange.requestBody.readBytes()
                if (exchange.requestURI.path.endsWith("/other")) secondRequest.countDown()
                val response = """{"revision":8,"position_seconds":40,"finished":false}""".toByteArray()
                exchange.sendResponseHeaders(200, response.size.toLong())
                exchange.responseBody.use { it.write(response) }
            } finally { exchange.close() }
        }
        server.start()
        val job = SupervisorJob()
        val scope = CoroutineScope(job + Dispatchers.Default)
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            var bytes: ByteArray? = null
            val writes = AtomicInteger()
            val journal = VideoProgressJournal(videoProgressOwner(credential), { bytes }, {
                if (writes.incrementAndGet() == 2) false else { bytes = it; true }
            })
            val reporter = VideoProgressReporter(scope, now = { 100 }, epochNow = { 1000 }, journal = journal,
                onPersistenceFailure = { failure.countDown(); release.await(5, TimeUnit.SECONDS) })
            val entry = VideoPlaybackEntry("lib", "item", "", "", "", 0, 100.0, "", progressRevision = 7)
            reporter.submit(credential, entry, 40_000, 100_000, false, false)
            assertTrue(failure.await(5, TimeUnit.SECONDS))
            reporter.submit(credential, entry.copy(itemId = "other"), 40_000, 100_000, false, false)
            release.countDown()
            assertTrue(secondRequest.await(5, TimeUnit.SECONDS))
            assertTrue(journal.entries().any { it.itemId == "item" && it.positionSeconds == 40.0 })
        } finally { release.countDown(); scope.cancel(); server.stop(0); job.join() }
    }
    @Test fun `submission uses durable rebased revision`() = runBlocking {
        val received = CountDownLatch(1)
        var revision = -1L
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                revision = JSONObject(exchange.requestBody.readBytes().decodeToString()).getLong("expected_revision")
                val response = """{"revision":9,"position_seconds":40,"finished":false}""".toByteArray()
                exchange.sendResponseHeaders(200, response.size.toLong())
                exchange.responseBody.use { it.write(response) }
            } finally { received.countDown(); exchange.close() }
        }
        server.start()
        val job = SupervisorJob()
        val scope = CoroutineScope(job + Dispatchers.Default)
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            var bytes: ByteArray? = null
            val journal = VideoProgressJournal(videoProgressOwner(credential), { bytes }, { bytes = it; true })
            val old = PendingVideoProgress(videoProgressOwner(credential), "lib", "item", 30.0, 100.0, 7, false)
            assertTrue(journal.rebaseNewer(old, 8))
            val reporter = VideoProgressReporter(scope, now = { 100 }, epochNow = { 1000 }, journal = journal)
            val entry = VideoPlaybackEntry("lib", "item", "", "", "", 0, 100.0, "", progressRevision = 7)
            reporter.submit(credential, entry, 40_000, 100_000, false, false)
            assertTrue(received.await(5, TimeUnit.SECONDS))
            assertEquals(8L, revision)
        } finally { scope.cancel(); server.stop(0); job.join() }
    }
}
