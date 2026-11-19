package net.kdhmedia.mochil.reader.epub

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import kotlinx.coroutines.runBlocking
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Test

class AnnotationClientRedirectTest {
    @Test fun `annotation read follows safe redirect while changes stay PUT`() = runBlocking {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        var mutationMethod = ""
        val response = """{"revision":1,"applied":[],"conflicts":[]}""".encodeToByteArray()
        server.createContext("/api/android/v1/annotations/books/book") { exchange ->
            if (exchange.requestMethod == "GET") {
                exchange.responseHeaders.add("Location", "/api/android/v1/canonical-annotations")
                exchange.sendResponseHeaders(303, -1); exchange.close()
            } else {
                mutationMethod = exchange.requestMethod
                exchange.requestBody.use { it.readBytes() }
                exchange.sendResponseHeaders(200, response.size.toLong()); exchange.responseBody.use { it.write(response) }
            }
        }
        server.createContext("/api/android/v1/canonical-annotations") { exchange ->
            exchange.sendResponseHeaders(200, response.size.toLong()); exchange.responseBody.use { it.write(response) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val client = AnnotationClient()
            assertEquals(1, client.changes(credential, "books", "book", 0).revision)
            client.apply(credential, "books", "book", listOf(AnnotationChange(0, SyncedAnnotation("highlight", "highlight"))))
            assertEquals("PUT", mutationMethod)
        } finally { server.stop(0) }
    }

    @Test fun `a 401 synchronizing annotations expires the device session, other failures do not`() = runBlocking {
        for (status in listOf(401, 500)) {
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/annotations/books/book") { exchange ->
                exchange.requestBody.use { it.readBytes() }
                exchange.sendResponseHeaders(status, -1); exchange.close()
            }
            server.start()
            try {
                net.kdhmedia.mochil.auth.DeviceSessionEvents.connected()
                val before = net.kdhmedia.mochil.auth.DeviceSessionEvents.generation.value
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                runCatching { AnnotationClient().changes(credential, "books", "book", 0) }
                val expectedBump = if (status == 401) before + 1 else before
                assertEquals("status $status", expectedBump, net.kdhmedia.mochil.auth.DeviceSessionEvents.generation.value)
            } finally { net.kdhmedia.mochil.auth.DeviceSessionEvents.connected(); server.stop(0) }
        }
    }
}
