package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class TvAlphabetSelectionTest {
    @Test fun `selecting the active bucket clears the filter`() {
        assertEquals("", toggledTvBucket("M", "M"))
    }

    @Test fun `selecting a different bucket replaces the filter`() {
        assertEquals("M", toggledTvBucket("A", "M"))
        assertEquals("#", toggledTvBucket("", "#"))
    }
}
