package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.video.*
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w1280dp-h720dp")
class TvCastBehaviorTest {
    @get:Rule val compose = createComposeRule()
    private val details = VideoTitleDetails(cast = listOf(VideoCastMember("Actor One", "Hero"), VideoCastMember("Actor Two")))
    private val movie = VideoItem("id", "movies", "Movie", "", 0, 0, 0.0, "", "", "", 0.0, emptyList())

    @Test fun movieCastShowsSeparateNamesAndRolesAndSupportsDpad() {
        compose.setContent { MaterialTheme { Box(Modifier.size(1280.dp, 720.dp)) {
            TvMovieDetail(movie, null, "", "", {}, {}, details = details)
        } } }
        compose.onNodeWithText("Actor One").assertIsDisplayed().performSemanticsAction(SemanticsActions.RequestFocus).assertIsFocused()
        compose.onNodeWithText("Hero").assertIsDisplayed()
        compose.onNodeWithText("Actor One").performKeyInput { pressKey(Key.DirectionRight) }
        compose.onNodeWithText("Actor Two").assertIsFocused()
    }
    @Test fun castFocusRingMatchesPortraitBounds() {
        compose.setContent { MaterialTheme { VideoCastRow(details, { "" }, null) } }
        compose.onNodeWithText("Actor One").performSemanticsAction(SemanticsActions.RequestFocus).assertIsFocused()
        val ring = compose.onNodeWithTag("cast-focus-Actor One", useUnmergedTree = true).getUnclippedBoundsInRoot()
        val portrait = compose.onNodeWithTag("cast-portrait-Actor One", useUnmergedTree = true).getUnclippedBoundsInRoot()
        org.junit.Assert.assertEquals(portrait, ring)
        org.junit.Assert.assertTrue(ring.right - ring.left in 68.dp..71.dp)
        org.junit.Assert.assertTrue(ring.bottom - ring.top in 68.dp..71.dp)
    }
    @Test fun showDisplaysSeparateCastNamesAndRoles() {
        val show = VideoShow("Show", "Show", "id", 0, emptyList(), "")
        compose.setContent { MaterialTheme { Box(Modifier.size(1280.dp, 720.dp)) {
            TvShowDetail(show, VideoState(details = mapOf("show:Show" to details)), null, { "" }, {}, {}, { _, _ -> }, {}, {})
        } } }
        compose.onNodeWithText("Actor One").assertIsDisplayed()
        compose.onNodeWithText("Hero").assertIsDisplayed()
    }
}