package net.kdhmedia.mochil.ui

import androidx.compose.ui.unit.dp
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * A television hides a few percent of the picture around its edges, and the design specification puts it at
 * 5% of the short side. A fixed 24dp margin (2.2% of 1080p) left content inside the hidden band, which is how
 * a rail's first letter disappeared on a real set.
 */
class TvOverscanTest {
    @Test
    fun `a 1080p panel clears five percent of its short side`() {
        assertEquals(54.dp, tvOverscan(1920f, 1080f))
    }

    @Test
    fun `a 4k panel clears twice as much`() {
        assertEquals(108.dp, tvOverscan(3840f, 2160f))
    }

    @Test
    fun `the short side decides, whichever way the panel is turned`() {
        assertEquals(54.dp, tvOverscan(1080f, 1920f))
    }

    @Test
    fun `a small window keeps the floor rather than a sliver`() {
        assertEquals(24.dp, tvOverscan(400f, 300f))
    }
}
