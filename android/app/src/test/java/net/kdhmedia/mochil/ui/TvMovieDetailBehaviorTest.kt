package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performKeyInput
import androidx.compose.ui.test.performSemanticsAction
import androidx.compose.ui.test.pressKey
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.video.VideoItem
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w1280dp-h720dp")
class TvMovieDetailBehaviorTest {
    @get:Rule val compose = createComposeRule()

    @Test fun movieHeroShowsPosterAndKeepsRemotePlayAction() {
        var played = false
        val movie = VideoItem(id = "1", libraryId = "movies", title = "Example", show = "", season = 0, episode = 0, durationSeconds = 0.0, resumeSeconds = 0.0, artworkPath = "poster", streamPath = "", hlsPath = "", subtitles = emptyList())
        compose.setContent {
            MaterialTheme {
                Box(Modifier.size(1280.dp, 720.dp)) {
                    TvMovieDetail(movie, null, "https://example.invalid/poster.jpg", "", { played = true }, {})
                }
            }
        }
        compose.onNodeWithContentDescription("Artwork for Example").assertIsDisplayed()
        compose.onNodeWithText("Play").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.onNodeWithText("Play").assertIsFocused().performKeyInput { pressKey(Key.Enter) }
        compose.runOnIdle { assertTrue(played) }
    }
}
