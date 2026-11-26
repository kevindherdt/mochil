package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.BookStatusFilter
import net.kdhmedia.mochil.testing.Goldens
import net.kdhmedia.mochil.ui.kit.MochilCard
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemeMode
import net.kdhmedia.mochil.ui.theme.AppThemeSelection
import net.kdhmedia.mochil.ui.theme.LocalEInkPalette
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import net.kdhmedia.mochil.ui.theme.MochilTheme
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/**
 * Every palette in both modes on one sheet, so a colour that reads in Tide dark and vanishes in Ember light (or on E-ink) is
 * seen before it ships. Each image holds the light theme on the left and the dark one on the right: the same card, chips and
 * pill the library and Discover draw.
 */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35], qualifiers = "w390dp-h844dp-xxhdpi")
class PaletteMatrixTest {
    @get:Rule val compose = createComposeRule()

    private fun book() = BookSummary(
        "id", "books", "Neuromancer", listOf("William Gibson"), "Sprawl", 3.0, "epub", "application/epub+zip", "en", "", 1,
        readingFraction = .42, readAlong = true,
    )

    /** One half of the sheet: a cover, the status chips, and a settings card, on this theme's own ground. */
    @Composable
    private fun Half(palette: AppPalette, mode: AppThemeMode) {
        MochilTheme(AppThemeSelection(palette, mode), glow = false) {
            Column(
                Modifier.width(390.dp).height(430.dp).background(MaterialTheme.colorScheme.background).padding(8.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Box(Modifier.width(150.dp)) {
                    BookCard(
                        book(), "books", null, null, {}, {},
                        art = { modifier -> Box(modifier.background(Color(0xFF3A3A3A))) },
                    )
                }
                StatusChipRow(StatusChips(BookStatusFilter.InProgress) {})
                MochilCard(contentPadding = PaddingValues(horizontal = 12.dp, vertical = 14.dp)) {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("Theme", color = MaterialTheme.colorScheme.onSurface)
                        Text(palette.label, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1)
                    }
                }
            }
        }
    }

    @Composable
    private fun Sheet(palette: AppPalette, mode: AppThemeMode) {
        // One theme per image: two MochilThemes in one composition drew only the first and left the rest of the frame blank.
        Box(Modifier.testTag("host")) { Half(palette, mode) }
    }

    private fun golden(palette: AppPalette, mode: AppThemeMode, name: String) {
        compose.setContent { Sheet(palette, mode) }
        Goldens.assertMatches(compose.onNodeWithTag("host"), name)
    }

    @Test fun `golden Ember light`() = golden(AppPalette.Ember, AppThemeMode.Light, "palette-ember-light")
    @Test fun `golden Ember dark`() = golden(AppPalette.Ember, AppThemeMode.Dark, "palette-ember-dark")
    @Test fun `golden Tide light`() = golden(AppPalette.Tide, AppThemeMode.Light, "palette-tide-light")
    @Test fun `golden Tide dark`() = golden(AppPalette.Tide, AppThemeMode.Dark, "palette-tide-dark")
    @Test fun `golden Magenta Cream light`() = golden(AppPalette.MagentaCream, AppThemeMode.Light, "palette-magenta-light")
    @Test fun `golden Magenta Cream dark`() = golden(AppPalette.MagentaCream, AppThemeMode.Dark, "palette-magenta-dark")
    @Test fun `golden E-ink light`() = golden(AppPalette.EInk, AppThemeMode.Light, "palette-eink-light")
    @Test fun `golden E-ink dark`() = golden(AppPalette.EInk, AppThemeMode.Dark, "palette-eink-dark")

    @Test fun `the E-ink palette is flagged, so shadows and glow can be dropped`() {
        val flags = mutableListOf<Pair<Boolean, Boolean>>()
        compose.setContent {
            MochilTheme(AppThemeSelection(AppPalette.EInk, AppThemeMode.Light), glow = false) {
                flags.add(LocalEInkPalette.current to false)
            }
            MochilTheme(AppThemeSelection(AppPalette.Tide, AppThemeMode.Dark), glow = false) {
                flags.add(LocalEInkPalette.current to true)
            }
        }
        compose.waitForIdle()
        assertTrue("E-ink must report itself", flags.any { it.first && !it.second })
        assertTrue("a colour palette must not", flags.any { !it.first && it.second })
    }
}
