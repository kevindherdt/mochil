package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class SeriesNumberLabelTest {
    @Test
    fun wholeNumbersLoseTheirDecimal() {
        assertEquals("3", seriesNumberLabel(3.0))
        assertEquals("3.5", seriesNumberLabel(3.5))
        assertNull(seriesNumberLabel(null))
    }
}
