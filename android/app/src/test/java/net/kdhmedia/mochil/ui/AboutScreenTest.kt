package net.kdhmedia.mochil.ui

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import net.kdhmedia.mochil.BuildConfig
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/** The notices the app owes: Lucide's icons and the two bundled fonts, with their licence texts shipped in res/raw. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class AboutScreenTest {
    @get:Rule val compose = createComposeRule()

    private fun about() {
        compose.setContent { MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) { AboutContent() } }
    }

    @Test fun `the page names the version and every notice`() {
        about()
        compose.onNodeWithText("Mochil ${BuildConfig.VERSION_NAME}").assertIsDisplayed()
        compose.onNodeWithText("Lucide icons").assertIsDisplayed()
        compose.onNodeWithText("Source Sans 3").assertIsDisplayed()
        compose.onNodeWithText("Newsreader").assertIsDisplayed()
    }

    @Test fun `a notice opens to its licence text, which really is in the app`() {
        about()
        compose.onNodeWithText("Lucide icons").performClick()
        // The ISC permission notice itself, not just the name of the licence.
        compose.onNodeWithText("Permission to use, copy, modify, and/or distribute this software for any", substring = true).assertIsDisplayed()
    }

    @Test fun `the settings hub offers About`() {
        org.junit.Assert.assertEquals(SettingsCategory.About, SettingsCategory.fromName("About"))
    }
}
