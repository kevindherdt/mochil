package net.kdhmedia.mochil.ui

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ExpandedMobileNavigationTest {
    @Test fun `expanded navigation follows window width rather than device identity`() {
        assertFalse(usesExpandedMobileNavigation(320))
        assertFalse(usesExpandedMobileNavigation(599))
        assertFalse(usesExpandedMobileNavigation(839))
        assertTrue(usesExpandedMobileNavigation(840))
        assertTrue(usesExpandedMobileNavigation(1_280))
    }
}
