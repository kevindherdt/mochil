package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.Density
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.video.*
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.junit.Assert.assertTrue

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w1280dp-h720dp")
class TvShowEpisodeBehaviorTest {
    @get:Rule val compose = createComposeRule()
    private fun episode(id: String, description: String) = VideoItem(id, "tv", id, "Show", 1, 1, 0.0, "", "", "", 0.0, emptyList(), overview = description)

    @Test fun standardNormalFontDescriptionRemainsVisible() = checkStandardDescription(1f)
    @Test fun standardLargeFontDescriptionRemainsVisible() = checkStandardDescription(1.6f)

    private fun checkStandardDescription(font: Float) {
        val show = VideoShow("Show", "Show", "id", 3, listOf(1), "", overview = "Show story")
        val long = "A long episode story ".repeat(30)
        val state = VideoState(items = listOf(episode("Episode One", long), episode("Episode Two", "Brief"), episode("Episode Three", "")), details = mapOf("show:Show" to VideoTitleDetails(cast = listOf(VideoCastMember("Actor One", "Hero")))))
        compose.setContent { CompositionLocalProvider(LocalDensity provides Density(1f, font)) { MaterialTheme { Box(Modifier.size(1280.dp, 720.dp)) {
            TvShowDetail(show, state, null, { "" }, {}, {}, { _, _ -> }, {}, {})
        } } } }
        compose.onNodeWithText("Play").performSemanticsAction(SemanticsActions.RequestFocus).performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Episode One").assertIsFocused()
        fun visible(text: String) { val b = compose.onNodeWithText(text).getUnclippedBoundsInRoot(); assertTrue("description clipped $font: $b", b.top >= 64.dp && b.bottom <= 624.dp) }
        visible(long)
        val initial = compose.onNodeWithText("Episode One").getUnclippedBoundsInRoot().top
        compose.onNodeWithText("Episode One").performKeyInput { pressKey(Key.DirectionRight) }
        compose.onNodeWithText("Episode Two").assertIsFocused(); visible("Brief")
        val next = compose.onNodeWithText("Episode Two").getUnclippedBoundsInRoot().top
        assertTrue("rail moved on description change: $initial $next", kotlin.math.abs((initial - next).value) < 1f)
        compose.onNodeWithText("Episode Two").performKeyInput { pressKey(Key.DirectionRight) }
        compose.onNodeWithText("Episode Three").assertIsFocused()
        compose.onNodeWithText("Episode Three").performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Actor One").assertIsFocused().performKeyInput { pressKey(Key.DirectionUp) }
        // Up from the cast re-enters the rail on the card the viewer left, and that episode's synopsis comes back in the
        // same measured slot, so the rail has not jumped.
        val returned = compose.onNode(isFocused()).fetchSemanticsNode().config[SemanticsProperties.Text].first().text
        assertTrue("Up from cast must return into the episode rail: $returned", returned.startsWith("Episode "))
        val returnedTop = compose.onNodeWithText(returned).getUnclippedBoundsInRoot().top
        assertTrue("rail moved on cast return: $initial $returnedTop", kotlin.math.abs((initial - returnedTop).value) < 1f)
        // An episode without its own synopsis falls back to the show's, which is what an empty description shows.
        visible(if (returned == "Episode Three") "Show story" else "Brief")
    }

    @Test fun episodesAreVisibleBeforeCastOnInitialViewport() {
        val show = VideoShow("Show", "Show", "id", 2, listOf(1), "", overview = "Show story")
        val state = VideoState(items = listOf(episode("Episode One", "First story"), episode("Episode Two", "Second story")), details = mapOf("show:Show" to VideoTitleDetails(cast = listOf(VideoCastMember("Actor One", "Hero")))))
        compose.setContent { MaterialTheme { Box(Modifier.size(1280.dp, 720.dp)) {
            TvShowDetail(show, state, null, { "" }, {}, {}, { _, _ -> }, {}, {})
        } } }
        compose.onNodeWithText("Episode One").assertIsDisplayed()
        val episodeBounds = compose.onNodeWithText("Episode One").getUnclippedBoundsInRoot()
        val castBounds = compose.onNodeWithText("Actor One").getUnclippedBoundsInRoot()
        assertTrue("Cast must follow episodes: episode=$episodeBounds cast=$castBounds", castBounds.top > episodeBounds.bottom)
    }

    @Test fun showControlsUseCompactVisualsAndKeepRemoteTargets() {
        val show = VideoShow("Show", "Show", "id", 2, listOf(1, 2), "")
        val state = VideoState(items = listOf(episode("Episode One", "First story")), selectedSeason = 1)
        var played = false
        var selected = 0
        compose.setContent { MaterialTheme { Box(Modifier.size(1280.dp, 720.dp)) {
            TvShowDetail(show, state, null, { "" }, { selected = it }, { played = true }, { _, _ -> }, {}, {})
        } } }
        for (label in listOf("Play", "Season 1")) {
            val target = compose.onNodeWithText(label).getUnclippedBoundsInRoot()
            val visual = compose.onNodeWithTag("compact-$label", useUnmergedTree = true).getUnclippedBoundsInRoot()
            assertTrue("$label target too small: $target", target.bottom - target.top >= 48.dp)
            assertTrue("$label target too narrow: $target", target.right - target.left >= 48.dp)
            assertTrue("$label visual too large: $visual", visual.bottom - visual.top <= 36.dp)
        }
        compose.onNodeWithText("Play").performSemanticsAction(SemanticsActions.RequestFocus).assertIsFocused().performKeyInput { pressKey(Key.Enter); pressKey(Key.DirectionRight) }
        compose.onNodeWithText("Season 1").assertIsFocused().performKeyInput { pressKey(Key.DirectionRight) }
        compose.onNodeWithText("Season 2").assertIsFocused().performKeyInput { pressKey(Key.Enter) }
        compose.runOnIdle { assertTrue(played); assertTrue(selected == 2) }
    }

    @Test fun remoteTraversesEpisodesThenCastAndBack() {
        val show = VideoShow("Show", "Show", "id", 2, listOf(1), "", overview = "Show story")
        val state = VideoState(items = listOf(episode("Episode One", "First story"), episode("Episode Two", "Second story")), details = mapOf("show:Show" to VideoTitleDetails(cast = listOf(VideoCastMember("Actor One", "Hero")))))
        compose.setContent { MaterialTheme { Box(Modifier.size(1280.dp, 720.dp)) {
            TvShowDetail(show, state, null, { "" }, {}, {}, { _, _ -> }, {}, {})
        } } }
        compose.onNodeWithText("Play").performSemanticsAction(SemanticsActions.RequestFocus).performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Episode One").assertIsFocused().performKeyInput { pressKey(Key.DirectionRight) }
        compose.onNodeWithText("Episode Two").assertIsFocused()
        compose.onNodeWithText("Second story").assertIsDisplayed()
        compose.onNodeWithText("Episode Two").performKeyInput { pressKey(Key.DirectionDown) }
        compose.onNodeWithText("Actor One").assertIsFocused().assertIsDisplayed()
        compose.onNodeWithText("Actor One").performKeyInput { pressKey(Key.DirectionUp) }
        // Up from the cast re-enters the rail on the episode the viewer left, rather than jumping to whatever card
        // happens to sit above the portrait.
        val returned = compose.onNode(isFocused()).fetchSemanticsNode().config[SemanticsProperties.Text].first().text
        assertTrue("Up from cast must return into the episode rail: $returned", returned.startsWith("Episode "))
        compose.onNodeWithText("Second story").assertIsDisplayed()
        compose.onNodeWithText(returned).performKeyInput { pressKey(Key.DirectionLeft) }
        compose.onNodeWithText("Episode One").assertIsFocused()
        compose.onNodeWithText("First story").assertIsDisplayed()
    }

    @Test fun changingOverviewLengthKeepsEpisodeRailInPlace() {
        val show = VideoShow("Show", "Show", "id", 2, listOf(1), "", overview = "Short story")
        val state = VideoState(items = listOf(episode("Episode One", "First line\nSecond line\nThird line"), episode("Episode Two", "Brief")))
        compose.setContent { MaterialTheme { Box(Modifier.size(1280.dp, 720.dp)) {
            TvShowDetail(show, state, null, { "" }, {}, {}, { _, _ -> }, {}, {})
        } } }
        val initial = compose.onNodeWithTag("compact-Play", useUnmergedTree = true).getUnclippedBoundsInRoot().top
        compose.onNodeWithText("Episode One").performSemanticsAction(SemanticsActions.RequestFocus)
        val long = compose.onNodeWithTag("compact-Play", useUnmergedTree = true).getUnclippedBoundsInRoot().top
        val description = compose.onNodeWithText("First line\nSecond line\nThird line").getUnclippedBoundsInRoot()
        assertTrue("Episode description clipped: $description", description.top >= 64.dp && description.bottom <= 624.dp)
        compose.onNodeWithText("Episode One").performKeyInput { pressKey(Key.DirectionRight) }
        val shortDescription = compose.onNodeWithText("Brief").getUnclippedBoundsInRoot()
        assertTrue("Description height changed: $description / $shortDescription", description.bottom - description.top == shortDescription.bottom - shortDescription.top)
        val short = compose.onNodeWithTag("compact-Play", useUnmergedTree = true).getUnclippedBoundsInRoot().top
        assertTrue("Episode rail moved as description changed: $initial / $long / $short", initial == long && long == short)
        compose.onNodeWithText("Brief").assertIsDisplayed()
    }

    @Test fun dpadEpisodeFocusShowsDescriptionAndRestoresShowOnExit() {
        val show = VideoShow("Show", "Show", "id", 2, listOf(1), "", overview = "Show story")
        val state = VideoState(items = listOf(episode("Episode One", "First story"), episode("Episode Two", "Second story")))
        compose.setContent { MaterialTheme { Box(Modifier.size(1280.dp, 720.dp)) {
            TvShowDetail(show, state, null, { "" }, {}, {}, { _, _ -> }, {}, {})
        } } }
        compose.onNodeWithText("Show story").assertIsDisplayed()
        compose.onNodeWithText("Episode One").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.onNodeWithText("First story").assertIsDisplayed()
        compose.onNodeWithText("Episode One").performKeyInput { pressKey(Key.DirectionRight) }
        compose.onNodeWithText("Episode Two").assertIsFocused()
        compose.onNodeWithText("Second story").assertIsDisplayed()
        compose.onNodeWithText("Play").performSemanticsAction(SemanticsActions.RequestFocus)
        compose.onNodeWithText("Show story").assertIsDisplayed()
    }
}
