package net.kdhmedia.mochil.ui

import androidx.compose.ui.unit.dp
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * The TV poster wall put six covers across a 1080p screen at 118dp each. Four is the shape a couch can read,
 * and this pins both the number and the columns it produces on the densities a television reports.
 */
class TvCardWidthTest {
    @Test
    fun `a poster wall is wider than a shelf of stills`() {
        assertEquals(176.dp, tvCardMinWidth(posterShelf = true))
        assertEquals(210.dp, tvCardMinWidth(posterShelf = false))
    }

    @Test
    fun `1080p at television density gives four posters across`() {
        // 1920px at 320dpi is 960dp wide; the grid pads 40dp a side and leaves 16dp between cards.
        assertEquals(4, tvGridColumns(widthDp = 960f, minColumnWidth = tvCardMinWidth(true)))
    }

    @Test
    fun `a wider logical screen does not collapse into a contact sheet`() {
        // 1920px at 213dpi is 1442dp wide: seven posters, where the old 118dp managed ten.
        assertEquals(7, tvGridColumns(widthDp = 1442f, minColumnWidth = tvCardMinWidth(true)))
        assertEquals(10, tvGridColumns(widthDp = 1442f, minColumnWidth = 118.dp))
    }

    @Test
    fun `a shelf of stills keeps fewer, larger cards at television density`() {
        assertEquals(3, tvGridColumns(widthDp = 960f, minColumnWidth = tvCardMinWidth(false)))
    }
}
