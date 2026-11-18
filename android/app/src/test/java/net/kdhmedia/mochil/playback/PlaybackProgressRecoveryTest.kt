package net.kdhmedia.mochil.playback

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.Collections
import java.util.concurrent.atomic.AtomicInteger
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.video.VideoPlaybackEntry
import net.kdhmedia.mochil.video.VideoProgressClient
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

/** Exercise actual HTTP clients, not just queue callbacks or ownership policy. */
class PlaybackProgressRecoveryTest {
    @Test fun `audio resumes after failed mutation without advancing its revision`() = verifyRecovery(video = false)

    @Test fun `video resumes after failed mutation without advancing its revision`() = verifyRecovery(video = true)

    private fun verifyRecovery(video: Boolean) {
        val revisions = Collections.synchronizedList(mutableListOf<Long>())
        val attempts = AtomicInteger()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/") { exchange ->
            try {
                var position = 0.0
                var revision = 0L
                if (exchange.requestMethod == "PUT") {
                    val request = JSONObject(exchange.requestBody.use { it.readBytes().decodeToString() })
                    revision = request.getLong("expected_revision")
                    revisions += revision
                    if (attempts.incrementAndGet() == 1) {
                        exchange.sendResponseHeaders(503, -1)
                        return@createContext
                    }
                    position = request.getDouble("position_seconds")
                    revision++
                }
                val body = """{"revision":$revision,"position_seconds":$position,"finished":false}""".encodeToByteArray()
                exchange.sendResponseHeaders(200, body.size.toLong())
                exchange.responseBody.use { it.write(body) }
            } finally { exchange.close() }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "fixture-device", "fixture-token")
            val audio = AudioProgressClient()
            val movie = VideoProgressClient()
            val entry = VideoPlaybackEntry("anime", "episode", "Episode", "unused", "", 0, 600.0, "")
            fun save(position: Double) = if (video) movie.save(credential, entry, position, 600.0, false)
                else audio.save(credential, "podcasts", "episode", position, 600.0)

            assertTrue("Failed network mutation must not be acknowledged", runCatching { save(30.0) }.isFailure)
            assertTrue(save(40.0))
            assertTrue(save(50.0))
            assertEquals(listOf(0L, 0L, 1L), revisions.toList())
        } finally { server.stop(0) }
    }
}
