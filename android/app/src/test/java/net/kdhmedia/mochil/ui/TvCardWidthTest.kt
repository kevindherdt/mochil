package net.kdhmedia.mochil.ui

import androidx.compose.ui.unit.dp
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * The TV poster wall has been sized twice against a real television: six across at 118dp read as a contact sheet, four at
 * 176dp read as enormous, and six is where it settled. This pins the number and the columns it produces on the densities a
 * television reports, so the next change to it is a decision rather than a drift.
 */
class TvCardWidthTest {
    @Test
    fun `a poster wall is wider than a shelf of stills`() {
        assertEquals(132.dp, tvCardMinWidth(posterShelf = true))
        assertEquals(208.dp, tvCardMinWidth(posterShelf = false))
    }

    @Test
    fun `1080p at television density gives six posters across`() {
        // 1920px at 320dpi is 960dp wide; the grid pads 40dp a side and leaves 16dp between cards.
        assertEquals(6, tvGridColumns(widthDp = 960f, minColumnWidth = tvCardMinWidth(true)))
    }

    @Test
    fun `a wider logical screen does not collapse into a contact sheet`() {
        // 1920px at 213dpi is 1442dp wide: nine posters, where 118dp managed ten and 176dp only seven.
        assertEquals(9, tvGridColumns(widthDp = 1442f, minColumnWidth = tvCardMinWidth(true)))
        assertEquals(4, tvGridColumns(widthDp = 960f, minColumnWidth = 176.dp))
    }

    @Test
    fun `a shelf of stills keeps fewer, larger cards at television density`() {
        assertEquals(4, tvGridColumns(widthDp = 960f, minColumnWidth = tvCardMinWidth(false)))
    }
}
