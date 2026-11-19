package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.Collections
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.delay
import kotlinx.coroutines.withTimeout
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class VideoProgressReporterTest {
    @Test fun `untouched replay close preserves offline checkpoint while explicit seek is captured`() = runBlocking {
        val job = SupervisorJob()
        val scope = CoroutineScope(job + Dispatchers.Default)
        scope.cancel(); job.join()
        val credential = DeviceCredential("http://127.0.0.1:1", "tv", "fixture")
        var stored: ByteArray? = null
        val journal = VideoProgressJournal(videoProgressOwner(credential), { stored }, { stored = it; true })
        val reporter = VideoProgressReporter(scope, now = { 100 }, epochNow = { 1000 }, journal = journal)
        val entry = VideoPlaybackEntry("anime", "one", "Episode", "unused", "", 600_000, 600.0, "", progressRevision = 7, finished = true)
        reporter.submit(credential, entry, 10_000, 600_000, false, false)
        reporter.submit(credential, entry, 0, 600_000, false, false)
        assertEquals(10.0, journal.entries().single().positionSeconds, 0.0)
        reporter.submit(credential, entry, 0, 600_000, false, true)
        assertEquals(0.0, journal.entries().single().positionSeconds, 0.0)
        assertEquals(1000L, journal.entries().single().explicitSeekAtEpochMs)
    }
    @Test fun `slow request coalesces a burst and preserves the queued seek`() = runBlocking {
        val started = CountDownLatch(1)
        val release = CountDownLatch(1)
        val delivered = CountDownLatch(2)
        val requests = Collections.synchronizedList(mutableListOf<JSONObject>())
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        val job = SupervisorJob()
        val scope = CoroutineScope(job + Dispatchers.Default)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                val request = JSONObject(exchange.requestBody.use { it.readBytes().decodeToString() })
                requests += request
                if (requests.size == 1) {
                    started.countDown()
                    release.await(5, TimeUnit.SECONDS)
                }
                val body = JSONObject().put("revision", 1).put("position_seconds", request.getDouble("position_seconds")).put("finished", false).toString().encodeToByteArray()
                exchange.sendResponseHeaders(200, body.size.toLong())
                exchange.responseBody.use { it.write(body) }
                delivered.countDown()
            } finally { exchange.close() }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            val entry = VideoPlaybackEntry("anime", "one", "Episode", "unused", "", 0, 2000.0, "")
            var stored: ByteArray? = null
            val journal = VideoProgressJournal(videoProgressOwner(credential), { stored }, { stored = it; true })
            val reporter = VideoProgressReporter(scope, now = { 1000 }, epochNow = { 1000 }, journal = journal)
            reporter.submit(credential, entry, 10_000, 2_000_000, false, false)
            assertTrue(started.await(2, TimeUnit.SECONDS))
            val second = entry.copy(itemId = "two")
            reporter.submit(credential, second, 1000, 2_000_000, false, true)
            repeat(1000) { reporter.submit(credential, second, (it + 1L) * 1000, 2_000_000, false, false) }
            release.countDown()
            assertTrue(delivered.await(5, TimeUnit.SECONDS))
            assertEquals(2, requests.size)
            assertEquals(1000.0, requests[1].getDouble("position_seconds"), 0.0)
            assertTrue(requests[1].getBoolean("explicit_seek"))
            withTimeout(2000) { while (journal.entries().isNotEmpty()) delay(10) }
        } finally {
            release.countDown()
            scope.cancel()
            server.stop(0)
            job.join()
        }
    }

    @Test fun `capture remains durable when delivery scope has already closed`() = runBlocking {
        val job = SupervisorJob()
        val scope = CoroutineScope(job + Dispatchers.Default)
        scope.cancel()
        job.join()
        val credential = DeviceCredential("http://127.0.0.1:1", "tv", "fixture")
        var stored: ByteArray? = null
        val journal = VideoProgressJournal(videoProgressOwner(credential), { stored }, { stored = it; true })
        val reporter = VideoProgressReporter(scope, now = { 100 }, epochNow = { 1000 }, journal = journal)
        val entry = VideoPlaybackEntry("anime", "one", "Episode", "unused", "", 0, 600.0, "")
        reporter.submit(credential, entry, 10_000, 600_000, false, true)
        reporter.submit(credential, entry, 11_000, 600_000, false, false)
        val retained = journal.entries().single()
        assertEquals(11.0, retained.positionSeconds, 0.0)
        assertEquals(1000L, retained.explicitSeekAtEpochMs)
    }
}
