package net.kdhmedia.mochil.wiki

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.DeviceSessionEvents
import org.junit.Assert.assertEquals
import org.junit.Test

class WikiClientSessionExpiryTest {
    @Test fun `a 401 reading or saving a wiki page expires the device session, other failures do not`() {
        for (status in listOf(401, 500)) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/documents") { exchange -> exchange.sendResponseHeaders(status, -1); exchange.close() }
            server.createContext("/api/android/v1/vaults") { exchange ->
                exchange.requestBody.use { it.readBytes() }
                exchange.sendResponseHeaders(status, -1); exchange.close()
            }
            server.start()
            try {
                DeviceSessionEvents.connected()
                val before = DeviceSessionEvents.generation.value
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                val client = WikiClient()
                runCatching { client.tree(credential, "wiki") }
                val expectedBump = if (status == 401) before + 1 else before
                assertEquals("tree status $status", expectedBump, DeviceSessionEvents.generation.value)
                DeviceSessionEvents.connected()

                val beforeSave = DeviceSessionEvents.generation.value
                runCatching { client.save(credential, "wiki", WikiPage(id = "page", title = "Page", content = "", revision = "r1", writable = true), "content") }
                val expectedSaveBump = if (status == 401) beforeSave + 1 else beforeSave
                assertEquals("save status $status", expectedSaveBump, DeviceSessionEvents.generation.value)
            } finally { DeviceSessionEvents.connected(); server.stop(0) }
        }
    }
}
