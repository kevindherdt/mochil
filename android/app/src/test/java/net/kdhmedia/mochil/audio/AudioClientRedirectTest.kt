package net.kdhmedia.mochil.audio

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Test

class AudioClientRedirectTest {
    @Test fun `group browse follows a safe canonical API redirect`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/audio-groups") { exchange ->
            exchange.responseHeaders.add("Location", "/api/android/v1/canonical-audio-groups")
            exchange.sendResponseHeaders(303, -1)
            exchange.close()
        }
        server.createContext("/api/android/v1/canonical-audio-groups") { exchange ->
            val body = """[{"id":"series","name":"Series","count":2}]""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val page = AudioClient().groups(credential, "audiobooks", MusicGrouping.Series, 0)
            assertEquals(listOf("Series"), page.items.map(AudioGroup::name))
        } finally { server.stop(0) }
    }

    @Test fun `favorite mutation remains a PUT and never preconnects as GET`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        var method = ""
        server.createContext("/api/android/v1/audio-favorites") { exchange ->
            method = exchange.requestMethod
            exchange.requestBody.use { it.readBytes() }
            exchange.sendResponseHeaders(204, -1)
            exchange.close()
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            AudioClient().setFavorite(credential, AudioTrack("track", "music", "Track", "Artist", "Album", 60.0), true)
            assertEquals("PUT", method)
        } finally { server.stop(0) }
    }
}
