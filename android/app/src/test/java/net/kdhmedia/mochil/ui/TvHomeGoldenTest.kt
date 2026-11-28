package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.dashboard.ContinueListening
import net.kdhmedia.mochil.dashboard.DashboardState
import net.kdhmedia.mochil.testing.Goldens
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import net.kdhmedia.mochil.video.VideoItem
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/**
 * Home at a 1080p box's size, in the cinema scheme: the billboard over the two resume rails. This is the TV screen the
 * sheets draw in most detail, and the one nobody can check from a desk, so it gets a golden in both a resumed and a
 * fresh state.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w960dp-h540dp-xhdpi")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class TvHomeGoldenTest {
    @get:Rule val compose = createComposeRule()

    private val watching = VideoItem(
        "ep", "tv", "Episode Four: The Hanging", "Silo", 1, 4, 3120.0, "", "stream", "", 1680.0, emptyList(),
        overview = "A body is found in the silo's lower levels.",
    )
    private val listening = ContinueListening(AudioTrack("trk", "music", "Gee", "Girls' Generation", "Oh!", 201.0), 92.0, 201.0)

    @Composable
    private fun Home(state: DashboardState) {
        MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false, cinema = true) {
            Surface(Modifier.fillMaxSize().testTag("home")) {
                Box(Modifier.fillMaxSize()) {
                    TvHomeScreen(
                        state,
                        { "https://example.invalid/listening.jpg" },
                        { "https://example.invalid/watching.jpg" },
                        authorizationHeader = null,
                        onListen = {}, onWatch = {}, onDismissListening = {}, onDismissWatching = {}, onRetry = {}, focusRail = {},
                    )
                }
            }
        }
    }

    @Test
    fun homeResumesWithABillboardAndTwoRails() {
        compose.setContent { Home(DashboardState(watching = listOf(watching), listening = listOf(listening), loading = false)) }
        compose.waitForIdle()
        Goldens.assertMatches(compose.onNodeWithTag("home"), "tv-home-billboard-cinema")
    }

    @Test
    fun aFreshLibrarySaysWhatToDoRatherThanShowingNothing() {
        compose.setContent { Home(DashboardState(watching = emptyList(), listening = emptyList(), loading = false)) }
        compose.waitForIdle()
        Goldens.assertMatches(compose.onNodeWithTag("home"), "tv-home-empty-cinema")
    }
}
