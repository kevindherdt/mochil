package net.kdhmedia.mochil.playback

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class AudioProgressRedirectTest {
    @Test fun `progress read follows safe redirect while revision save stays PUT`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        var mutationMethod = ""
        server.createContext("/api/android/v1/audio-progress/music/track") { exchange ->
            if (exchange.requestMethod == "GET") {
                exchange.responseHeaders.add("Location", "/api/android/v1/canonical-audio-progress")
                exchange.sendResponseHeaders(303, -1); exchange.close()
            } else {
                mutationMethod = exchange.requestMethod
                exchange.requestBody.use { it.readBytes() }
                val body = """{"revision":1,"position_seconds":30,"finished":false}""".encodeToByteArray()
                exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
            }
        }
        server.createContext("/api/android/v1/canonical-audio-progress") { exchange ->
            val body = """{"revision":0,"position_seconds":0,"finished":false}""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            assertTrue(AudioProgressClient().save(credential, "music", "track", 30.0, 120.0))
            assertEquals("PUT", mutationMethod)
        } finally { server.stop(0) }
    }
}
