package net.kdhmedia.mochil.playback

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AudioSeekOriginTest {
    @Test fun `matching server resume is passive and consumed once`() {
        val origin = AudioSeekOrigin()
        assertFalse(origin.isExplicit(45_000, "resume-1", 45_300))
        assertTrue(origin.isExplicit(45_000, "resume-1", 45_300))
    }

    @Test fun `different or unmarked seeks remain explicit`() {
        val origin = AudioSeekOrigin()
        assertTrue(origin.isExplicit(45_000, "resume-2", 12_000))
        assertTrue(origin.isExplicit(-1, "", 45_000))
    }
}
