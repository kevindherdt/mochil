package net.kdhmedia.mochil.audio

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.*
import org.junit.Test

class MusicQueueClientTest {
    @Test fun `only a successful array response can establish an empty saved queue`() {
        for ((status, body) in listOf(404 to "[]", 503 to "[]", 200 to "null", 200 to "[]")) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/audio-queue") { exchange ->
                val bytes = body.encodeToByteArray()
                exchange.sendResponseHeaders(status, bytes.size.toLong())
                exchange.responseBody.use { it.write(bytes) }
            }
            server.start()
            try {
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                val result = runCatching { MusicQueueClient().get(credential) }
                assertEquals("status=$status body=$body", status == 200 && body == "[]", result.isSuccess)
            } finally { server.stop(0) }
        }
    }

    @Test fun `a 401 reading or saving the queue expires the device session, other failures do not`() {
        for (status in listOf(401, 500)) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/audio-queue") { exchange ->
                exchange.requestBody.use { it.readBytes() }
                exchange.sendResponseHeaders(status, -1); exchange.close()
            }
            server.start()
            try {
                net.kdhmedia.mochil.auth.DeviceSessionEvents.connected()
                val before = net.kdhmedia.mochil.auth.DeviceSessionEvents.generation.value
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                runCatching { MusicQueueClient().get(credential) }
                val expectedBump = if (status == 401) before + 1 else before
                assertEquals("status $status", expectedBump, net.kdhmedia.mochil.auth.DeviceSessionEvents.generation.value)
            } finally { net.kdhmedia.mochil.auth.DeviceSessionEvents.connected(); server.stop(0) }
        }
    }
}
