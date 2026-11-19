package net.kdhmedia.mochil.reader.comic

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import kotlinx.coroutines.flow.MutableSharedFlow
import net.kdhmedia.mochil.reader.FixedLayoutLocalProgressStore
import net.kdhmedia.mochil.reader.FixedLayoutReaderKind
import net.kdhmedia.mochil.reader.ReaderPageTurn
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class ComicReaderScreenStreamingBehaviorTest {
    @get:Rule val compose = createComposeRule()

    @Test fun `a streamed comic persists reading position under the same key as a downloaded comic`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/stream/manga/one") { exchange ->
            val body = byteArrayOf(1, 2, 3, 4)
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val source = ComicSource.Streamed(
                serverUrl = "http://127.0.0.1:${server.address.port}",
                pageStreamURL = "/stream/manga/one?page={pageNumber}",
                pageCount = 5,
                token = "device-token",
            )
            val pageTurns = MutableSharedFlow<ReaderPageTurn>(extraBufferCapacity = 1)
            val progressKey = "manga:one"
            var latestPage = -1
            compose.setContent {
                ComicReaderScreen(
                    source = source,
                    title = "One",
                    progressKey = progressKey,
                    onPage = { latestPage = it },
                    onClose = {},
                    pageTurns = pageTurns,
                )
            }
            compose.waitForIdle()
            pageTurns.tryEmit(ReaderPageTurn.Next)
            compose.waitForIdle()

            assertEquals(1, latestPage)
            val store = FixedLayoutLocalProgressStore(RuntimeEnvironment.getApplication(), FixedLayoutReaderKind.Comic)
            // The same store, keyed the same way, is used regardless of whether the comic was
            // streamed or downloaded, so a previously downloaded reading position still applies.
            assertEquals(1, store.load(progressKey, maximumPage = 4))
        } finally {
            server.stop(0)
        }
    }

    @Test fun `a persistently failing streamed page shows a retry control that re-fetches on tap`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        val healthy = AtomicBoolean(false)
        val requests = AtomicInteger(0)
        server.createContext("/stream/manga/two") { exchange ->
            requests.incrementAndGet()
            if (healthy.get()) {
                val body = byteArrayOf(5, 6, 7)
                exchange.sendResponseHeaders(200, body.size.toLong())
                exchange.responseBody.use { it.write(body) }
            } else {
                exchange.sendResponseHeaders(500, -1)
                exchange.close()
            }
        }
        server.start()
        try {
            val source = ComicSource.Streamed(
                serverUrl = "http://127.0.0.1:${server.address.port}",
                pageStreamURL = "/stream/manga/two?page={pageNumber}",
                pageCount = 1,
                token = null,
            )
            compose.setContent {
                ComicReaderScreen(
                    source = source,
                    title = "Two",
                    progressKey = "manga:two",
                    onClose = {},
                    pageTurns = MutableSharedFlow(),
                )
            }
            compose.waitForIdle()
            compose.onNodeWithText("This page could not be loaded.").assertIsDisplayed()

            healthy.set(true)
            compose.onNodeWithText("Retry").performClick()

            // The retry fetches on Dispatchers.IO, which waitForIdle does not wait for: without this
            // the assertion ran before the answer had arrived and failed every time.
            compose.waitUntil(timeoutMillis = 10_000) {
                compose.onAllNodesWithTextOrEmpty("This page could not be loaded.")
            }

            assertTrue("the retry must actually ask the server again", requests.get() >= 2)
        } finally {
            server.stop(0)
        }
    }

    private fun androidx.compose.ui.test.junit4.ComposeContentTestRule.onAllNodesWithTextOrEmpty(text: String): Boolean =
        onAllNodesWithText(text).fetchSemanticsNodes().isEmpty()
}
