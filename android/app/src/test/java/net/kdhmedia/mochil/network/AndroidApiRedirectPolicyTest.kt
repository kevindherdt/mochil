package net.kdhmedia.mochil.network

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.net.URI
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AndroidApiRedirectPolicyTest {
    @Test fun `safe redirect keeps bearer token inside Android API origin`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        var authorization = ""
        server.createContext("/api/android/v1/start") { exchange ->
            exchange.responseHeaders.add("Location", "/api/android/v1/final")
            exchange.sendResponseHeaders(303, -1)
            exchange.close()
        }
        server.createContext("/api/android/v1/final") { exchange ->
            authorization = exchange.requestHeaders.getFirst("Authorization").orEmpty()
            exchange.sendResponseHeaders(204, -1)
            exchange.close()
        }
        server.start()
        try {
            val base = "http://127.0.0.1:${server.address.port}"
            val connection = openAndroidApiGet(DeviceCredential(base, "device", "secret-token"), "/api/android/v1/start")
            try { assertEquals(204, connection.responseCode) } finally { connection.disconnect() }
            assertEquals("Bearer secret-token", authorization)
        } finally { server.stop(0) }
    }

    @Test fun `redirect policy rejects login pages and credential exfiltration`() {
        val api = URI("https://books.example/api/android/v1/documents?library=wiki")
        assertFalse(isSafeAndroidApiRedirect(api, URI("https://books.example/login")))
        assertFalse(isSafeAndroidApiRedirect(api, URI("https://attacker.example/api/android/v1/documents")))
        assertFalse(isSafeAndroidApiRedirect(api, URI("http://books.example/api/android/v1/documents")))
        assertTrue(isSafeAndroidApiRedirect(api, URI("https://books.example/api/android/v1/documents")))
    }

    @Test fun `a 401 from the shared GET path expires every client's device session`() {
        for (status in listOf(401, 403, 404, 500)) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/whoami") { exchange -> exchange.sendResponseHeaders(status, -1); exchange.close() }
            server.start()
            try {
                net.kdhmedia.mochil.auth.DeviceSessionEvents.connected()
                val before = net.kdhmedia.mochil.auth.DeviceSessionEvents.generation.value
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                val connection = openAndroidApiGet(credential, "/api/android/v1/whoami")
                try { connection.responseCode } finally { connection.disconnect() }
                val expectedBump = if (status == 401) before + 1 else before
                assertEquals("status $status", expectedBump, net.kdhmedia.mochil.auth.DeviceSessionEvents.generation.value)
            } finally { net.kdhmedia.mochil.auth.DeviceSessionEvents.connected(); server.stop(0) }
        }
    }
}
