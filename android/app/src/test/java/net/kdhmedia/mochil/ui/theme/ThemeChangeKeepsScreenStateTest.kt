package net.kdhmedia.mochil.ui.theme

import androidx.compose.material3.Text
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.test.junit4.createComposeRule
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * Choosing a palette used to throw the person out of Settings and back to Home, because the theme
 * wrapper changed its structure when the palette changed: the content sat inside a Box for most
 * palettes and directly under MaterialTheme for e-ink. A different call site is a different
 * composition, so the whole app shell was disposed and rebuilt, and every rememberSaveable in it
 * (including which screen was open) came back as its initial value.
 *
 * The screen must keep its state when only the colours change.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class ThemeChangeKeepsScreenStateTest {

    @get:Rule
    val compose = createComposeRule()

    private fun creationsAfterSwitchingTo(palette: AppPalette): Int {
        val selection = mutableStateOf(AppThemeSelection(palette = AppPalette.Espresso, mode = AppThemeMode.Light))
        var creations = 0
        compose.setContent {
            MochilTheme(selection = selection.value) {
                // Stands in for the app shell: the identity of this remember is what "the screen was
                // rebuilt" means. A new instance means the shell lost its state.
                remember { creations += 1 }
                Text("Settings")
            }
        }
        compose.runOnIdle { selection.value = selection.value.copy(palette = palette) }
        compose.waitForIdle()
        return creations
    }

    @Test
    fun switchingToPaperKeepsTheScreenInPlace() {
        assertEquals("Paper rebuilt the screen instead of recolouring it", 1, creationsAfterSwitchingTo(AppPalette.Paper))
    }

    @Test
    fun switchingToEInkKeepsTheScreenInPlace() {
        assertEquals("E-ink rebuilt the screen instead of recolouring it", 1, creationsAfterSwitchingTo(AppPalette.EInk))
    }

    @Test
    fun switchingToKilnKeepsTheScreenInPlace() {
        assertEquals("Kiln rebuilt the screen instead of recolouring it", 1, creationsAfterSwitchingTo(AppPalette.Kiln))
    }
}
