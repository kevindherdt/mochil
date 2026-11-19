package net.kdhmedia.mochil.ui

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.luminance
import net.kdhmedia.mochil.catalogue.LibraryKind
import org.junit.Assert.assertTrue
import org.junit.Test

/** The kind icons are the only label a library row has, so their colour has to stay readable on both
 * surface families: a pastel tuned for a dark surface measures barely above 1:1 on a near-white one. */
class MediaKindContrastTest {

    private fun contrast(foreground: Color, background: Color): Double {
        val light = maxOf(foreground.luminance(), background.luminance()).toDouble()
        val dark = minOf(foreground.luminance(), background.luminance()).toDouble()
        return (light + 0.05) / (dark + 0.05)
    }

    // The lightest surface a light palette can show (MaterialLight / Oat / ManjaroLight) and the
    // lightest dark-palette surface (TwilightDark), so the test covers the worst case in each mode.
    private val worstLightSurface = Color(0xFFFFFFFF)
    private val worstDarkSurface = Color(0xFF2C2A40)

    @Test
    fun everyKindIconStaysReadableOnLightAndDarkSurfaces() {
        for (kind in LibraryKind.entries) {
            val style = mediaKindStyle(kind.wire)
            val onLight = contrast(style.onLight, worstLightSurface)
            val onDark = contrast(style.color, worstDarkSurface)
            assertTrue(
                "${style.label} on a light surface is ${"%.2f".format(onLight)}:1, needs 3:1",
                onLight >= 3.0,
            )
            assertTrue(
                "${style.label} on a dark surface is ${"%.2f".format(onDark)}:1, needs 3:1",
                onDark >= 3.0,
            )
        }
    }

    @Test
    fun tintPicksTheTwinMatchingTheSurface() {
        val style = mediaKindStyle("books")
        assertTrue("a light surface must use the dark twin", style.tint(Color(0xFFFFFFFF)) == style.onLight)
        assertTrue("a dark surface must keep the pastel", style.tint(Color(0xFF171513)) == style.color)
    }
}
