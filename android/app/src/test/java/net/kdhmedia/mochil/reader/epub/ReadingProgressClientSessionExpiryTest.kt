package net.kdhmedia.mochil.reader.epub

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import kotlinx.coroutines.runBlocking
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.DeviceSessionEvents
import org.junit.Assert.assertEquals
import org.junit.Test

class ReadingProgressClientSessionExpiryTest {
    @Test fun `a 401 reading progress expires the device session, other failures do not`() = runBlocking {
        for (status in listOf(401, 500)) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/reading") { exchange -> exchange.sendResponseHeaders(status, -1); exchange.close() }
            server.start()
            try {
                DeviceSessionEvents.connected()
                val before = DeviceSessionEvents.generation.value
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                runCatching { ReadingProgressClient().get(credential, "books", "book") }
                val expectedBump = if (status == 401) before + 1 else before
                assertEquals("status $status", expectedBump, DeviceSessionEvents.generation.value)
            } finally { DeviceSessionEvents.connected(); server.stop(0) }
        }
    }
}
