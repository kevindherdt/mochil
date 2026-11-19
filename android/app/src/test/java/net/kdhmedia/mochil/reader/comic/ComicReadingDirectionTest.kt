package net.kdhmedia.mochil.reader.comic

import android.graphics.Bitmap
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.click
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTouchInput
import kotlinx.coroutines.flow.MutableSharedFlow
import net.kdhmedia.mochil.reader.ReaderPageTurn
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import java.io.ByteArrayOutputStream
import java.io.File
import java.util.zip.ZipEntry
import java.util.zip.ZipOutputStream

/** Manga opens right-to-left and other comics left-to-right (the web's "Manga default"); a saved choice always wins. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class ComicReadingDirectionTest {
    @get:Rule val compose = createComposeRule()
    private val context get() = RuntimeEnvironment.getApplication()
    private val prefs get() = context.getSharedPreferences("comic_reader", 0)

    private fun archive(): File {
        prefs.edit().clear().commit()
        val bitmap = Bitmap.createBitmap(40, 200, Bitmap.Config.ARGB_8888)
        val bytes = ByteArrayOutputStream().also { bitmap.compress(Bitmap.CompressFormat.PNG, 100, it) }.toByteArray()
        return File.createTempFile("comic-direction", ".cbz", context.cacheDir).also { file ->
            ZipOutputStream(file.outputStream()).use { zip ->
                repeat(3) { i -> zip.putNextEntry(ZipEntry("$i.png")); zip.write(bytes); zip.closeEntry() }
            }
        }
    }

    private fun open(file: File, key: String, kind: String?): Unit {
        compose.setContent {
            ComicReaderScreen(ComicSource.Local(file), "Comic", key, onClose = {}, pageTurns = MutableSharedFlow<ReaderPageTurn>(extraBufferCapacity = 1), libraryKind = kind)
        }
        compose.waitUntil(timeoutMillis = 10_000) { runCatching { compose.onNodeWithContentDescription("Page 1").fetchSemanticsNode() }.isSuccess }
        compose.onNodeWithContentDescription("Page 1").performTouchInput { click() }
    }

    private fun shows(direction: String) = compose.onNodeWithText(direction).assertIsDisplayed()

    @Test fun mangaOpensRightToLeft() { val f = archive(); open(f, "d1", "manga"); shows("RTL") }
    @Test fun comicsOpenLeftToRight() { val f = archive(); open(f, "d2", "comics"); shows("LTR") }
    @Test fun booksLibraryComicsOpenLeftToRight() { val f = archive(); open(f, "d3", "books"); shows("LTR") }
    @Test fun anUnknownKindKeepsTheOldRightToLeft() { val f = archive(); open(f, "d4", null); shows("RTL") }

    @Test fun aSavedUserChoiceWinsOverTheKind() {
        val f = archive(); prefs.edit().putBoolean("right_to_left", false).commit()
        open(f, "d5", "manga"); shows("LTR")
    }

    @Test fun aSavedUserChoiceOfRightToLeftWinsForComics() {
        val f = archive(); prefs.edit().putBoolean("right_to_left", true).commit()
        open(f, "d6", "comics"); shows("RTL")
    }

    @Test fun aSavedPerBookChoiceWinsOverEverything() {
        val f = archive(); prefs.edit().putBoolean("right_to_left", true).putBoolean("right_to_left:d7", false).commit()
        open(f, "d7", "manga"); shows("LTR")
    }

    @Test fun flippingTheDirectionIsRememberedForThatBookOnly() {
        val f = archive()
        open(f, "d8", "comics")
        compose.onNodeWithText("LTR").performClick()
        shows("RTL")
        assertEquals(true, prefs.getBoolean("right_to_left:d8", false))
        assertEquals(false, prefs.contains("right_to_left"))
    }
}
