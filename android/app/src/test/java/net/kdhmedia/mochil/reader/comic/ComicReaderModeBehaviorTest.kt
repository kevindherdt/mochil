package net.kdhmedia.mochil.reader.comic

import android.graphics.Bitmap
import com.sun.net.httpserver.HttpServer
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.click
import androidx.compose.ui.test.getUnclippedBoundsInRoot
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
import java.io.ByteArrayOutputStream
import java.io.File
import java.net.InetSocketAddress
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.zip.ZipEntry
import java.util.zip.ZipOutputStream

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class ComicReaderModeBehaviorTest {
    @get:Rule val compose = createComposeRule()

    private fun archive(): File {
        val context = RuntimeEnvironment.getApplication()
        context.getSharedPreferences("comic_reader", 0).edit().clear().commit()
        val bitmap = Bitmap.createBitmap(40, 200, Bitmap.Config.ARGB_8888)
        val bytes = ByteArrayOutputStream().also { bitmap.compress(Bitmap.CompressFormat.PNG, 100, it) }.toByteArray()
        return File.createTempFile("comic-mode", ".cbz", context.cacheDir).also { file ->
            ZipOutputStream(file.outputStream()).use { zip ->
                repeat(8) { i -> zip.putNextEntry(ZipEntry("$i.png")); zip.write(bytes); zip.closeEntry() }
            }
        }
    }

    @Test fun switchingModesKeepsCurrentPageAndPersistedProgress() {
        val file = archive()
        val turns = MutableSharedFlow<ReaderPageTurn>(extraBufferCapacity = 1)
        val key = "mode-switch-test"
        val store = FixedLayoutLocalProgressStore(RuntimeEnvironment.getApplication(), FixedLayoutReaderKind.Comic)
        store.save(key, 4)
        var current = -1
        compose.setContent { ComicReaderScreen(ComicSource.Local(file), "Comic", key, onPage = { current = it }, onClose = {}, pageTurns = turns) }
        compose.waitUntil(timeoutMillis = 10_000) {
            runCatching { compose.onNodeWithContentDescription("Page 5").fetchSemanticsNode() }.isSuccess
        }
        assertEquals(4, current)
        compose.onNodeWithContentDescription("Page 5").performTouchInput { click() }
        compose.onNodeWithText("Paged").performClick()
        compose.onNodeWithText("Webtoon").performClick()
        compose.waitForIdle()
        assertEquals(4, current)
        assertEquals(4, store.load(key, 7))
        turns.tryEmit(ReaderPageTurn.Next)
        compose.waitForIdle()
        assertEquals(5, current)
        compose.onNodeWithText("Webtoon").performClick()
        compose.onNodeWithText("Paged").performClick()
        compose.waitForIdle()
        assertEquals(5, current)
        assertEquals(5, store.load(key, 7))
    }

    @Test fun switchingToWebtoonWhilePagesLoadDoesNotOverwriteProgress() {
        val context = RuntimeEnvironment.getApplication()
        context.getSharedPreferences("comic_reader", 0).edit().clear().commit()
        val release = CountDownLatch(1)
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/pages") { exchange ->
            release.await(10, TimeUnit.SECONDS)
            exchange.sendResponseHeaders(500, -1)
            exchange.close()
        }
        server.start()
        try {
            val key = "pending-webtoon-test"
            val store = FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Comic)
            store.save(key, 4)
            val controls = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
            var current = -1
            val source = ComicSource.Streamed("http://127.0.0.1:${server.address.port}", "/pages?page={pageNumber}", 8, null)
            compose.setContent { ComicReaderScreen(source, "Comic", key, onPage = { current = it }, onClose = {}, pageTurns = MutableSharedFlow(), controlRequests = controls) }
            compose.waitForIdle()
            assertEquals(4, current)
            controls.tryEmit(Unit)
            compose.waitForIdle()
            compose.onNodeWithText("Paged").performClick()
            compose.onNodeWithText("Webtoon").performClick()
            compose.waitForIdle()
            assertEquals(4, current)
            assertEquals(4, store.load(key, 7))
        } finally {
            release.countDown()
            server.stop(0)
        }
    }

    @Test fun widthModeLetsTallPageExtendBeyondViewport() {
        val file = archive()
        compose.setContent { ComicReaderScreen(ComicSource.Local(file), "Comic", "width-mode-test", onClose = {}, pageTurns = MutableSharedFlow()) }
        compose.waitUntil(timeoutMillis = 10_000) {
            runCatching { compose.onNodeWithContentDescription("Page 1").fetchSemanticsNode() }.isSuccess
        }
        val fitHeight = compose.onNodeWithContentDescription("Page 1").getUnclippedBoundsInRoot().let { it.bottom - it.top }
        compose.onNodeWithContentDescription("Page 1").performTouchInput { click() }
        compose.onNodeWithText("Fit").performClick()
        compose.onNodeWithText("Width").performClick()
        compose.waitUntil(timeoutMillis = 10_000) {
            runCatching { compose.onNodeWithContentDescription("Page 1").getUnclippedBoundsInRoot().let { it.bottom - it.top } > fitHeight }.getOrDefault(false)
        }
        assertTrue(compose.onNodeWithContentDescription("Page 1").getUnclippedBoundsInRoot().let { it.bottom - it.top } > fitHeight)
    }

    @Test fun aSavedStretchFitOpensAsFitAndOnlyFitAndWidthAreOffered() {
        val file = archive()
        RuntimeEnvironment.getApplication().getSharedPreferences("comic_reader", 0).edit().putString("page_fit", "Stretch").commit()
        compose.setContent { ComicReaderScreen(ComicSource.Local(file), "Comic", "stretch-migration-test", onClose = {}, pageTurns = MutableSharedFlow()) }
        compose.waitUntil(timeoutMillis = 10_000) {
            runCatching { compose.onNodeWithContentDescription("Page 1").fetchSemanticsNode() }.isSuccess
        }
        compose.onNodeWithContentDescription("Page 1").performTouchInput { click() }
        compose.onNodeWithText("Stretch").assertDoesNotExist()
        compose.onNodeWithText("Fit").performClick()
        compose.onNodeWithText("Width").assertExists()
        compose.onNodeWithText("Stretch").assertDoesNotExist()
        assertEquals(listOf("Fit", "Width"), PageFit.entries.map { it.name })
    }
}
