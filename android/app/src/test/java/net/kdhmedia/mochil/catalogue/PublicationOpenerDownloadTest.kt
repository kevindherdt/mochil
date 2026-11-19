package net.kdhmedia.mochil.catalogue

import com.sun.net.httpserver.HttpServer
import java.io.File
import java.net.InetSocketAddress
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import kotlinx.coroutines.runBlocking
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.reader.comic.ComicReaderActivity
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class PublicationOpenerDownloadTest {
    private fun book(id: String, stream: Boolean = false) = BookSummary(
        id = id, libraryId = "download-test", title = id, authors = emptyList(), series = "", seriesIndex = null,
        format = "cbz", mediaType = "application/vnd.comicbook+zip", language = "", description = "", size = 4,
        pageStreamURL = if (stream) "/stream/test" else null,
    )

    @Test fun `offline preparation downloads and records a streamable comic`() {
        val bytes = byteArrayOf(1, 2, 3, 4)
        val requests = AtomicInteger()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            requests.incrementAndGet()
            exchange.sendResponseHeaders(200, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        val context = RuntimeEnvironment.getApplication()
        val item = book("offline-streamable", stream = true)
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val intent = runBlocking { PublicationOpener(context).prepare(credential, item, forceDownload = true) }
            val path = requireNotNull(intent.getStringExtra(ComicReaderActivity.EXTRA_PATH))
            assertNull(intent.getStringExtra(ComicReaderActivity.EXTRA_STREAM_URL))
            assertArrayEquals(bytes, File(path).readBytes())
            assertNotNull(PublicationDownloadStore(context).find(item.libraryId, item.id))
            assertEquals(1, requests.get())
        } finally {
            server.stop(0)
            File(context.filesDir, "publications").deleteRecursively()
        }
    }

    @Test fun `simultaneous read and offline preparation share one complete download`() {
        val bytes = byteArrayOf(5, 6, 7, 8)
        val requests = AtomicInteger()
        val firstRequest = CountDownLatch(1)
        val release = CountDownLatch(1)
        val secondStarted = CountDownLatch(1)
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        val serverPool = Executors.newCachedThreadPool()
        val workers = Executors.newFixedThreadPool(2)
        server.executor = serverPool
        server.createContext("/") { exchange ->
            requests.incrementAndGet()
            firstRequest.countDown()
            release.await(5, TimeUnit.SECONDS)
            exchange.sendResponseHeaders(200, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        val context = RuntimeEnvironment.getApplication()
        val item = book("concurrent")
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val first = workers.submit<android.content.Intent> { runBlocking { PublicationOpener(context).prepare(credential, item) } }
            assertTrue(firstRequest.await(5, TimeUnit.SECONDS))
            val second = workers.submit<android.content.Intent> {
                secondStarted.countDown()
                runBlocking { PublicationOpener(context).prepare(credential, item, forceDownload = true) }
            }
            assertTrue(secondStarted.await(5, TimeUnit.SECONDS))
            Thread.sleep(500)
            release.countDown()
            val firstPath = requireNotNull(first.get(10, TimeUnit.SECONDS).getStringExtra(ComicReaderActivity.EXTRA_PATH))
            val secondPath = requireNotNull(second.get(10, TimeUnit.SECONDS).getStringExtra(ComicReaderActivity.EXTRA_PATH))
            assertEquals(firstPath, secondPath)
            assertArrayEquals(bytes, File(firstPath).readBytes())
            assertEquals(1, requests.get())
        } finally {
            release.countDown()
            workers.shutdownNow()
            server.stop(0)
            serverPool.shutdownNow()
            File(context.filesDir, "publications").deleteRecursively()
        }
    }
}
