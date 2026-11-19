package net.kdhmedia.mochil.ui

import android.view.KeyEvent
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class TvReadinessCheckTest {
    @Test
    fun mapsRemoteAndControllerKeysWithoutClaimingUnrelatedKeys() {
        assertEquals(TvRemoteAction.Up, tvRemoteAction(KeyEvent.KEYCODE_DPAD_UP))
        assertEquals(TvRemoteAction.Select, tvRemoteAction(KeyEvent.KEYCODE_DPAD_CENTER))
        assertEquals(TvRemoteAction.Select, tvRemoteAction(KeyEvent.KEYCODE_BUTTON_A))
        assertEquals(TvRemoteAction.PlayPause, tvRemoteAction(KeyEvent.KEYCODE_MEDIA_PLAY_PAUSE))
        assertNull(tvRemoteAction(KeyEvent.KEYCODE_VOLUME_UP))
        assertNull(tvRemoteAction(KeyEvent.KEYCODE_BACK))
    }

    @Test
    fun stateOnlyRecordsKeysDuringAnActiveCheck() {
        val state = TvReadinessState()
        state.observe(KeyEvent.KEYCODE_DPAD_LEFT)
        assertEquals(emptySet<TvRemoteAction>(), state.observed)

        state.start()
        state.observe(KeyEvent.KEYCODE_DPAD_LEFT)
        state.observe(KeyEvent.KEYCODE_DPAD_LEFT)
        assertEquals(setOf(TvRemoteAction.Left), state.observed)

        state.stop()
        state.observe(KeyEvent.KEYCODE_DPAD_RIGHT)
        assertEquals(setOf(TvRemoteAction.Left), state.observed)
    }
}
