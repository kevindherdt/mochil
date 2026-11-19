package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class MobileLibraryRailLayoutTest {
    @Test fun `overflow affordance cannot expand the rail into catalogue space`() {
        assertEquals(48f, MOBILE_LIBRARY_RAIL_HEIGHT.value)
    }
}
