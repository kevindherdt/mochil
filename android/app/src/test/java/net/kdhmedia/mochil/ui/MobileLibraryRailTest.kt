package net.kdhmedia.mochil.ui

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MobileLibraryRailTest {
    @Test fun homeAndLibraryCanNeverBothBeSelected() {
        assertFalse(isRailLibrarySelected(MobileShellDestination.Discover, "books", "books"))
        assertTrue(isRailLibrarySelected(MobileShellDestination.Library, "books", "books"))
        assertFalse(isRailLibrarySelected(MobileShellDestination.Library, "books", "manga"))
        assertFalse(isRailLibrarySelected(MobileShellDestination.Settings, "books", "books"))
    }
}
