package net.kdhmedia.mochil.wiki

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Test

class WikiClientRedirectTest {
    @Test fun `vault image asset accepts a bounded image response`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        val bytes = byteArrayOf(0x89.toByte(), 0x50, 0x4e, 0x47)
        server.createContext("/api/android/v1/document-assets/wiki/image/cover.png") { exchange ->
            exchange.responseHeaders.add("Content-Type", "image/png")
            exchange.sendResponseHeaders(200, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val entry = WikiEntry("page", "image", "Assets/cover.png", "cover", "png")
            assertEquals(bytes.toList(), WikiClient().asset(credential, "wiki", entry).toList())
        } finally { server.stop(0) }
    }

    @Test fun `vault tree follows a safe canonical API redirect`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/documents") { exchange ->
            exchange.responseHeaders.add("Location", "/api/android/v1/canonical-documents")
            exchange.sendResponseHeaders(303, -1)
            exchange.close()
        }
        server.createContext("/api/android/v1/canonical-documents") { exchange ->
            val body = """[{"kind":"page","id":"home","path":"Home.md","title":"Home","format":"md"}]""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            assertEquals(listOf("Home.md"), WikiClient().tree(credential, "wiki").map(WikiEntry::path))
        } finally {
            server.stop(0)
        }
    }

    @Test fun `vault mutation remains a POST`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        var method = ""
        server.createContext("/api/android/v1/vaults/wiki/folders") { exchange ->
            method = exchange.requestMethod
            exchange.requestBody.use { it.readBytes() }
            exchange.sendResponseHeaders(204, -1)
            exchange.close()
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            WikiClient().createFolder(credential, "wiki", "Notes")
            assertEquals("POST", method)
        } finally {
            server.stop(0)
        }
    }
}
