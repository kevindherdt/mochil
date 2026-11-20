package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * The description overlay answers its own arrows because it shares a window with the page underneath, so the order of
 * those answers is plain logic and is pinned here: read the text first, hand over to Close at the end, and never let a
 * direction press leave the overlay.
 */
class TvSynopsisNavigationTest {
    private fun move(
        action: TvRemoteAction,
        closeFocused: Boolean = false,
        canScrollBack: Boolean = false,
        canScrollForward: Boolean = false,
    ) = tvSynopsisMove(action, closeFocused, canScrollBack, canScrollForward)

    @Test
    fun downReadsTheWholeTextBeforeCloseGetsTheRemote() {
        assertEquals(TvSynopsisMove.ScrollForward, move(TvRemoteAction.Down, canScrollForward = true))
        assertEquals(TvSynopsisMove.ToClose, move(TvRemoteAction.Down, canScrollForward = false))
        // Nothing below Close, and it must not scroll the text behind the remote's back either.
        assertNull(move(TvRemoteAction.Down, closeFocused = true, canScrollForward = true))
    }

    @Test
    fun upReturnsToTheTextFromCloseAndStopsAtTheTop() {
        assertEquals(TvSynopsisMove.ToStory, move(TvRemoteAction.Up, closeFocused = true))
        assertEquals(TvSynopsisMove.ScrollBack, move(TvRemoteAction.Up, canScrollBack = true))
        assertNull("nothing above the first line", move(TvRemoteAction.Up, canScrollBack = false))
    }

    @Test
    fun leftAndRightCrossBetweenTheTextAndCloseInBothDirections() {
        assertEquals(TvSynopsisMove.ToClose, move(TvRemoteAction.Right))
        assertEquals(TvSynopsisMove.ToClose, move(TvRemoteAction.Left))
        assertEquals(TvSynopsisMove.ToStory, move(TvRemoteAction.Right, closeFocused = true))
        assertEquals(TvSynopsisMove.ToStory, move(TvRemoteAction.Left, closeFocused = true))
    }

    @Test
    fun selectAndPlayPauseAreLeftToTheFocusedControl() {
        assertNull(move(TvRemoteAction.Select))
        assertNull(move(TvRemoteAction.PlayPause))
    }
}
