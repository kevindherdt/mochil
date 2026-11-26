package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MobileShellNavigationTest {
    @Test fun `the app opens on Library like the web`() {
        assertEquals(MobileShellDestination.Library, MOBILE_SHELL_START)
    }

    @Test fun `Library delegates system Back and every other root handles it`() {
        assertFalse(handlesMobileShellBack(MobileShellDestination.Library))
        MobileShellDestination.entries.filterNot { it == MobileShellDestination.Library }.forEach {
            assertTrue("$it handles Back", handlesMobileShellBack(it))
        }
    }

    @Test fun `Back is handled from Library while a vault is open`() {
        assertTrue(handlesMobileShellBack(MobileShellDestination.Library, vaultOpen = true))
        assertFalse(handlesMobileShellBack(MobileShellDestination.Library, vaultOpen = false))
    }

    @Test fun `Back from any other root goes to Library, not to where it was opened from`() {
        MobileShellDestination.entries.filterNot { it == MobileShellDestination.Library }.forEach {
            assertEquals("Back from $it", MobileShellDestination.Library, mobileShellBackTarget(it))
        }
    }
}
