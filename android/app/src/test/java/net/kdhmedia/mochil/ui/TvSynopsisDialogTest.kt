package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.captureToImage
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.test.performClick
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.video.VideoCastMember
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoShow
import net.kdhmedia.mochil.video.VideoState
import net.kdhmedia.mochil.video.VideoTitleDetails
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/** A long synopsis is clamped so the cast fits; Read more opens the whole text on the remote. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w960dp-h540dp-mdpi")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class TvSynopsisDialogTest {
    @get:Rule val compose = createComposeRule()

    private val story = "Ja-yoon is a high school student who struggles with memory loss, and the story keeps going " +
        "long past the lines a detail page can spare for it. ".repeat(16)
    private val episode = VideoItem("ep", "tv", "Episode One", "Show", 1, 1, 1200.0, "", "stream", "", 0.0, emptyList(), overview = story)
    private val show = VideoShow("The Witch: Part 1", "The Witch", "show", 1, listOf(1), "", year = 2018, overview = story)

    private fun mount(overview: String) {
        val state = VideoState(
            libraryId = "tv", libraryKind = "tv", selectedShow = show.copy(overview = overview), selectedSeason = 1,
            items = listOf(episode.copy(overview = overview)),
            details = mapOf("show:The Witch" to VideoTitleDetails(cast = listOf(VideoCastMember("Kim Da-mi", "Koo Ja-yoon")))),
        )
        compose.setContent {
            MaterialTheme {
                Box(Modifier.size(960.dp, 540.dp)) {
                    TvShowDetail(show.copy(overview = overview), state, null, { it }, {}, {}, { _, _ -> }, {}, {})
                }
            }
        }
        compose.waitForIdle()
    }

    @Test
    fun readMoreOpensTheWholeTextAndClosesAgain() {
        mount(story)
        compose.onNodeWithText("Read more").performClick()
        compose.waitForIdle()
        val dir = java.io.File("build/reports/tv-layout").apply { mkdirs() }
        java.io.File(dir, "synopsis-overlay.png").outputStream().use { out ->
            compose.onRoot().captureToImage().asAndroidBitmap().compress(android.graphics.Bitmap.CompressFormat.PNG, 100, out)
        }
        compose.onNodeWithText("Close").assertIsDisplayed()
        compose.onNodeWithText("Close").performClick()
        compose.onNodeWithText("Close").assertDoesNotExist()
    }

    @Test
    fun aStoryThatFitsHasNoReadMore() {
        mount("A short synopsis.")
        compose.onNodeWithText("Read more").assertDoesNotExist()
    }
}
