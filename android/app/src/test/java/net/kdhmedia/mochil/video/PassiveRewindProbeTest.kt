package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import kotlinx.coroutines.*
import java.util.concurrent.atomic.AtomicInteger

class PassiveRewindProbeTest {
    @Test fun `recreated reporter does not rewind acknowledged replay`() = runBlocking {
        val puts = AtomicInteger()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                if (exchange.requestMethod == "PUT") puts.incrementAndGet()
                val bytes = """{"revision":8,"position_seconds":12,"finished":false}""".toByteArray()
                exchange.sendResponseHeaders(200, bytes.size.toLong())
                exchange.responseBody.use { it.write(bytes) }
            } finally { exchange.close() }
        }
        server.start()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            var bytes: ByteArray? = null
            fun journal() = VideoProgressJournal(videoProgressOwner(credential), { bytes }, { bytes = it; true })
            val saved = PendingVideoProgress(videoProgressOwner(credential), "anime", "one", 12.0, 600.0, 7, false, mediaRevision = "file-r1", replayFinished = true)
            assertTrue(journal().rebaseNewer(saved, 8))
            assertEquals(8L, journal().put(saved.copy(positionSeconds = 5.0))!!.revision)
            VideoProgressReporter(scope, { 100 }, { 700_000 }, journal()).replay(credential).join()
            withTimeout(3000) { while (journal().entries().isNotEmpty()) delay(10) }
            assertEquals(0, puts.get())
        } finally { scope.cancel(); server.stop(0) }
    }
    @Test fun `rebased snapshot does not rewind acknowledged position`() {
        var position = 12.0
        var requests = 0
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                if (exchange.requestMethod == "PUT") {
                    val body = JSONObject(exchange.requestBody.use { it.readBytes().decodeToString() })
                    assertEquals(8L, body.getLong("expected_revision"))
                    requests++
                    position = body.getDouble("position_seconds")
                }
                val bytes = """{"revision":8,"position_seconds":$position,"finished":false}""".toByteArray()
                exchange.sendResponseHeaders(200, bytes.size.toLong())
                exchange.responseBody.use { it.write(bytes) }
            } finally { exchange.close() }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            val entry = VideoPlaybackEntry("anime", "one", "", "", "", 600_000, 600.0, "", revision = "file-r1", progressRevision = 7, finished = true)
            assertTrue(VideoProgressClient().save(credential, entry, 5.0, 600.0, false, initialRevision = 8))
            assertEquals(12.0, position, 0.0)
            assertEquals(0, requests)
        } finally { server.stop(0) }
    }
}
