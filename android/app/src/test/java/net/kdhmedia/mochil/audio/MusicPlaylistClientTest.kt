package net.kdhmedia.mochil.audio

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.*
import org.junit.Test

class MusicPlaylistClientTest {
    @Test fun `detail uses one encoded request and preserves ordered tracks`() {
        withServer(200, """{"name":"Road / 100%25","items":[{"id":"two","library_id":"music"},{"id":"one","library_id":"music"}]}""") { credential, requests ->
            val result = MusicPlaylistClient().get(credential, "Road / 100%25")
            assertEquals(listOf("two", "one"), result.tracks.map { it.id })
            assertEquals(listOf("/api/android/v1/audio-playlists/Road%20%2F%20100%2525"), requests)
        }
    }

    @Test fun `older server method rejection falls back to collection`() {
        withServer(405, "unsupported") { credential, requests ->
            assertEquals("Saved", MusicPlaylistClient().get(credential, "saved").name)
            assertEquals(2, requests.size)
            assertEquals("/api/android/v1/audio-playlists", requests.last())
        }
    }

    @Test fun `missing unauthorized redirect and server errors never fall back`() {
        for (status in listOf(404, 401, 403, 303, 500)) {
            withServer(status, "failure") { credential, requests ->
                assertTrue(runCatching { MusicPlaylistClient().get(credential, "Saved") }.isFailure)
                assertEquals("status $status", 1, requests.size)
            }
        }
    }

    @Test fun `incomplete or mismatched detail cannot become replacement tracks`() {
        for (body in listOf("{}", """{"name":"Saved"}""", """{"name":"Saved","items":null}""", """{"name":"Other","items":[]}""", "<html>Sign in</html>")) {
            withServer(200, body) { credential, requests ->
                assertTrue(runCatching { MusicPlaylistClient().get(credential, "Saved") }.isFailure)
                assertEquals(1, requests.size)
            }
        }
    }

    @Test fun `existing empty detail remains a valid playlist`() {
        withServer(200, """{"name":"Saved","items":[]}""") { credential, _ ->
            assertTrue(MusicPlaylistClient().get(credential, "Saved").tracks.isEmpty())
        }
    }

    @Test fun `a 401 fetching or mutating a playlist expires the device session, other failures do not`() {
        for (status in listOf(401, 500)) {
            withServer(status, "failure") { credential, _ ->
                net.kdhmedia.mochil.auth.DeviceSessionEvents.connected()
                val before = net.kdhmedia.mochil.auth.DeviceSessionEvents.generation.value
                try {
                    runCatching { MusicPlaylistClient().get(credential, "Saved") }
                    val expectedBump = if (status == 401) before + 1 else before
                    assertEquals("status $status", expectedBump, net.kdhmedia.mochil.auth.DeviceSessionEvents.generation.value)
                } finally { net.kdhmedia.mochil.auth.DeviceSessionEvents.connected() }
            }
        }
    }

    private fun withServer(status: Int, detail: String, test: (DeviceCredential, MutableList<String>) -> Unit) {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        val requests = java.util.Collections.synchronizedList(mutableListOf<String>())
        server.createContext("/api/android/v1/audio-playlists") { exchange ->
            requests.add(exchange.requestURI.rawPath)
            val collection = exchange.requestURI.path == "/api/android/v1/audio-playlists"
            val body = (if (collection) """[{"name":"Saved","items":[]}]""" else detail).encodeToByteArray()
            exchange.sendResponseHeaders(if (collection) 200 else status, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try { test(DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token"), requests) }
        finally { server.stop(0) }
    }
}
