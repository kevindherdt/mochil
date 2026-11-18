package net.kdhmedia.mochil.ui

import androidx.compose.ui.graphics.Color
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class MediaKindStyleTest {
    private val kinds = listOf("books", "lightnovels", "manga", "comics", "movies", "tv", "anime", "music", "audiobooks", "podcasts")

    @Test
    fun monochromePaletteKeepsTheDistinctIconsAndLabelsButUsesTheRequestedInkColor() {
        val monochrome = Color(0xFF111111)
        kinds.forEach { kind ->
            val regular = mediaKindStyle(kind)
            val eink = mediaKindStyle(kind, monochrome)
            assertEquals(monochrome, eink.color)
            assertEquals(regular.icon, eink.icon)
            assertEquals(regular.label, eink.label)
        }
    }

    @Test
    fun everyKindOfLibraryHasItsOwnColour() {
        assertEquals(kinds.size, kinds.map { mediaKindStyle(it).color }.toSet().size)
    }

    @Test
    fun unknownAndMixedLibrariesFallBackToTheBookStyle() {
        assertEquals(mediaKindStyle("books"), mediaKindStyle("calibre"))
        assertEquals(mediaKindStyle("books"), mediaKindStyle("mixed"))
        assertNotEquals(mediaKindStyle("books"), mediaKindStyle("manga"))
    }
}
