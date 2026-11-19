package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class TvFocusDetailTest {
    @Test fun `focus details expose the catalogue letter for tv navigation`() {
        assertEquals("R", TvFocusDetail("Rebuild World", "Volume 1").navigationBucket)
        assertEquals("A", tvNavigationBucket("  album"))
        assertEquals("#", tvNavigationBucket("86—Eighty Six"))
        assertEquals("#", tvNavigationBucket(""))
        assertEquals("#", tvNavigationBucket("Élan"))
        assertEquals("A", tvNavigationBucket("The Album"))
        assertEquals("R", tvNavigationBucket("  AN   Record  "))
        assertEquals("S", tvNavigationBucket("A Song"))
        assertEquals("#", tvNavigationBucket("The 86—Eighty Six"))
        assertEquals("#", tvNavigationBucket("The 日本語"))
        assertEquals("T", tvNavigationBucket("Theatre"))
    }
}
