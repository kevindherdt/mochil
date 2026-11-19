package net.kdhmedia.mochil.playback

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class SleepTimerTest {
    @Test fun `expiry clears buffered playback intent even when not playing`() {
        var expired = true
        var playWhenReady = true
        var ready = false
        var pauses = 0
        val consume = { expired.also { expired = false } }
        val pause = { pauses++; playWhenReady = false }

        assertFalse(ready && playWhenReady)
        pauseForExpiredSleepTimer(consume, pause)
        ready = true
        assertFalse(ready && playWhenReady)
        assertEquals(1, pauses)

        // Once consumed, a later explicit play is not paused by the old timer.
        playWhenReady = true
        pauseForExpiredSleepTimer(consume, pause)
        assertTrue(playWhenReady)
        assertEquals(1, pauses)
    }

    @Test fun `unexpired timer does not pause and expiry is safe while already paused`() {
        var pauses = 0
        pauseForExpiredSleepTimer({ false }) { pauses++ }
        assertEquals(0, pauses)
        pauseForExpiredSleepTimer({ true }) { pauses++ }
        assertEquals(1, pauses)
    }

    @Test fun spokenPlaybackSpeedCyclesThroughBoundedChoices() {
        assertEquals(1.25f, nextSpokenPlaybackSpeed(1f))
        assertEquals(1f, nextSpokenPlaybackSpeed(2f))
        assertEquals(1f, nextSpokenPlaybackSpeed(1.1f))
    }
}
