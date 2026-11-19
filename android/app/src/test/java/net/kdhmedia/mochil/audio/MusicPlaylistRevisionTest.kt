package net.kdhmedia.mochil.audio

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicReference
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.*
import org.junit.Test

class MusicPlaylistRevisionTest {
    @Test fun `client carries revision and surfaces stale replacement without retry`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        val puts = AtomicInteger()
        val received = AtomicReference<String>()
        server.createContext("/api/android/v1/audio-playlists/Saved") { exchange ->
            if (exchange.requestMethod == "GET") {
                val body = """{"name":"Saved","items":[]}""".encodeToByteArray()
                exchange.responseHeaders.set("ETag", "\"initial\"")
                exchange.responseHeaders.set("X-Mochil-Playlist-Edit", "visible-v1")
                exchange.sendResponseHeaders(200, body.size.toLong())
                exchange.responseBody.use { it.write(body) }
            } else {
                puts.incrementAndGet()
                received.set(exchange.requestHeaders.getFirst("If-Match"))
                exchange.requestBody.use { it.readBytes() }
                exchange.sendResponseHeaders(412, -1)
                exchange.close()
            }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val client = MusicPlaylistClient()
            val playlist = client.get(credential, "Saved")
            assertEquals("\"initial\"", playlist.revision)
            assertTrue(playlist.visibleEdit)
            assertThrows(MusicPlaylistConflict::class.java) {
                client.put(credential, playlist.name, playlist.tracks, playlist.revision)
            }
            assertEquals("\"initial\"", received.get())
            assertEquals(1, puts.get())
        } finally { server.stop(0) }
    }
}
