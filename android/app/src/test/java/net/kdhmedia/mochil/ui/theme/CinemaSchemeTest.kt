package net.kdhmedia.mochil.ui.theme

import androidx.compose.material3.ColorScheme
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.luminance
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The TV treatment used to flatten every palette onto one navy base (#0C0D12) with only the accent
 * differing, so picking a theme changed almost nothing on a television. These tests pin the parts
 * that make a TV scheme correct: it stays dark enough for a dim room, it keeps the chosen palette's
 * own hue, its text stays readable, and its accent is the palette's accent rather than a leftover.
 */
class CinemaSchemeTest {

    private val darkest = listOf(AppPalette.Ember, AppPalette.Tide, AppPalette.MagentaCream, AppPalette.EInk)

    private fun tv(palette: AppPalette): ColorScheme = paletteScheme(palette, dark = true).asCinema()

    /** WCAG relative-luminance contrast ratio. */
    private fun contrast(a: Color, b: Color): Double {
        val la = a.luminance().toDouble()
        val lb = b.luminance().toDouble()
        return (maxOf(la, lb) + 0.05) / (minOf(la, lb) + 0.05)
    }

    @Test
    fun everyPaletteGetsADarkScreen() {
        for (palette in AppPalette.entries) {
            val c = tv(palette)
            assertTrue("$palette background is not near-black: ${c.background}", c.background.luminance() < 0.05f)
            assertTrue("$palette surface is darker than its background", c.surface.luminance() >= c.background.luminance())
            assertTrue("$palette raised is darker than its surface", c.surfaceVariant.luminance() >= c.surface.luminance())
            // E-ink is deliberately pure black and white, so its steps are a hair apart by design;
            // every other palette must show a visible card edge rather than a flat wall.
            if (palette != AppPalette.EInk) {
                assertTrue("$palette surface is not visibly above its background", c.surface.luminance() > c.background.luminance())
            }
        }
    }

    @Test
    fun theThemeYouPickIsTheThemeYouSee() {
        // The whole point: three different themes must not land on the same background. Under the old
        // fixed navy they did, and the TV looked identical whichever one was chosen.
        val backgrounds = darkest.take(3).map { tv(it).background }
        assertEquals("Ember, Tide and Magenta Cream share a TV background", 3, backgrounds.toSet().size)

        // Warm palettes stay warm and the cool one stays cool, so the hue survives the darkening.
        val ember = tv(AppPalette.Ember).background
        assertTrue("Ember's TV base lost its warmth", ember.red > ember.blue)
        val tide = tv(AppPalette.Tide).background
        assertTrue("Tide's TV base lost its cool cast", tide.blue >= tide.red)
    }

    @Test
    fun textStaysReadableOnEveryTheme() {
        for (palette in AppPalette.entries) {
            val c = tv(palette)
            assertTrue("$palette body text contrast ${contrast(c.onBackground, c.background)}",
                contrast(c.onBackground, c.background) >= 7.0)
            assertTrue("$palette muted text contrast ${contrast(c.onSurfaceVariant, c.background)}",
                contrast(c.onSurfaceVariant, c.background) >= 4.5)
            assertTrue("$palette accent contrast ${contrast(c.primary, c.background)}",
                contrast(c.primary, c.background) >= 4.5)
            assertTrue("$palette ink on accent ${contrast(c.onPrimary, c.primary)}",
                contrast(c.onPrimary, c.primary) >= 4.5)
        }
    }

    @Test
    fun theAccentIsThePalettesOwnAccent() {
        // Kiln used to be an amber theme and its TV accent was forced to amber; when it became Tide
        // (a teal palette) that override was left in place, so a teal theme showed up orange on TV.
        val tide = tv(AppPalette.Tide).primary
        assertTrue("Tide's TV accent is not teal: $tide", tide.green > tide.red && tide.blue > tide.red)
    }

    @Test
    fun aLightPaletteIsPushedMuchFurtherIntoTheDark() {
        // Paper is a light theme. Darkening it by the same amount as a dark theme would leave a
        // mid-grey wall behind the artwork, which is unreadable in a dark room.
        val paper = tv(AppPalette.Paper)
        val ember = tv(AppPalette.Ember)
        assertTrue("Paper is not dark on TV: ${paper.background}", paper.background.luminance() < 0.05f)
        assertTrue("Paper's TV background should not match its phone paper",
            paper.background.luminance() < paper.surface.luminance())
        assertTrue("Ember should still be dark", ember.background.luminance() < 0.05f)
    }
}
