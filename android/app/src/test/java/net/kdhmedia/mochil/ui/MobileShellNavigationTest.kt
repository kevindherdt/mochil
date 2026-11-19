package net.kdhmedia.mochil.ui

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MobileShellNavigationTest {
    @Test fun `Home delegates system Back while every opened root is handled`() {
        assertFalse(handlesMobileShellBack(MobileShellDestination.Home))
        MobileShellDestination.entries.filterNot { it == MobileShellDestination.Home }.forEach {
            assertTrue(handlesMobileShellBack(it))
        }
    }

    @Test fun `auxiliary roots return to the primary destination that opened them`() {
        for (auxiliary in listOf(MobileShellDestination.Collections, MobileShellDestination.Sources, MobileShellDestination.Settings)) {
            assertTrue(mobileShellBackTarget(auxiliary, MobileShellDestination.Library) == MobileShellDestination.Library)
            assertTrue(mobileShellBackTarget(auxiliary, MobileShellDestination.Home) == MobileShellDestination.Home)
        }
        assertTrue(mobileShellBackTarget(MobileShellDestination.Library, MobileShellDestination.Library) == MobileShellDestination.Home)
    }

    @Test fun `only Home and Library can become an auxiliary return destination`() {
        assertTrue(mobileShellSettingsBack(MobileShellDestination.Collections, MobileShellDestination.Library) == MobileShellDestination.Collections)
        assertTrue(mobileShellSettingsBack(MobileShellDestination.Library, MobileShellDestination.Library) == MobileShellDestination.Library)
        assertTrue(mobileShellSettingsBack(MobileShellDestination.Home, MobileShellDestination.Home) == MobileShellDestination.Home)
        assertTrue(mobileShellAuxiliaryReturn(MobileShellDestination.Library, MobileShellDestination.Home) == MobileShellDestination.Library)
        assertTrue(mobileShellAuxiliaryReturn(MobileShellDestination.Home, MobileShellDestination.Library) == MobileShellDestination.Home)
        assertTrue(mobileShellAuxiliaryReturn(MobileShellDestination.Settings, MobileShellDestination.Library) == MobileShellDestination.Library)
        assertTrue(mobileShellAuxiliaryReturn(MobileShellDestination.Settings, MobileShellDestination.Sources) == MobileShellDestination.Home)
    }
}
