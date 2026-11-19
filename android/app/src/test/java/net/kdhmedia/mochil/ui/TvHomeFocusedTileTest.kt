package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.InputMode
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalInputModeManager
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.performKeyInput
import androidx.compose.ui.test.performSemanticsAction
import androidx.compose.ui.test.pressKey
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.dashboard.ContinueListening
import net.kdhmedia.mochil.dashboard.DashboardState
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import net.kdhmedia.mochil.video.VideoItem
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/**
 * At 1.6x text on a short window the tile's art fits but its title sits below the fold, so a focused shelf looked like a
 * picture with no name. The whole tile — art, title, artist and the room the focus ring needs — has to be inside the
 * scrolling viewport when the remote lands on it.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w800dp-h450dp-mdpi")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class TvHomeFocusedTileTest {
    @get:Rule val compose = createComposeRule()

    private val listeningTitle = "Listening fixture"
    private val listening = ContinueListening(AudioTrack("a", "music", listeningTitle, "Artist", "Album", 3600.0), 60.0, 3600.0)
    private val watching = VideoItem("ep", "tv", "005 - The Cotton Drifting Chapter Part 1 Jealousy and Another Extremely Long Episode Title", "Baki", 1, 1, 1440.0, "", "stream", "", 120.0, emptyList())

    @Composable
    private fun Shell(content: @Composable () -> Unit) {
        MochilTheme(selection = AppThemeSelection(), cinema = true, glow = false) {
            val input = LocalInputModeManager.current
            LaunchedEffect(Unit) { input.requestInputMode(InputMode.Keyboard) }
            Surface {
                TvContentFrame(showRail = true, playbackDockVisible = false) {
                    CompositionLocalProvider(LocalDensity provides Density(1f, 1.6f)) { content() }
                }
            }
        }
    }

    private fun mount() {
        compose.setContent {
            Shell {
                Box(Modifier.fillMaxSize()) {
                    TvHomeScreen(
                        DashboardState(watching = listOf(watching), listening = listOf(listening), loading = false),
                        { "https://example.invalid/listening.jpg" },
                        { "https://example.invalid/watching.jpg" },
                        authorizationHeader = null,
                        onListen = {}, onWatch = {}, onDismissListening = {}, onDismissWatching = {}, onRetry = {}, focusRail = {},
                    )
                }
            }
        }
        compose.waitForIdle()
        compose.mainClock.advanceTimeBy(600)
        compose.onNode(hasContentDescription("Artwork for ${watching.displayTitle}") and hasClickAction())
            .performSemanticsAction(androidx.compose.ui.semantics.SemanticsActions.RequestFocus) { it() }
        compose.onNode(hasContentDescription("Artwork for ${watching.displayTitle}") and hasClickAction())
            .performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNode(hasContentDescription("Artwork for $listeningTitle") and hasClickAction()).assertIsFocused()
        compose.mainClock.advanceTimeBy(600)
        compose.waitForIdle()
    }

    /** The scrolling viewport the Home shelves live in, in raw pixels like the bounds they are compared with. */
    private fun scrollViewport() = compose
        .onNode(SemanticsMatcher.keyIsDefined(SemanticsProperties.VerticalScrollAxisRange))
        .fetchSemanticsNode().boundsInRoot

    @Test
    fun theFocusedTileShowsItsTitleAndArtistInsideTheViewport() {
        mount()
        val viewport = scrollViewport()
        val art = compose.onNodeWithContentDescription("Artwork for $listeningTitle", useUnmergedTree = true).fetchSemanticsNode().boundsInRoot
        // The hero header repeats the focused title, so the caption is the match that sits below the art.
        val caption = compose.onAllNodesWithText(listeningTitle, useUnmergedTree = true).fetchSemanticsNodes()
            .map { it.boundsInRoot }
            .filter { it.top > art.top }
            .maxByOrNull { it.top }
        assertTrue("the focused tile has no title beneath its art", caption != null)
        assertTrue(
            "the focused tile's title sits below the Home viewport: caption=$caption viewport=$viewport art=$art",
            caption!!.bottom <= viewport.bottom + 0.5f,
        )
    }

    @Test
    fun theFocusedTileKeepsRoomForItsFocusRing() {
        mount()
        val viewport = scrollViewport()
        val art = compose.onNodeWithContentDescription("Artwork for $listeningTitle", useUnmergedTree = true).fetchSemanticsNode().boundsInRoot
        // The ring grows the tile about 3.5% around its centre, so a tile flush with the viewport edge loses its halo.
        val room = with(compose.density) { 6.dp.toPx() }
        assertTrue(
            "the focus ring is cropped by the Home viewport: art=$art viewport=$viewport",
            art.top - room >= viewport.top && art.bottom + room <= viewport.bottom,
        )
    }
}
