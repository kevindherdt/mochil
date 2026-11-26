package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.assertHeightIsAtLeast
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performClick
import androidx.compose.ui.unit.dp
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Redo
import androidx.compose.material.icons.filled.Undo
import net.kdhmedia.mochil.testing.Goldens
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/** The mock-up's 15-second skips: a full touch target each, named for TalkBack, and the number visible inside the arrow. */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
class PlayerTransportTest {
    @get:Rule val compose = createComposeRule()

    @Test fun `the skips are full targets that report a tap`() {
        var back = 0
        compose.setContent {
            MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) {
                SkipSecondsButton(Icons.Default.Undo, "Back 15 seconds", { back++ })
            }
        }
        compose.onNodeWithContentDescription("Back 15 seconds").assertHeightIsAtLeast(MochilChrome.touchMin)
        compose.onNodeWithContentDescription("Back 15 seconds").performClick()
        assertEquals(1, back)
    }

    @Test fun `golden the skip pair`() {
        compose.setContent {
            MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) {
                Surface(Modifier.width(390.dp).testTag("host"), color = MaterialTheme.colorScheme.background) {
                    Row(Modifier.padding(16.dp), horizontalArrangement = Arrangement.spacedBy(24.dp)) {
                        SkipSecondsButton(Icons.Default.Undo, "Back 15 seconds") {}
                        SkipSecondsButton(Icons.Default.Redo, "Forward 15 seconds") {}
                    }
                }
            }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "player-skips-tide-dark")
    }
}
