package net.kdhmedia.mochil.dashboard

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import net.kdhmedia.mochil.catalogue.toBookSummary
import net.kdhmedia.mochil.ui.DashboardScreen
import org.json.JSONObject
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class DashboardRecoveryUiTest {
    @get:Rule val compose = createComposeRule()
    @Test fun tvExistingItemsRemainVisibleWhenRefreshFails() {
        val track = net.kdhmedia.mochil.audio.AudioTrack("id", "audio", "Chapter", "Artist", "Album", 100.0, libraryKind = "audiobooks")
        val state = DashboardState(listening = listOf(ContinueListening(track, 20.0, 100.0)), loading = false, error = "Could not load your activity.")
        compose.setContent {
            net.kdhmedia.mochil.ui.TvHomeScreen(state, { "" }, { "" }, { "" }, null, {}, {}, {}, {}, {}, {})
        }
        compose.onNodeWithText("Continue listening").assertIsDisplayed()
        compose.onNodeWithText("Could not load your activity.").assertIsDisplayed()
    }
    @Test fun existingItemsRemainVisibleWhenRefreshFails() {
        val book = JSONObject("""{"id":"book","library_id":"books","title":"Book","authors":[]}""").toBookSummary()
        val state = DashboardState(reading = listOf(ContinueReading(book, 0.5, 0)), loading = false, error = "Could not load your activity.")
        compose.setContent {
            DashboardScreen(state, { "" }, { "" }, { "" }, null, {}, {}, {}, {}, {}, {}, {}, {}, emptyList())
        }
        compose.onNodeWithText("Book").assertIsDisplayed()
        compose.onNodeWithText("Could not load your activity.").assertIsDisplayed()
    }
}
