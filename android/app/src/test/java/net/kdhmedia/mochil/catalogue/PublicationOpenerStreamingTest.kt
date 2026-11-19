package net.kdhmedia.mochil.catalogue

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.runBlocking
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.reader.comic.ComicReaderActivity
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class PublicationOpenerStreamingTest {
    @Test fun `a comic with a page stream URL opens via streaming without downloading the archive`() {
        val requestedPaths = CopyOnWriteArrayList<String>()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            requestedPaths.add(exchange.requestURI.path)
            val bytes = ByteArray(16)
            exchange.sendResponseHeaders(200, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val book = BookSummary(
                id = "one", libraryId = "manga", title = "One", authors = emptyList(), series = "", seriesIndex = null,
                format = "cbz", mediaType = "application/vnd.comicbook+zip", language = "", description = "", size = 5_000_000L,
                pageStreamURL = "/stream/manga/one?page={pageNumber}", pageCount = 12,
            )
            val opener = PublicationOpener(RuntimeEnvironment.getApplication())
            val intent = runBlocking { opener.prepare(credential, book) }

            assertEquals(ComicReaderActivity::class.java.name, intent.component?.className)
            // The reader must stream through the Bearer-gated Android publications route, never
            // the server's raw page_stream_url value: that field only gates /stream/, which only
            // accepts a session cookie or HTTP Basic auth and always 401s a device Bearer token.
            assertEquals(
                "/api/android/v1/publications/manga/one/pages/{pageNumber}",
                intent.getStringExtra(ComicReaderActivity.EXTRA_STREAM_URL),
            )
            assertEquals(12, intent.getIntExtra(ComicReaderActivity.EXTRA_PAGE_COUNT, -1))
            assertNull(intent.getStringExtra(ComicReaderActivity.EXTRA_PATH))
            assertTrue("no bytes should be downloaded for a streamable comic", requestedPaths.isEmpty())
        } finally {
            server.stop(0)
        }
    }

    @Test fun `the streamed reader fetches pages from the Bearer-gated Android route with the device token`() {
        var requestedPath = ""
        var authorization = ""
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/publications/manga/one/pages/0") { exchange ->
            requestedPath = exchange.requestURI.path
            authorization = exchange.requestHeaders.getFirst("Authorization").orEmpty()
            val body = byteArrayOf(1, 2, 3)
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        // The legacy session/Basic-only stream path must never be hit by the streaming reader.
        server.createContext("/stream/manga/one") { exchange ->
            exchange.sendResponseHeaders(401, -1)
            exchange.close()
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "device-token")
            val book = BookSummary(
                id = "one", libraryId = "manga", title = "One", authors = emptyList(), series = "", seriesIndex = null,
                format = "cbz", mediaType = "application/vnd.comicbook+zip", language = "", description = "", size = 5_000_000L,
                pageStreamURL = "/stream/manga/one?page={pageNumber}", pageCount = 12,
            )
            val opener = PublicationOpener(RuntimeEnvironment.getApplication())
            val intent = runBlocking { opener.prepare(credential, book) }
            val streamURL = requireNotNull(intent.getStringExtra(ComicReaderActivity.EXTRA_STREAM_URL))

            val source = net.kdhmedia.mochil.reader.comic.StreamedComicPageSource(
                credential.serverUrl, streamURL, intent.getIntExtra(ComicReaderActivity.EXTRA_PAGE_COUNT, -1), credential.token,
            )
            val page = source.page(0)

            assertArrayEquals(byteArrayOf(1, 2, 3), page)
            assertEquals("/api/android/v1/publications/manga/one/pages/0", requestedPath)
            assertEquals("Bearer device-token", authorization)
        } finally {
            server.stop(0)
        }
    }

    @Test fun `a comic without a page stream URL falls back to downloading the whole archive`() {
        val requestedPaths = CopyOnWriteArrayList<String>()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            requestedPaths.add(exchange.requestURI.path)
            val bytes = ByteArray(16)
            exchange.sendResponseHeaders(200, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val book = BookSummary(
                id = "two", libraryId = "manga", title = "Two", authors = emptyList(), series = "", seriesIndex = null,
                format = "cbz", mediaType = "application/vnd.comicbook+zip", language = "", description = "", size = 16L,
            )
            val context = RuntimeEnvironment.getApplication()
            val opener = PublicationOpener(context)
            val intent = runBlocking { opener.prepare(credential, book) }

            assertEquals(ComicReaderActivity::class.java.name, intent.component?.className)
            assertNull(intent.getStringExtra(ComicReaderActivity.EXTRA_STREAM_URL))
            assertTrue(requireNotNull(intent.getStringExtra(ComicReaderActivity.EXTRA_PATH)).isNotBlank())
            assertTrue("the whole archive should have been downloaded", requestedPaths.isNotEmpty())
        } finally {
            server.stop(0)
            java.io.File(RuntimeEnvironment.getApplication().filesDir, "publications").deleteRecursively()
        }
    }
}
