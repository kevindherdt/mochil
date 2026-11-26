package net.kdhmedia.mochil.ui.theme

import androidx.compose.ui.graphics.Color
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

/**
 * The web paints a card border with `--color-line` and the text on an accent fill with `--color-accent-ink`. Compose
 * had no role for either, so the first settled for Material's default grey and the second for the page colour. The kit
 * reads them from outlineVariant and onPrimary, and these pin that each palette and mode gets its own token value.
 */
class MochilSchemeRolesTest {
    private fun tokens(palette: AppPalette, dark: Boolean): GeneratedTokens.Mode = when (palette) {
        AppPalette.Ember -> if (dark) GeneratedTokens.Mochil.Dark else GeneratedTokens.Mochil.Light
        AppPalette.Tide -> if (dark) GeneratedTokens.Kiln.Dark else GeneratedTokens.Kiln.Light
        AppPalette.MagentaCream -> if (dark) GeneratedTokens.MagentaCream.Dark else GeneratedTokens.MagentaCream.Light
        AppPalette.EInk -> if (dark) GeneratedTokens.Eink.Dark else GeneratedTokens.Eink.Light
    }

    @Test fun `outlineVariant is the line token in every palette and mode`() {
        for (palette in AppPalette.entries) for (dark in listOf(true, false)) {
            assertEquals("$palette dark=$dark", Color(tokens(palette, dark).line), paletteScheme(palette, dark).outlineVariant)
        }
    }

    @Test fun `onPrimary is the accent ink token in every palette and mode`() {
        for (palette in AppPalette.entries) for (dark in listOf(true, false)) {
            assertEquals("$palette dark=$dark", Color(tokens(palette, dark).accentInk), paletteScheme(palette, dark).onPrimary)
        }
    }

    @Test fun `a swatch shows the page colour of its palette, which is not the accent`() {
        for (palette in AppPalette.entries) for (dark in listOf(true, false)) {
            assertEquals("$palette dark=$dark", Color(tokens(palette, dark).bg), palette.backdrop(dark))
            assertNotEquals("$palette dark=$dark", palette.backdrop(dark), palette.swatch(dark))
        }
    }
}
