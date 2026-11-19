package net.kdhmedia.mochil.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import net.kdhmedia.mochil.video.*
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w400dp-h800dp")
class MobileShowDetailTest {
    @get:Rule val compose = createComposeRule()

    @Test fun seasonSelectionAndEpisodePlaybackRemainAvailable() {
        val show = VideoShow("Show", "Show", "show", 2, listOf(1, 2), "")
        val episode = VideoItem("ep", "tv", "Pilot", "Show", 1, 1, 3600.0, "", "stream", "", 0.0, emptyList())
        var selected = -1
        var played: VideoItem? = null
        compose.setContent { MaterialTheme {
            VideoScreen(VideoState(libraryId = "tv", libraryKind = "tv", selectedShow = show, selectedSeason = 1, items = listOf(episode)), { it }, null,
                {}, {}, { selected = it }, {}, {}, {}, { played = it }, {}, { _, _ -> }, {}, {}, {})
        } }
        compose.onNodeWithText("Season 2").performClick()
        assertEquals(2, selected)
        compose.onNodeWithText("Pilot").performClick()
        assertEquals(episode, played)
    }

    @Test fun partialSeasonDoesNotOfferAnUnsafeShowLevelResume() {
        val show = VideoShow("Example Show", "Example Show", "show", 1, listOf(1), "", year = 2022, overview = "A show story")
        val episode = VideoItem("ep", "tv", "Pilot", "Example Show", 1, 1, 3600.0, "", "stream", "", 120.0, emptyList(), overview = "An episode story")
        var played: VideoItem? = null
        compose.setContent { MaterialTheme {
            VideoScreen(VideoState(libraryId = "tv", libraryKind = "tv", selectedShow = show, selectedSeason = 1, items = listOf(episode)), { it }, null,
                {}, {}, {}, {}, {}, {}, { played = it }, {}, { _, _ -> }, {}, {}, {})
        } }
        compose.onNodeWithText("A show story").assertExists()
        compose.onNodeWithText("An episode story").assertExists()
        compose.onNodeWithText("Resume · S1E1").assertDoesNotExist()
        compose.onNodeWithText("Play").assertDoesNotExist()
    }

    @Test fun openingShowClearsHiddenSearch() {
        val show = VideoShow("Show", "Show", "show", 2, listOf(1, 2), "")
        var query = "not cleared"
        compose.setContent { MaterialTheme {
            VideoScreen(VideoState(libraryId = "tv", libraryKind = "tv", query = "Show", shows = listOf(show)), { it }, null,
                {}, {}, {}, {}, {}, { query = it }, {}, {}, { _, _ -> }, {}, {}, {})
        } }
        compose.onNodeWithText("Show").performClick()
        assertEquals("", query)
    }
}
