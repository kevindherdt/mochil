package net.kdhmedia.mochil.playback

import com.sun.net.httpserver.HttpServer
import java.io.IOException
import java.net.InetSocketAddress
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.*
import org.junit.Test

class AudioProgressResponseBoundsTest {
    @Test fun `oversized progress responses are not acknowledged and later requests recover`() {
        for (oversizedMethod in listOf("GET", "PUT")) {
            val oversized = AtomicBoolean(true)
            val writes = AtomicInteger()
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/audio-progress/") { exchange ->
                exchange.requestBody.use { it.readBytes() }
                if (exchange.requestMethod == "PUT") writes.incrementAndGet()
                val json = """{"revision":1,"position_seconds":0,"finished":false}""".encodeToByteArray()
                val body = if (oversized.get() && exchange.requestMethod == oversizedMethod) {
                    json + ByteArray(MAX_PLAYBACK_PROGRESS_RESPONSE_BYTES) { 32 }
                } else json
                try {
                    exchange.sendResponseHeaders(200, body.size.toLong())
                    exchange.responseBody.use { it.write(body) }
                } finally { exchange.close() }
            }
            server.start()
            try {
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "fixture")
                val client = AudioProgressClient()
                assertThrows(IOException::class.java) { client.save(credential, "podcasts", "one", 30.0, 600.0) }
                assertEquals(if (oversizedMethod == "GET") 0 else 1, writes.get())
                oversized.set(false)
                assertTrue(client.save(credential, "podcasts", "one", 30.0, 600.0))
                assertEquals(if (oversizedMethod == "GET") 1 else 2, writes.get())
            } finally { server.stop(0) }
        }
    }
}
