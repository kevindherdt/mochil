package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import org.junit.Assert.assertTrue
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performSemanticsAction
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.isFocused
import androidx.compose.ui.test.performKeyInput
import androidx.compose.ui.test.pressKey
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoTitleDetails
import net.kdhmedia.mochil.video.VideoCastMember
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w800dp-h450dp")
class TvCompactDetailBehaviorTest {
    @get:Rule val compose = createComposeRule()
    @Test fun compactShowNavigatesFromEpisodeToCastAndBack() {
        val show = net.kdhmedia.mochil.video.VideoShow("Show", "Show", "show", 1, listOf(1), "", overview = "Story ".repeat(30))
        val episode = VideoItem("ep", "tv", "Episode One", "Show", 1, 1, 100.0, "", "stream", "", 0.0, emptyList(), overview = "Episode story ".repeat(30))
        val state = net.kdhmedia.mochil.video.VideoState(libraryId = "tv", libraryKind = "tv", items = listOf(episode), details = mapOf("show:Show" to VideoTitleDetails(cast = listOf(VideoCastMember("Actor One", "Hero")))))
        compose.setContent { MaterialTheme { Box(Modifier.size(800.dp, 450.dp)) {
            TvShowDetail(show, state, null, { it }, {}, {}, { _, _ -> }, {}, {})
        } } }
        compose.onNodeWithText("Play").performSemanticsAction(SemanticsActions.RequestFocus).performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Episode One").assertIsFocused().assertIsDisplayed()
        val story = compose.onNodeWithText("Episode story ".repeat(30)).getUnclippedBoundsInRoot()
        assertTrue("Overview clipped after episode focus: $story", story.top >= 64.dp)
        compose.onNodeWithText("Episode One").performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Actor One").assertIsFocused().assertIsDisplayed().performKeyInput { pressKey(Key.DirectionUp) }
        compose.onNodeWithText("Episode One").assertIsFocused().assertIsDisplayed()
    }

    @Test fun episodeDescriptionVisibleAfterCastReturnNormalFont() = probeBounds(1f)
    @Test fun episodeDescriptionVisibleAfterCastReturnLargeFont() = probeBounds(1.6f)
    private fun probeBounds(font: Float) {
        val show = net.kdhmedia.mochil.video.VideoShow("Show", "Show", "show", 1, listOf(1), "", overview = "Story ".repeat(30))
        val episode = VideoItem("ep", "tv", "Episode One", "Show", 1, 1, 100.0, "", "stream", "", 0.0, emptyList(), overview = "Episode story ".repeat(30))
        val short = episode.copy(id = "ep2", title = "Episode Two", overview = "Brief")
        val empty = episode.copy(id = "ep3", title = "Episode Three", overview = "")
        val state = net.kdhmedia.mochil.video.VideoState(libraryId = "tv", libraryKind = "tv", items = listOf(episode, short, empty), details = mapOf("show:Show" to VideoTitleDetails(cast = listOf(VideoCastMember("Actor One", "Hero")))))
        val scale = font
        compose.setContent { CompositionLocalProvider(LocalDensity provides Density(1f, scale)) { MaterialTheme { Box(Modifier.size(800.dp, 450.dp)) {
            TvShowDetail(show, state, null, { it }, {}, {}, { _, _ -> }, {}, {})
        } } } }
        run {
            compose.onNodeWithText("Play").performSemanticsAction(SemanticsActions.RequestFocus)
            compose.onNodeWithText("Play").performKeyInput { pressKey(Key.DirectionDown) }
            compose.onNodeWithText("Episode One").assertIsFocused()
            // Reported from the real set: browsing episodes scrolled the title and story out of
            // sight. Both now stay put, and the cast row is what sits below the fold.
            val heroTitle = compose.onNodeWithText("Show").getUnclippedBoundsInRoot()
            assertTrue("hero title scrolled away: $heroTitle", heroTitle.top >= 64.dp && heroTitle.bottom <= 430.dp)
            val focused = compose.onNodeWithText("Episode story ".repeat(30)).getUnclippedBoundsInRoot()
            assertTrue("focused story clipped: $focused", focused.top >= 64.dp && focused.bottom <= 430.dp)
            val firstTop = compose.onNodeWithText("Episode One").getUnclippedBoundsInRoot().top
            compose.onNodeWithText("Episode One").performKeyInput { pressKey(Key.DirectionRight) }
            compose.onNodeWithText("Episode Two").assertIsFocused()
            val brief = compose.onNodeWithText("Brief").getUnclippedBoundsInRoot()
            assertTrue("brief story clipped: $brief", brief.top >= 64.dp && brief.bottom <= 430.dp)
            val secondTop = compose.onNodeWithText("Episode Two").getUnclippedBoundsInRoot().top
            assertTrue("rail moved: $firstTop $secondTop", kotlin.math.abs((firstTop - secondTop).value) <= 4f)
            compose.onNodeWithText("Episode Two").performKeyInput { pressKey(Key.DirectionRight) }
            compose.onNodeWithText("Episode Three").assertIsFocused().performKeyInput { pressKey(Key.DirectionDown) }
            compose.onNodeWithText("Actor One").assertIsFocused().performKeyInput { pressKey(Key.DirectionUp) }
            // Up from the cast row re-enters the rail geometrically, so it lands on whichever card
            // sits above the actor rather than the one last focused. Whichever it is, its story has
            // to be fully visible and the rail must not have jumped to show it.
            val landedOn = focusedText()
            assertTrue("Up from cast left the episode rail: $landedOn", landedOn.startsWith("Episode"))
            val storyFor = { title: String ->
                when (title) {
                    "Episode One" -> "Episode story ".repeat(30)
                    "Episode Two" -> "Brief"
                    else -> "Story ".repeat(30)   // Episode Three has no synopsis, so the show's fills the slot.
                }
            }
            val returnedStory = compose.onNodeWithText(storyFor(landedOn)).getUnclippedBoundsInRoot()
            assertTrue("returned story clipped: $returnedStory", returnedStory.top >= 64.dp && returnedStory.bottom <= 430.dp)
            val returnedTop = compose.onNodeWithText(landedOn).getUnclippedBoundsInRoot().top
            assertTrue("rail jumped on return: $firstTop $returnedTop", kotlin.math.abs((firstTop - returnedTop).value) <= 4f)
            compose.onNodeWithText(landedOn).performKeyInput { pressKey(Key.DirectionRight) }
            val afterReturn = focusedText()
            val nextStory = compose.onNodeWithText(storyFor(afterReturn)).getUnclippedBoundsInRoot()
            assertTrue("story after return clipped: $afterReturn $nextStory", nextStory.top >= 64.dp && nextStory.bottom <= 430.dp)
        }
    }

    private fun focusedText(): String =
        compose.onNode(isFocused()).fetchSemanticsNode().config[SemanticsProperties.Text].first().text

    @Test fun compactMovieCanReachCastFromPlay() {
        val movie = VideoItem("id", "movies", "Movie", "", 0, 0, 0.0, "", "", "", 0.0, emptyList(), overview = "Story ".repeat(100))
        val details = VideoTitleDetails(cast = listOf(VideoCastMember("Actor One", "Hero")))
        compose.setContent { MaterialTheme { Box(Modifier.size(800.dp, 450.dp)) {
            TvMovieDetail(movie, null, "", "", {}, {}, details = details)
        } } }
        compose.onNodeWithText("Play").performSemanticsAction(SemanticsActions.RequestFocus).performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Actor One").assertIsFocused().assertIsDisplayed()
    }
    @Test fun longMovieDetailsKeepControlsVisibleOnCompactLargeFontScreen() {
        val movie = VideoItem(id = "1", libraryId = "movies", title = "An extremely long movie title ".repeat(5), show = "", season = 0, episode = 0, durationSeconds = 7000.0, resumeSeconds = 0.0, artworkPath = "", streamPath = "", hlsPath = "", subtitles = emptyList(), overview = "Long story ".repeat(100))
        val details = VideoTitleDetails(cast = List(10) { VideoCastMember("A lengthy performer name $it", "A lengthy character name") })
        var played = false
        var watched = false
        compose.setContent {
            CompositionLocalProvider(LocalDensity provides Density(1f, 1.6f)) {
                MaterialTheme { Box(Modifier.size(800.dp, 450.dp)) { TvMovieDetail(movie, null, "", "", { played = true }, { watched = it }, details = details) } }
            }
        }
        compose.onNodeWithText("Play").assertIsDisplayed()
        compose.onNodeWithText("Mark watched").assertIsDisplayed()
        val bounds = compose.onNodeWithText("Mark watched").getUnclippedBoundsInRoot()
        assertTrue("Control exceeds visible content: $bounds", bounds.bottom <= 354.dp)
        compose.onNodeWithText("Play").performSemanticsAction(SemanticsActions.RequestFocus).assertIsFocused().performKeyInput { pressKey(Key.Enter) }
        compose.onNodeWithText("Mark watched").performSemanticsAction(SemanticsActions.RequestFocus).assertIsFocused().performKeyInput { pressKey(Key.Enter) }
        compose.runOnIdle { assertTrue(played); assertTrue(watched) }
    }
}
