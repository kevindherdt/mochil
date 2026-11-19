package net.kdhmedia.mochil.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.performClick
import net.kdhmedia.mochil.video.*
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w400dp-h800dp")
class MobileMovieDetailTest {
    @get:Rule val compose = createComposeRule()

    @Test fun movieOpensDetailsBeforePlayback() {
        val movie = VideoItem("id", "lib", "Movie", "", 0, 0, 100.0, "", "stream", "", 20.0, emptyList(), overview = "A synopsis")
        val details = VideoTitleDetails(cast = listOf(VideoCastMember("Actor", "Hero")))
        var plays = 0
        var requested: VideoItem? = null
        compose.setContent { MaterialTheme {
            VideoScreen(VideoState(libraryId = "lib", libraryKind = "movies", items = listOf(movie), details = mapOf("item:id" to details)), { it }, null,
                {}, {}, {}, {}, {}, {}, { plays++ }, {}, { _, _ -> }, {}, {}, {}, { requested = it })
        } }
        compose.onNodeWithText("Movie").performClick()
        assertEquals(0, plays)
        compose.waitForIdle()
        assertEquals(movie, requested)
        compose.onNodeWithText("A synopsis").assertIsDisplayed()
        compose.onNodeWithText("Actor").assertIsDisplayed()
        compose.onNodeWithText("Resume").performClick()
        assertEquals(1, plays)
        compose.onNodeWithContentDescription("Back to movies").performClick()
        compose.onNodeWithText("Movie").assertIsDisplayed()
    }

    @Test fun openMovieTracksUpdatedPlaybackProgress() {
        val movie = VideoItem("id", "lib", "Movie", "", 0, 0, 100.0, "", "stream", "", 0.0, emptyList())
        val state = androidx.compose.runtime.mutableStateOf(VideoState(libraryId = "lib", libraryKind = "movies", items = listOf(movie)))
        var played: VideoItem? = null
        compose.setContent { MaterialTheme {
            VideoScreen(state.value, { it }, null, {}, {}, {}, {}, {}, {}, { played = it }, {}, { _, _ -> }, {}, {}, {})
        } }
        compose.onNodeWithText("Movie").performClick()
        compose.onNodeWithText("Play").assertIsDisplayed()
        compose.runOnIdle { state.value = state.value.copy(items = listOf(movie.copy(resumeSeconds = 30.0))) }
        compose.onNodeWithText("Resume").performClick()
        assertEquals(30.0, played!!.resumeSeconds, 0.0)
    }

    @Test fun unplayableMovieOpensDetailsWithoutStartingPlayback() {
        val movie = VideoItem("id", "lib", "Movie", "", 0, 0, 100.0, "", "", "", 0.0, emptyList(), overview = "Story")
        var plays = 0
        compose.setContent { MaterialTheme {
            VideoScreen(VideoState(libraryId = "lib", libraryKind = "movies", items = listOf(movie)), { it }, null,
                {}, {}, {}, {}, {}, {}, { plays++ }, {}, { _, _ -> }, {}, {}, {})
        } }
        compose.onNodeWithText("Movie").performClick()
        compose.onNodeWithText("Story").assertIsDisplayed()
        assertEquals(0, plays)
    }

    @Test fun movieDetailsShowPosterAndFacts() {
        val movie = VideoItem("id", "lib", "Movie", "", 0, 0, 7200.0, "poster", "stream", "", 0.0, emptyList(), year = 2024)
        val details = VideoTitleDetails(genres = listOf("Drama"), runtimeMinutes = 120)
        compose.setContent { MaterialTheme {
            VideoScreen(VideoState(libraryId = "lib", libraryKind = "movies", items = listOf(movie), details = mapOf("item:id" to details)), { it }, null,
                {}, {}, {}, {}, {}, {}, {}, {}, { _, _ -> }, {}, {}, {})
        } }
        compose.onNodeWithText("Movie").performClick()
        compose.onNodeWithContentDescription("Artwork for Movie").assertIsDisplayed()
        compose.onNodeWithText("2024 · 2h · Drama", substring = true).assertIsDisplayed()
    }

    @Test fun movieCardAnnouncesOpenRatherThanPlay() {
        val movie = VideoItem("id", "lib", "Movie", "", 0, 0, 100.0, "", "stream", "", 0.0, emptyList())
        compose.setContent { MaterialTheme {
            VideoScreen(VideoState(libraryId = "lib", libraryKind = "movies", items = listOf(movie)), { it }, null,
                {}, {}, {}, {}, {}, {}, {}, {}, { _, _ -> }, {}, {}, {})
        } }
        val action = compose.onNodeWithText("Movie").fetchSemanticsNode().config[androidx.compose.ui.semantics.SemanticsActions.OnClick]
        assertEquals("Open Movie", action.label)
    }

    @Test fun libraryDoesNotRepeatContinueWatching() {
        val movie = VideoItem("id", "lib", "Movie", "", 0, 0, 100.0, "", "stream", "", 20.0, emptyList())
        compose.setContent { MaterialTheme {
            VideoScreen(VideoState(libraryId = "lib", libraryKind = "movies", items = listOf(movie), continueWatching = listOf(movie)), { it }, null,
                {}, {}, {}, {}, {}, {}, {}, {}, { _, _ -> }, {}, {}, {})
        } }
        compose.onNodeWithText("Continue watching").assertDoesNotExist()
        compose.onNodeWithText("Movie").performClick()
        compose.onNodeWithText("Resume").assertIsDisplayed()
    }
}
