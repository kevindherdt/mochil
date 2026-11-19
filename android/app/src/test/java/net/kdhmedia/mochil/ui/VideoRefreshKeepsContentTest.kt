package net.kdhmedia.mochil.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithContentDescription
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoState
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * Every return to the app re-reads playback progress, and that refresh used to own the whole screen: the shelf the
 * viewer was on dropped to a spinner (and to the error panel on a failed refresh) and came back scrolled to the top.
 * The empty state is what a first load looks like, so only an empty screen may show the spinner or the error.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w400dp-h800dp")
class VideoRefreshKeepsContentTest {
    @get:Rule val compose = createComposeRule()

    private val movie = VideoItem("id", "lib", "Movie", "", 0, 0, 100.0, "", "stream", "", 20.0, emptyList())

    private fun mount(state: VideoState) {
        compose.setContent {
            MaterialTheme {
                VideoScreen(state, { it }, null, {}, {}, {}, {}, {}, {}, {}, {}, { _, _ -> }, {}, {}, {})
            }
        }
        compose.waitForIdle()
    }

    @Test
    fun aRefreshWithContentOnScreenKeepsTheShelfVisible() {
        mount(VideoState(libraryId = "lib", libraryKind = "movies", items = listOf(movie), loading = true))
        compose.onNodeWithText("Movie").assertIsDisplayed()
        assertEquals("a populated shelf must never show the full-screen load", 0, compose.onAllNodesWithContentDescription("Loading videos").fetchSemanticsNodes().size)
    }

    @Test
    fun aFailedRefreshWithContentOnScreenKeepsTheShelfVisible() {
        mount(VideoState(libraryId = "lib", libraryKind = "movies", items = listOf(movie), error = "Could not load these videos."))
        compose.onNodeWithText("Movie").assertIsDisplayed()
        assertEquals("a failed background refresh must not replace the shelf", 0, compose.onAllNodesWithText("Try again").fetchSemanticsNodes().size)
    }

    @Test
    fun anEmptyScreenStillShowsTheLoadAndTheError() {
        mount(VideoState(libraryId = "lib", libraryKind = "movies", loading = true))
        // The indicator carries its label as contentDescription, not as text.
        compose.onNodeWithContentDescription("Loading videos").assertExists()
    }

    @Test
    fun anEmptyScreenStillShowsTheRetryPanel() {
        mount(VideoState(libraryId = "lib", libraryKind = "movies", error = "Could not load these videos."))
        compose.onNodeWithText("Try again").assertIsDisplayed()
    }
}
