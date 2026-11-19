package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class AudioGroupIdentityTest {
    @Test fun `opaque ids keep same named albums distinct`() {
        val first = group(id = "album:first")
        val second = group(id = "album:second")

        assertNotEquals(first.stableKey(), second.stableKey())
        assertFalse(first.hasSameIdentity(second))
    }

    @Test fun `matching opaque ids survive changed display metadata`() {
        val selected = group(subtitle = "Various artists", id = "album:shared")
        val refreshed = group(subtitle = "Paramore", id = "album:shared")

        assertTrue(selected.hasSameIdentity(refreshed))
    }

    @Test fun `legacy groups have a deterministic composite fallback`() {
        val first = group()
        val second = group()

        assertEquals(first.stableKey(), second.stableKey())
        assertTrue(first.hasSameIdentity(second))
    }

    private fun group(subtitle: String = "Artist", id: String = "") =
        AudioGroup("Greatest Hits", subtitle, 12, "cover", "music", id)
}
