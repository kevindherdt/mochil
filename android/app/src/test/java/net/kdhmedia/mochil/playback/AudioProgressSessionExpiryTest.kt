package net.kdhmedia.mochil.playback

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.DeviceSessionEvents
import org.junit.Assert.assertEquals
import org.junit.Test

class AudioProgressSessionExpiryTest {
    @Test fun `a 401 saving audio progress expires the device session, other failures do not`() {
        for (status in listOf(401, 500)) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/audio-progress") { exchange ->
                if (exchange.requestMethod == "GET") {
                    val body = """{"revision":1,"position_seconds":0,"finished":false}""".encodeToByteArray()
                    exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
                } else {
                    exchange.requestBody.use { it.readBytes() }
                    exchange.sendResponseHeaders(status, -1); exchange.close()
                }
            }
            server.start()
            try {
                DeviceSessionEvents.connected()
                val before = DeviceSessionEvents.generation.value
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                runCatching { AudioProgressClient().save(credential, "books", "item", 10.0, 100.0) }
                val expectedBump = if (status == 401) before + 1 else before
                assertEquals("status $status", expectedBump, DeviceSessionEvents.generation.value)
            } finally { DeviceSessionEvents.connected(); server.stop(0) }
        }
    }
}
