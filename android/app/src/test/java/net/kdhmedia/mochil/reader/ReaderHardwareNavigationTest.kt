package net.kdhmedia.mochil.reader

import android.view.KeyEvent
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ReaderHardwareNavigationTest {
    @Test fun pageKeysAlwaysTurnPages() {
        assertEquals(ReaderPageTurn.Previous, readerPageTurn(KeyEvent.KEYCODE_PAGE_UP, false))
        assertEquals(ReaderPageTurn.Next, readerPageTurn(KeyEvent.KEYCODE_PAGE_DOWN, false))
        assertEquals(ReaderPageTurn.Previous, readerPageTurn(KeyEvent.KEYCODE_DPAD_LEFT, false))
        assertEquals(ReaderPageTurn.Next, readerPageTurn(KeyEvent.KEYCODE_DPAD_RIGHT, false))
        assertEquals(ReaderPageTurn.Previous, readerPageTurn(KeyEvent.KEYCODE_BUTTON_L1, false))
        assertEquals(ReaderPageTurn.Next, readerPageTurn(KeyEvent.KEYCODE_BUTTON_R1, false))
    }

    @Test fun volumeKeysRequireTheExplicitReaderSetting() {
        assertNull(readerPageTurn(KeyEvent.KEYCODE_VOLUME_DOWN, false))
        assertEquals(ReaderPageTurn.Previous, readerPageTurn(KeyEvent.KEYCODE_VOLUME_UP, true))
        assertEquals(ReaderPageTurn.Next, readerPageTurn(KeyEvent.KEYCODE_VOLUME_DOWN, true))
    }

    @Test fun unrelatedKeysRemainAvailableToTheSystem() {
        assertNull(readerPageTurn(KeyEvent.KEYCODE_BACK, true))
    }

    @Test fun aRemoteOpensTheControlsWithoutLosingItsPageTurns() {
        for (key in listOf(KeyEvent.KEYCODE_DPAD_CENTER, KeyEvent.KEYCODE_ENTER, KeyEvent.KEYCODE_DPAD_UP, KeyEvent.KEYCODE_DPAD_DOWN, KeyEvent.KEYCODE_MENU)) {
            assertTrue(opensReaderControls(key))
            assertNull(readerPageTurn(key, false))
        }
        assertFalse(opensReaderControls(KeyEvent.KEYCODE_DPAD_LEFT))
        assertFalse(opensReaderControls(KeyEvent.KEYCODE_DPAD_RIGHT))
        assertFalse(opensReaderControls(KeyEvent.KEYCODE_BACK))
        assertTrue(isFocusKey(KeyEvent.KEYCODE_DPAD_LEFT))
        assertFalse(isFocusKey(KeyEvent.KEYCODE_PAGE_DOWN))
    }
}
