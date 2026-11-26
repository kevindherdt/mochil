package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.requestFocus
import androidx.compose.ui.test.performClick
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.testing.Goldens
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/**
 * The TV rail at a 1080p box's width, in the cinema theme the TV actually renders: resting, open, and with a destination
 * focused. The three states are the whole point of the component, and a golden is the only way to see that the tinted
 * "you are here" tile and the filled focus tile stay distinguishable in a palette as dark as this one.
 */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w960dp-h540dp-xhdpi")
class TvRailGoldenTest {
    @get:Rule val compose = createComposeRule()

    private val libraries = listOf(
        LibrarySummary(id = "books", name = "Books", kind = "books", count = 0),
        LibrarySummary(id = "audiobooks", name = "Audiobooks", kind = "audiobooks", count = 0),
        LibrarySummary(id = "tv", name = "TV Series", kind = "tv", count = 0),
        LibrarySummary(id = "music", name = "Music", kind = "music", count = 0),
    )

    @Composable
    private fun Host(rail: @Composable () -> Unit) {
        MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false, cinema = true) {
            Row(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).testTag("host")) {
                Box(Modifier.fillMaxHeight()) { rail() }
            }
        }
    }

    @Test
    fun restingRailIsTheMockupsIconColumn() {
        compose.setContent {
            Host {
                TvNavRail(libraries, tvRailLibraryKey("books"), onHome = {}, onLibrary = {}, onSettings = {}, onBack = {}, entry = FocusRequester())
            }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "tv-rail-resting-cinema")
    }

    @Test
    fun theFocusedDestinationStaysDistinctFromTheSelectedOne() {
        compose.setContent {
            Host {
                TvNavRail(libraries, tvRailLibraryKey("tv"), onHome = {}, onLibrary = {}, onSettings = {}, onBack = {}, entry = FocusRequester())
            }
        }
        // Focus on Home while Books is the selected destination: the two states have to read differently at ten feet.
        compose.onNodeWithContentDescription("Home").requestFocus()
        compose.waitForIdle()
        Goldens.assertMatches(compose.onNodeWithTag("host"), "tv-rail-focus-cinema")
    }
}
