package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.runBlocking
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class VideoResumeImmediateCloseTest {
    @Test fun `closing restored local progress uploads it before clearing checkpoint`() = runBlocking {
        val delivered = CountDownLatch(1)
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        val requests = mutableListOf<JSONObject>()
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                val body = JSONObject(exchange.requestBody.use { it.readBytes().decodeToString() })
                synchronized(requests) { requests.add(body) }
                val response = JSONObject().put("revision", 4).put("position_seconds", body.getDouble("position_seconds")).put("finished", false).toString().encodeToByteArray()
                exchange.sendResponseHeaders(200, response.size.toLong())
                exchange.responseBody.use { it.write(response) }
                delivered.countDown()
            } finally { exchange.close() }
        }
        server.start()
        val job = SupervisorJob()
        val scope = CoroutineScope(job + Dispatchers.Default)
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            var stored: ByteArray? = null
            val journal = VideoProgressJournal(videoProgressOwner(credential), { stored }, { stored = it; true })
            val entry = VideoPlaybackEntry("anime", "one", "Episode", "stream", "", 10_000, 600.0, "", revision = "file-r1", progressRevision = 3)
            journal.put(PendingVideoProgress(videoProgressOwner(credential), "anime", "one", 20.0, 600.0, 3, false, mediaRevision = "file-r1"))
            val reporter = VideoProgressReporter(scope, now = { 100 }, epochNow = { 1000 }, journal = journal)
            val restored = reporter.restore(credential, VideoPlaybackQueue(listOf(entry), 0))
            assertEquals(20_000L, restored.startPositionMs())
            // Model close winning the race against the asynchronously scheduled replay.
            reporter.submit(credential, restored.entries.single(), restored.startPositionMs(), 600_000, false, false)
            assertTrue("pending progress must be PUT rather than treated as remote baseline", delivered.await(3, TimeUnit.SECONDS))
            assertEquals(20.0, synchronized(requests) { requests.single().getDouble("position_seconds") }, 0.0)
        } finally {
            scope.cancel()
            server.stop(0)
            job.join()
        }
    }
}
