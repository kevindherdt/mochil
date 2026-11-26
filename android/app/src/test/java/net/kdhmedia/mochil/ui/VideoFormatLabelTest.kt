package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/** The strip on a film's artwork comes from the server's probe, never from a guess (web `resolutionLabel`). */
class VideoFormatLabelTest {
    @Test
    fun `the usual ladder, rounded down`() {
        assertEquals("4K", resolutionLabel(2160))
        assertEquals("1080p", resolutionLabel(1080))
        assertEquals("1080p", resolutionLabel(1000))
        assertEquals("720p", resolutionLabel(800))
        assertEquals("720p", resolutionLabel(700))
        assertEquals("480p", resolutionLabel(480))
        assertEquals("360p", resolutionLabel(360))
    }

    @Test
    fun `no probe means no strip`() {
        assertNull(resolutionLabel(null))
        assertNull(resolutionLabel(0))
        assertNull(videoFormatLabel(null, true))
    }

    @Test
    fun `direct play is said only when the server says it`() {
        assertEquals("1080p · Direct Play", videoFormatLabel(1080, true))
        assertEquals("1080p", videoFormatLabel(1080, false))
    }
}
