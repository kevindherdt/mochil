package net.kdhmedia.mochil.ui

import androidx.compose.foundation.focusable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.InputMode
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.platform.LocalInputModeManager
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performKeyInput
import androidx.compose.ui.test.performSemanticsAction
import androidx.compose.ui.test.pressKey
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.ui.theme.TvSettingsTypography
import net.kdhmedia.mochil.update.ReleaseNotes
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/**
 * The changelog is its own bounded panel, not a column of bullets in the page flow: it has to be one focus stop that the
 * remote scrolls, and the controls below it have to stay reachable once its end is reached.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w960dp-h540dp-mdpi")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class TvReleaseNotesPanelTest {
    @get:Rule val compose = createComposeRule()

    private val notes = ReleaseNotes(
        newFeatures = List(12) { "A new capability that needs a full line to describe it, number $it." },
        fixes = List(12) { "A fixed defect with a sentence long enough to wrap, number $it." },
        performance = List(6) { "Faster scrolling while a long library is loading, entry $it." },
        changes = List(6) { "A changed behaviour worth reading before updating, entry $it." },
    )

    private fun mount() {
        compose.setContent {
            // A TV is always in keyboard/D-pad input mode; touch mode does not make clickables focusable at all.
            val inputMode = LocalInputModeManager.current
            LaunchedEffect(Unit) { inputMode.requestInputMode(InputMode.Keyboard) }
            MaterialTheme(typography = TvSettingsTypography) {
                Box(Modifier.size(600.dp, 400.dp)) {
                    Column {
                        Box(Modifier.size(40.dp).focusable().testTag("above"))
                        TvReleaseNotes(notes)
                        Box(Modifier.size(40.dp).focusable().testTag("below"))
                    }
                }
            }
        }
        compose.waitForIdle()
    }

    private fun press(key: Key) {
        compose.onNodeWithTag("tv-release-notes", useUnmergedTree = true).performKeyInput { pressKey(key) }
        compose.waitForIdle()
    }

    @Test
    fun thePanelIsOneFocusStopTheRemoteScrollsBeforeLeavingIt() {
        mount()
        compose.onNodeWithTag("above", useUnmergedTree = true).performSemanticsAction(SemanticsActions.RequestFocus)
        compose.waitForIdle()
        compose.onNodeWithTag("above", useUnmergedTree = true).performKeyInput { pressKey(Key.DirectionDown) }
        compose.waitForIdle()
        compose.onNodeWithTag("tv-release-notes", useUnmergedTree = true).assertIsFocused()

        // Reading the whole changelog must not walk the remote out of the panel on the first press.
        press(Key.DirectionDown)
        compose.onNodeWithTag("tv-release-notes", useUnmergedTree = true).assertIsFocused()

        // Once its end is reached the next press continues to the control after it.
        repeat(30) { press(Key.DirectionDown) }
        compose.onNodeWithTag("below", useUnmergedTree = true).assertIsFocused()
    }

    @Test
    fun aChangelogWithNoEntriesTakesNoSpace() {
        compose.setContent {
            MaterialTheme(typography = TvSettingsTypography) {
                Box(Modifier.size(600.dp, 400.dp)) {
                    TvReleaseNotes(ReleaseNotes())
                    Text("after", Modifier.testTag("after"))
                }
            }
        }
        compose.waitForIdle()
        compose.onNodeWithTag("tv-release-notes", useUnmergedTree = true).assertDoesNotExist()
    }
}
