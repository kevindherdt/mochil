package net.kdhmedia.mochil.podcast

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.DeviceSessionEvents
import org.junit.Assert.assertEquals
import org.junit.Test

class PodcastClientSessionExpiryTest {
    @Test fun `a 401 subscribing to a podcast expires the device session, other failures do not`() {
        for (status in listOf(401, 500)) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/podcast-feeds") { exchange ->
                exchange.requestBody.use { it.readBytes() }
                exchange.sendResponseHeaders(status, -1); exchange.close()
            }
            server.start()
            try {
                DeviceSessionEvents.connected()
                val before = DeviceSessionEvents.generation.value
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                runCatching { PodcastClient().subscribe(credential, "https://feeds.example/show.xml") }
                val expectedBump = if (status == 401) before + 1 else before
                assertEquals("status $status", expectedBump, DeviceSessionEvents.generation.value)
            } finally { DeviceSessionEvents.connected(); server.stop(0) }
        }
    }

    @Test fun `a 401 refreshing episode progress expires the device session, other failures do not`() {
        for (status in listOf(401, 500)) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/audio-progress") { exchange -> exchange.sendResponseHeaders(status, -1); exchange.close() }
            server.start()
            try {
                DeviceSessionEvents.connected()
                val before = DeviceSessionEvents.generation.value
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                val episode = net.kdhmedia.mochil.audio.AudioTrack(id = "episode", libraryId = "podcasts", title = "Episode", artist = "Show", album = "Show", durationSeconds = 100.0)
                runCatching { PodcastClient().latestProgress(credential, episode) }
                val expectedBump = if (status == 401) before + 1 else before
                assertEquals("status $status", expectedBump, DeviceSessionEvents.generation.value)
            } finally { DeviceSessionEvents.connected(); server.stop(0) }
        }
    }
}
