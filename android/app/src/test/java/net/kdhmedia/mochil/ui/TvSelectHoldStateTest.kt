package net.kdhmedia.mochil.ui

import android.view.KeyEvent
import org.junit.Assert.assertEquals
import org.junit.Test

class TvSelectHoldStateTest {
    @Test
    fun heldSelectTriggersOnceAndConsumesItsRelease() {
        val state = TvSelectHoldState()

        assertEquals(TvHoldResult.Pass, state.handle(true, KeyEvent.ACTION_DOWN, 0, false))
        assertEquals(TvHoldResult.Trigger, state.handle(true, KeyEvent.ACTION_DOWN, 1, true))
        assertEquals(TvHoldResult.Consume, state.handle(true, KeyEvent.ACTION_DOWN, 2, true))
        assertEquals(TvHoldResult.Consume, state.handle(true, KeyEvent.ACTION_UP, 0, false))
    }

    @Test
    fun ordinarySelectAndUnrelatedKeysRemainAvailableToCards() {
        val state = TvSelectHoldState()

        assertEquals(TvHoldResult.Pass, state.handle(true, KeyEvent.ACTION_DOWN, 0, false))
        assertEquals(TvHoldResult.Pass, state.handle(true, KeyEvent.ACTION_UP, 0, false))
        assertEquals(TvHoldResult.Pass, state.handle(false, KeyEvent.ACTION_DOWN, 4, true))
    }

    @Test
    fun contextMenuConsumesOpeningHoldUntilSelectIsReleased() {
        val guard = TvContextMenuInputGuard()

        assertEquals(true, guard.consume(select = true, action = KeyEvent.ACTION_DOWN))
        assertEquals(true, guard.consume(select = true, action = KeyEvent.ACTION_DOWN))
        assertEquals(false, guard.consume(select = false, action = KeyEvent.ACTION_DOWN))
        assertEquals(true, guard.consume(select = true, action = KeyEvent.ACTION_UP))
        assertEquals(false, guard.consume(select = true, action = KeyEvent.ACTION_DOWN))
        assertEquals(false, guard.consume(select = true, action = KeyEvent.ACTION_UP))
    }


    @Test
    fun popupReleaseOwnershipDoesNotPoisonTheNextOrdinaryClick() {
        val state = TvSelectHoldState()

        assertEquals(TvHoldResult.Trigger, state.handle(true, KeyEvent.ACTION_DOWN, 1, true))
        assertEquals(TvHoldResult.Pass, state.handle(true, KeyEvent.ACTION_DOWN, 0, false))
        assertEquals(TvHoldResult.Pass, state.handle(true, KeyEvent.ACTION_UP, 0, false))
    }

    @Test
    fun theReleaseThatEndsAHoldNeverCountsAsAClick() {
        val state = TvSelectHoldState()

        assertEquals(TvHoldResult.Trigger, state.handle(true, KeyEvent.ACTION_DOWN, 1, true))
        assertEquals(TvHoldResult.Consume, state.handle(true, KeyEvent.ACTION_DOWN, 2, true))
        assertEquals(TvHoldResult.Consume, state.handle(true, KeyEvent.ACTION_UP, 0, false))
    }
}
