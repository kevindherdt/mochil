package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.MusicGrouping
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class MusicDiscoveryPolicyTest {
    @Test fun discoveryRemainsBoundedAndAddressable() {
        assertEquals(5, DISCOVERY_VISIBLE_TRACKS)
        assertEquals("Discover", musicGroupingLabel(MusicGrouping.Discover, "music"))
        assertTrue(DISCOVERY_VISIBLE_TRACKS < 10)
    }
}
