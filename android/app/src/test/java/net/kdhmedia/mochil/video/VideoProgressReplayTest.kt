package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.Collections
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import kotlinx.coroutines.*
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class VideoProgressReplayTest {
    @Test fun `reopening replays retained position and removes it only after success`() = runBlocking {
        val failing = AtomicBoolean(true)
        val firstAttempt = CountDownLatch(1)
        val requests = Collections.synchronizedList(mutableListOf<JSONObject>())
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                val input = JSONObject(exchange.requestBody.use { it.readBytes().decodeToString() })
                requests += input
                if (failing.get()) exchange.sendResponseHeaders(503, -1)
                else {
                    val body = """{"revision":8,"position_seconds":12,"finished":false}""".encodeToByteArray()
                    exchange.sendResponseHeaders(200, body.size.toLong())
                    exchange.responseBody.use { it.write(body) }
                }
                firstAttempt.countDown()
            } finally { exchange.close() }
        }
        val firstJob = SupervisorJob()
        val secondJob = SupervisorJob()
        val firstScope = CoroutineScope(firstJob + Dispatchers.Default)
        val secondScope = CoroutineScope(secondJob + Dispatchers.Default)
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            var bytes: ByteArray? = null
            val journal = VideoProgressJournal(videoProgressOwner(credential), { bytes }, { bytes = it; true })
            val checkpoint = PendingVideoProgress(videoProgressOwner(credential), "anime", "one", 12.0, 600.0, 7, false, 1000, "file-r1")
            journal.put(checkpoint)
            VideoProgressReporter(firstScope, { 100 }, { 2000 }, journal).replay(credential).join()
            assertTrue(firstAttempt.await(3, TimeUnit.SECONDS))
            firstScope.cancel()
            firstJob.join()
            assertEquals(listOf(checkpoint), journal.entries())
            failing.set(false)
            // Reopened much later: retain position/revision, but expire backward-seek authority.
            VideoProgressReporter(secondScope, { 500 }, { 700_000 }, journal).replay(credential).join()
            withTimeout(3000) { while (journal.entries().isNotEmpty()) delay(10) }
            assertEquals(2, requests.size)
            assertEquals(7L, requests[1].getLong("expected_revision"))
            assertEquals("file-r1", requests[1].getString("expected_media_revision"))
            assertEquals(12.0, requests[1].getDouble("position_seconds"), 0.0)
            assertTrue(requests[0].getBoolean("explicit_seek"))
            assertFalse(requests[1].getBoolean("explicit_seek"))
        } finally {
            firstScope.cancel(); secondScope.cancel()
            server.stop(0)
            firstJob.join(); secondJob.join()
        }
    }
}
