package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class MusicTrackRowsTest {
    @Test fun formatsTrackAndLongFormDurations() {
        assertEquals("0:00", formatTrackDuration(0.0))
        assertEquals("3:50", formatTrackDuration(230.0))
        assertEquals("1:02:03", formatTrackDuration(3723.0))
    }
}
