package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.InputMode
import androidx.compose.ui.platform.LocalInputModeManager
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performKeyInput
import androidx.compose.ui.test.pressKey
import androidx.compose.ui.unit.dp
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/**
 * The description overlay covers the page but shares its window, so its focus stops sit next to the page's own controls.
 * These tests pin the reported regression: a bare arrow walked the remote into the page underneath while the text was
 * still open, so Close could not be reached and Enter activated whatever sat below the description.
 *
 * Every assertion is behavioural rather than a focus-semantics check: the covered page control records whether it was
 * activated, and Enter only dismisses through the overlay's Close stop, so exactly one of those can happen.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w960dp-h540dp-mdpi")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class TvSynopsisOverlayFocusTest {
    @get:Rule val compose = createComposeRule()

    private val longStory = buildString {
        repeat(90) { append("The rain keeps falling on the empty street while nobody answers the door. ") }
    }

    private class Page {
        var played = false
        var dismissed = false
    }

    private fun showOverlay(): Page {
        val page = Page()
        compose.setContent {
            // A TV is always in keyboard/D-pad input mode; in touch mode Compose does not make clickables focusable at all.
            val inputMode = LocalInputModeManager.current
            LaunchedEffect(Unit) { inputMode.requestInputMode(InputMode.Keyboard) }
            MaterialTheme {
                Box(Modifier.size(900.dp, 500.dp).background(Color.Black)) {
                    // What sits under the description on a real detail page: activating it here means the remote escaped.
                    Box(
                        Modifier.align(Alignment.Center).size(120.dp).background(Color.DarkGray, CircleShape)
                            .clickable { page.played = true }.testTag("page-play"),
                    )
                    TvSynopsisOverlay(
                        title = "A long film",
                        story = longStory,
                        onDismiss = { page.dismissed = true },
                    )
                }
            }
        }
        compose.waitForIdle()
        return page
    }

    /** Keys go to whichever stop holds focus, the way the remote's do; the tag node is only a stable handle. */
    private fun press(key: Key) {
        compose.onNodeWithTag("tv-synopsis-text", useUnmergedTree = true).performKeyInput { pressKey(key) }
        compose.waitForIdle()
    }

    @Test
    fun theFirstDownScrollsTheTextInsteadOfLeavingIt() {
        val page = showOverlay()
        press(Key.DirectionDown)
        press(Key.Enter)
        assertFalse("a single Down must scroll the description, not hand the remote over", page.dismissed)
        assertFalse("the description never activates the covered page", page.played)
    }

    @Test
    fun readingToTheEndHandsTheRemoteToCloseWhichThenDismisses() {
        val page = showOverlay()
        repeat(40) { press(Key.DirectionDown) }
        press(Key.Enter)
        assertTrue("Enter after the text must dismiss through Close", page.dismissed)
        assertFalse("the page under the description must never be activated", page.played)
    }

    @Test
    fun directionsNeverLeaveTheOverlay() {
        val page = showOverlay()
        listOf(Key.DirectionDown, Key.DirectionUp, Key.DirectionLeft, Key.DirectionRight).forEach { key ->
            repeat(6) { press(key) }
        }
        // However the mashing left the two stops, reading to the end must still reach Close rather than the page.
        repeat(40) { press(Key.DirectionDown) }
        press(Key.Enter)
        assertTrue("focus left the overlay: Enter did not reach Close", page.dismissed)
        assertFalse("focus escaped to the page under the description", page.played)
    }

    @Test
    fun rightMovesFromTheTextToClose() {
        val page = showOverlay()
        press(Key.DirectionRight)
        press(Key.Enter)
        assertTrue("Right from the text must reach Close", page.dismissed)
        assertFalse("the covered page must stay untouched", page.played)
    }

    @Test
    fun leftReturnsFromCloseToTheText() {
        val page = showOverlay()
        press(Key.DirectionRight)
        press(Key.DirectionLeft)
        press(Key.Enter)
        assertFalse("Left must return to the text, where Enter closes nothing", page.dismissed)
        assertFalse(page.played)
    }
}
