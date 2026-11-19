package net.kdhmedia.mochil.dashboard

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.catalogue.toBookSummary
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test
import org.json.JSONObject

class DashboardClientTest {
    @Test fun `home retries after a transient 404`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        var requests = 0
        server.createContext("/api/android/v1/home") { exchange ->
            requests++
            if (requests == 1) { exchange.sendResponseHeaders(404, -1); exchange.close() }
            else {
                val body = """{"reading":[],"listening":[],"watching":[]}""".toByteArray()
                exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
            }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val client = DashboardClient()
            assertEquals(null, client.home(credential))
            org.junit.Assert.assertNotNull(client.home(credential))
            assertEquals(2, requests)
        } finally { server.stop(0) }
    }

    @Test fun `combined home snapshot remains bounded and parses every lane`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/home") { exchange ->
            val body = """{
                "reading":[{"id":"book","library_id":"books","title":"Book","authors":[],"fraction":0.5}],
                "listening":[{"id":"chapter","library_id":"audio","library_kind":"audiobooks","title":"Chapter","duration_seconds":100,"progress":{"position_seconds":25,"duration_seconds":100}}],
                "watching":[{"id":"episode","library_id":"tv","title":"Episode","duration_seconds":200,"stream_url":"/videos/tv/episode","progress":{"position_seconds":50,"duration_seconds":200}}]
            }""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val home = requireNotNull(DashboardClient().home(credential))
            assertEquals("Book", home.reading.single().book.title)
            assertEquals(25.0, home.listening.single().positionSeconds, 0.0)
            assertEquals("Episode", home.watching.single().title)
        } finally { server.stop(0) }
    }

    @Test fun `home reading follows safe Android API redirect`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/reading") { exchange ->
            exchange.responseHeaders.add("Location", "/api/android/v1/canonical-reading")
            exchange.sendResponseHeaders(303, -1); exchange.close()
        }
        server.createContext("/api/android/v1/canonical-reading") { exchange ->
            val body = """[{"id":"book","library_id":"books","title":"Book","authors":[],"fraction":0.5}]""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            assertEquals(listOf("Book"), DashboardClient().reading(credential).map { it.book.title })
        } finally { server.stop(0) }
    }

    @Test fun `continue listening dismissal contains status only`() {
        val body = JSONObject(listeningDismissalBody(true))

        assertEquals(setOf("dismissed"), body.keys().asSequence().toSet())
        assertEquals(true, body.getBoolean("dismissed"))
        assertFalse(body.has("position_seconds"))
        assertFalse(body.has("duration_seconds"))
    }

    @Test fun `a 401 dismissing reading or listening expires the device session, other failures do not`() {
        for (status in listOf(401, 500)) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/reading") { exchange -> exchange.sendResponseHeaders(status, -1); exchange.close() }
            server.createContext("/api/android/v1/audio-progress") { exchange ->
                exchange.requestBody.use { it.readBytes() }
                exchange.sendResponseHeaders(status, -1); exchange.close()
            }
            server.start()
            try {
                net.kdhmedia.mochil.auth.DeviceSessionEvents.connected()
                val before = net.kdhmedia.mochil.auth.DeviceSessionEvents.generation.value
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                val book = JSONObject().put("id", "book").put("library_id", "books").toBookSummary()
                runCatching { DashboardClient().dismissReading(credential, book) }
                val expectedAfterFirst = if (status == 401) before + 1 else before
                assertEquals("dismissReading status $status", expectedAfterFirst, net.kdhmedia.mochil.auth.DeviceSessionEvents.generation.value)
            } finally { net.kdhmedia.mochil.auth.DeviceSessionEvents.connected(); server.stop(0) }
        }
    }
}
