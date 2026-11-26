package net.kdhmedia.mochil.ui.kit

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.assertHeightIsAtLeast
import androidx.compose.ui.test.assertWidthIsAtLeast
import androidx.compose.ui.test.assertIsNotSelected
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.testing.Goldens
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilSpace
import net.kdhmedia.mochil.ui.theme.MochilTheme
import net.kdhmedia.mochil.ui.theme.backdrop
import net.kdhmedia.mochil.ui.theme.swatch
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class KitComponentsTest {
    @get:Rule val compose = createComposeRule()

    @Composable
    private fun Host(palette: AppPalette = AppPalette.Tide, mode: AppThemeMode = AppThemeMode.Dark, content: @Composable () -> Unit) {
        MochilTheme(AppThemeSelection(palette, mode), glow = false) {
            // A Surface, as the app's scaffold is: it paints the page colour and sets the default content colour.
            Surface(Modifier.width(390.dp).testTag("host"), color = MaterialTheme.colorScheme.background) {
                Column(Modifier.padding(MochilSpace.s16), verticalArrangement = Arrangement.spacedBy(MochilSpace.s12)) { content() }
            }
        }
    }

    private val modes = listOf(PillItem("system", "Device"), PillItem("light", "Light"), PillItem("dark", "Dark"))

    // ---- Card

    @Test fun `a card shows its content and reports a tap only when it is clickable`() {
        var taps = 0
        compose.setContent {
            Host {
                MochilCard { Text("Plain card") }
                MochilCard(onClick = { taps++ }) { Text("Tappable card") }
            }
        }
        compose.onNodeWithText("Plain card").performClick()
        assertEquals(0, taps)
        compose.onNodeWithText("Tappable card").performClick()
        assertEquals(1, taps)
    }

    @Test fun `golden card Tide dark`() {
        compose.setContent {
            Host {
                MochilCard {
                    Text("Server", style = MaterialTheme.typography.bodyMedium)
                    Text("https://media.example.com", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "card-tide-dark")
    }

    // ---- Pills

    @Test fun `the active pill is selected and the others are not`() {
        compose.setContent { Host { PillRow(modes, "light", {}) } }
        compose.onNodeWithText("Light").assertIsSelected()
        compose.onNodeWithText("Device").assertIsNotSelected()
        compose.onNodeWithText("Dark").assertIsNotSelected()
    }

    @Test fun `tapping a pill reports its id`() {
        var picked: String? = null
        compose.setContent { Host { PillRow(modes, "light", { picked = it }) } }
        compose.onNodeWithText("Dark").performClick()
        assertEquals("dark", picked)
    }

    @Test fun `a pill is drawn at the token height but is a full touch target both ways`() {
        compose.setContent { Host { PillRow(modes, "light", {}) } }
        modes.forEach {
            compose.onNodeWithText(it.label).assertHeightIsAtLeast(MochilChrome.touchMin)
            compose.onNodeWithText(it.label).assertWidthIsAtLeast(MochilChrome.touchMin)
        }
    }

    @Test fun `golden pills Tide dark`() {
        compose.setContent { Host { PillRow(modes, "dark", {}) } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "pills-tide-dark")
    }

    @Test fun `golden pills Ember light`() {
        compose.setContent { Host(AppPalette.Ember, AppThemeMode.Light) { PillRow(modes, "system", {}) } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "pills-ember-light")
    }

    // ---- Swatches

    private fun swatches(dark: Boolean) = AppPalette.entries.map { SwatchItem(it.name, it.label, it.backdrop(dark), it.swatch(dark)) }

    @Test fun `the chosen swatch is selected and tapping another reports it`() {
        var picked: String? = null
        compose.setContent { Host { SwatchRow(swatches(true), AppPalette.Tide.name, { picked = it }) } }
        compose.onNodeWithText("Tide").assertIsSelected()
        compose.onNodeWithText("Ember").assertIsNotSelected()
        compose.onNodeWithText("E-ink").performClick()
        assertEquals(AppPalette.EInk.name, picked)
    }

    @Test fun `a swatch is a full touch target`() {
        compose.setContent { Host { SwatchRow(swatches(true), AppPalette.Tide.name, {}) } }
        AppPalette.entries.forEach { compose.onNodeWithText(it.label).assertHeightIsAtLeast(MochilChrome.touchMin) }
    }

    @Test fun `golden swatches at the largest font scale wrap the long name instead of cutting it`() {
        compose.setContent {
            CompositionLocalProvider(LocalDensity provides Density(LocalDensity.current.density, 2f)) {
                Host { SwatchRow(swatches(true), AppPalette.Tide.name, {}) }
            }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "swatches-tide-dark-font2")
    }

    @Test fun `golden swatches Tide dark`() {
        compose.setContent { Host { SwatchRow(swatches(true), AppPalette.Tide.name, {}) } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "swatches-tide-dark")
    }

    @Test fun `golden swatches Ember light`() {
        compose.setContent { Host(AppPalette.Ember, AppThemeMode.Light) { SwatchRow(swatches(false), AppPalette.Ember.name, {}) } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "swatches-ember-light")
    }

    // ---- Settings entry

    @Test fun `an entry shows its label and description and reports a tap`() {
        var taps = 0
        compose.setContent { Host { SettingsEntry(LucideIcons.Palette, "Appearance", "Theme and light or dark mode.", { taps++ }) } }
        compose.onNodeWithText("Theme and light or dark mode.", useUnmergedTree = true).assertExists()
        compose.onNodeWithText("Appearance", useUnmergedTree = true).performClick()
        assertEquals(1, taps)
    }

    @Test fun `golden entries Tide dark`() {
        compose.setContent {
            Host {
                SettingsEntry(LucideIcons.Palette, "Appearance", "Theme and light or dark mode.", {})
                SettingsEntry(LucideIcons.Download, "Downloads", "Where offline copies are kept.", {})
            }
        }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "entries-tide-dark")
    }

    @Test fun `golden entry at the largest font scale wraps its description instead of clipping it`() {
        compose.setContent {
            CompositionLocalProvider(LocalDensity provides Density(LocalDensity.current.density, 2f)) {
                Host { SettingsEntry(LucideIcons.ShieldCheck, "Privacy", "Annotations, diagnostics and crash reports.", {}, Modifier.fillMaxWidth()) }
            }
        }
        compose.onNodeWithText("Annotations, diagnostics and crash reports.", useUnmergedTree = true).assertExists()
        Goldens.assertMatches(compose.onNodeWithTag("host"), "entry-tide-dark-font2")
    }

    // ---- Library header pieces

    private val shelves = listOf(PillItem("books", "Books", LucideIcons.BookOpen), PillItem("manga", "Manga", LucideIcons.BookImage))

    @Test fun `a pill with a glyph is still found and chosen by its label`() {
        var chosen = ""
        compose.setContent { Host { PillRow(shelves, "books", { chosen = it }) } }
        compose.onNodeWithText("Books").assertIsSelected()
        compose.onNodeWithText("Manga").performClick()
        assertEquals("manga", chosen)
    }

    @Test fun `the pill row brings the chosen pill into view`() {
        val many = (1..14).map { PillItem("lib$it", "Library number $it", LucideIcons.BookOpen) }
        compose.setContent { Host { PillRow(many, "lib14", {}) } }
        compose.waitForIdle()
        compose.onNodeWithText("Library number 14").assertIsDisplayed()
    }

    @Test fun `a chip is a full touch target in both directions and says whether it is chosen`() {
        val chips = listOf(PillItem("all", "All"), PillItem("in_progress", "In Progress"), PillItem("unread", "Unread"))
        var chosen = ""
        compose.setContent { Host { ChipRow(chips, "all", { chosen = it }) } }
        chips.forEach {
            compose.onNodeWithText(it.label).assertHeightIsAtLeast(MochilChrome.touchMin)
            compose.onNodeWithText(it.label).assertWidthIsAtLeast(MochilChrome.touchMin)
        }
        compose.onNodeWithText("All").assertIsSelected()
        compose.onNodeWithText("Unread").assertIsNotSelected()
        compose.onNodeWithText("Unread").performClick()
        assertEquals("unread", chosen)
    }

    @Test fun `a header icon button is a full touch target, has a name and reports a tap`() {
        var taps = 0
        compose.setContent { Host { HeaderIconButton(LucideIcons.Search, "Search", { taps++ }) } }
        compose.onNodeWithContentDescription("Search").assertHeightIsAtLeast(MochilChrome.touchMin)
        compose.onNodeWithContentDescription("Search").assertWidthIsAtLeast(MochilChrome.touchMin)
        compose.onNodeWithContentDescription("Search").performClick()
        assertEquals(1, taps)
    }

    @Test fun `a disabled header icon button does not report a tap`() {
        var taps = 0
        compose.setContent { Host { HeaderIconButton(LucideIcons.Plus, "Add", { taps++ }, enabled = false) } }
        compose.onNodeWithContentDescription("Add").performClick()
        assertEquals(0, taps)
    }

    @Test fun `golden chips Tide dark`() {
        val chips = listOf(PillItem("all", "All"), PillItem("in_progress", "In Progress"), PillItem("unread", "Unread"))
        compose.setContent { Host { ChipRow(chips, "in_progress", {}) } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "chips-tide-dark")
    }

    @Test fun `golden pills with glyphs Tide dark`() {
        compose.setContent { Host { PillRow(shelves, "books", {}) } }
        Goldens.assertMatches(compose.onNodeWithTag("host"), "pills-glyphs-tide-dark")
    }
}
