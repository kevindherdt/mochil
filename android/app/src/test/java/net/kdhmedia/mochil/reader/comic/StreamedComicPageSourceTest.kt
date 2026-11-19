package net.kdhmedia.mochil.reader.comic

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.atomic.AtomicInteger
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class StreamedComicPageSourceTest {
    @Test fun `fetches comic pages from the templated stream URL`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        var requestedPath = ""
        server.createContext("/stream/books/manga") { exchange ->
            requestedPath = exchange.requestURI.toString()
            val body = byteArrayOf(1, 2, 3)
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val base = "http://127.0.0.1:${server.address.port}"
            val source = StreamedComicPageSource(base, "/stream/books/manga?page={pageNumber}", pageCount = 5, token = "secret")
            assertArrayEquals(byteArrayOf(1, 2, 3), source.page(2))
            assertEquals("/stream/books/manga?page=2", requestedPath)
        } finally {
            server.stop(0)
        }
    }

    @Test fun `attaches the device bearer token to stream requests`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        var authorization = ""
        server.createContext("/stream/books/manga") { exchange ->
            authorization = exchange.requestHeaders.getFirst("Authorization").orEmpty()
            val body = byteArrayOf(9)
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val base = "http://127.0.0.1:${server.address.port}"
            StreamedComicPageSource(base, "/stream/books/manga?page={pageNumber}", pageCount = 1, token = "device-token").page(0)
            assertEquals("Bearer device-token", authorization)
        } finally {
            server.stop(0)
        }
    }

    @Test fun `no Authorization header is sent when there is no device token`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        var sawAuthorizationHeader = true
        server.createContext("/stream/books/manga") { exchange ->
            sawAuthorizationHeader = exchange.requestHeaders.containsKey("Authorization")
            val body = byteArrayOf(9)
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val base = "http://127.0.0.1:${server.address.port}"
            StreamedComicPageSource(base, "/stream/books/manga?page={pageNumber}", pageCount = 1, token = null).page(0)
            assertEquals(false, sawAuthorizationHeader)
        } finally {
            server.stop(0)
        }
    }

    @Test fun `a transient page fetch failure is retried until it succeeds`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        val attempts = AtomicInteger(0)
        server.createContext("/stream/books/manga") { exchange ->
            if (attempts.getAndIncrement() < 2) {
                exchange.sendResponseHeaders(503, -1)
                exchange.close()
            } else {
                val body = byteArrayOf(7)
                exchange.sendResponseHeaders(200, body.size.toLong())
                exchange.responseBody.use { it.write(body) }
            }
        }
        server.start()
        try {
            val base = "http://127.0.0.1:${server.address.port}"
            val source = StreamedComicPageSource(base, "/stream/books/manga?page={pageNumber}", pageCount = 1, token = null)
            assertArrayEquals(byteArrayOf(7), source.page(0))
            assertEquals(3, attempts.get())
        } finally {
            server.stop(0)
        }
    }

    @Test fun `a persistently failing page fetch throws a descriptive error after exhausting retries`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        val attempts = AtomicInteger(0)
        server.createContext("/stream/books/manga") { exchange ->
            attempts.incrementAndGet()
            exchange.sendResponseHeaders(500, -1)
            exchange.close()
        }
        server.start()
        try {
            val base = "http://127.0.0.1:${server.address.port}"
            val source = StreamedComicPageSource(base, "/stream/books/manga?page={pageNumber}", pageCount = 1, token = null)
            val error = assertThrows(IllegalStateException::class.java) { source.page(0) }
            assertEquals("This comic page could not be loaded.", error.message)
            assertEquals(3, attempts.get())
        } finally {
            server.stop(0)
        }
    }

    @Test fun `rejects an out of range page index`() {
        val source = StreamedComicPageSource("http://127.0.0.1:1", "/stream/books/manga?page={pageNumber}", pageCount = 3, token = null)
        assertThrows(IllegalArgumentException::class.java) { source.page(3) }
        assertThrows(IllegalArgumentException::class.java) { source.page(-1) }
    }
}
