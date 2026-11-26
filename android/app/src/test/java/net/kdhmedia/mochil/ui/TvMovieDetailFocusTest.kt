package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.activity.ComponentActivity
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performKeyInput
import androidx.compose.ui.test.performSemanticsAction
import androidx.compose.ui.test.pressKey
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.unit.dp
import androidx.compose.ui.semantics.SemanticsActions
import net.kdhmedia.mochil.audio.MusicState
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.podcast.PodcastState
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoState
import net.kdhmedia.mochil.video.VideoTitleDetails
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * The navigation rail opens whenever it takes focus. Opening a movie's detail page moves focus off the
 * poster grid, so nothing must hand focus to the rail: on a real set the panel slid open by itself when
 * a movie was opened, and again on the way back to the shelf.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w960dp-h540dp-mdpi")
class TvMovieDetailFocusTest {
    @get:Rule val compose = createAndroidComposeRule<ComponentActivity>()

    private val movie = VideoItem(
        "movie", "movies", "The Long Film", "", 0, 0, 7200.0, "poster", "stream", "",
        620.0, emptyList(), overview = "A film about focus.", backdropPath = "backdrop",
    )
    private val library = LibrarySummary("movies", "Movies", "movies", 1)

    @Composable
    private fun Shell(video: VideoState, onBack: () -> Unit) {
        val railFocus = FocusRequester()
        Box(Modifier.fillMaxSize()) {
            TvContentFrame(showRail = true, playbackDockVisible = false) {
                TvLibraryScreen(
                    LibraryState(selected = library, libraries = listOf(library), initialLoading = false),
                    MusicState(), video, PodcastState(), null,
                    TvLibraryArtwork({ "" }, { "" }, { "" }, { "" }, { "" },
                        { path -> if (path.isBlank()) "" else "https://example.invalid/$path.jpg" }, { "" }, { "" }),
                    TvLibraryActions(TvNavigationActions(onBack, {}, {}, {})), "", TvGridPositionStore(),
                )
            }
            TvNavRail(listOf(library), tvRailLibraryKey("movies"), {}, {}, {}, {}, railFocus, Modifier.testTag("rail"))
        }
    }

    private fun railWidth(): androidx.compose.ui.unit.Dp {
        val bounds = compose.onNodeWithTag("rail").getUnclippedBoundsInRoot()
        return bounds.right - bounds.left
    }

    @Test
    fun openingAMovieKeepsTheRailCollapsed() {
        val video = VideoState(
            libraryId = "movies", libraryKind = "movies", items = listOf(movie),
            details = mapOf("item:movie" to VideoTitleDetails(genres = listOf("Drama"), runtimeMinutes = 120)),
        )
        compose.setContent { Shell(video) { } }
        compose.waitForIdle()
        compose.onNode(hasText("The Long Film") and hasClickAction())
            .performSemanticsAction(SemanticsActions.RequestFocus) { it() }
        assertEquals("the rail must stay collapsed while the shelf has focus", TvNavRailWidth, railWidth())
        compose.onNode(hasText("The Long Film") and hasClickAction()).performKeyInput { pressKey(Key.Enter) }
        compose.waitForIdle()
        compose.mainClock.advanceTimeBy(500)
        assertEquals("opening a movie must not open the navigation rail", TvNavRailWidth, railWidth())
    }

    @Test
    fun goingBackFromAMovieKeepsTheRailCollapsed() {
        val video = VideoState(
            libraryId = "movies", libraryKind = "movies", items = listOf(movie),
            details = mapOf("item:movie" to VideoTitleDetails(genres = listOf("Drama"), runtimeMinutes = 120)),
        )
        compose.setContent { Shell(video) { } }
        compose.waitForIdle()
        compose.onNode(hasText("The Long Film") and hasClickAction())
            .performSemanticsAction(SemanticsActions.RequestFocus) { it() }
        compose.onNode(hasText("The Long Film") and hasClickAction()).performKeyInput { pressKey(Key.Enter) }
        compose.waitForIdle()
        compose.mainClock.advanceTimeBy(500)
        compose.activityRule.scenario.onActivity { it.onBackPressedDispatcher.onBackPressed() }
        compose.waitForIdle()
        compose.mainClock.advanceTimeBy(500)
        assertEquals("going back to the shelf must not open the navigation rail", TvNavRailWidth, railWidth())
    }
}
