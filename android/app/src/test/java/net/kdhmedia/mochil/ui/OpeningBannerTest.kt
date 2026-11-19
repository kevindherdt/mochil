package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import kotlinx.coroutines.flow.MutableStateFlow
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.DownloadQueueItem
import net.kdhmedia.mochil.catalogue.DownloadQueueState
import net.kdhmedia.mochil.catalogue.LibraryState
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * The banner draws the queue's own flow, so a progress tick is the banner's business and nobody
 * else's. These tests pin that the text and the bar follow the flow, that the banner still lays the
 * shelf out when it appears or goes away, and that an opening book still wins the line.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], qualifiers = "w400dp-h800dp")
class OpeningBannerTest {
    @get:Rule val compose = createComposeRule()

    private fun book(id: String = "a") = BookSummary(
        id = id, libraryId = "books", title = "Shelf book", authors = listOf("An Author"), series = "",
        seriesIndex = null, format = "epub", mediaType = "application/epub+zip", language = "en",
        description = "", size = 10,
    )

    private fun downloading(done: Long, total: Long) =
        DownloadQueueItem(book(), state = DownloadQueueState.Downloading, done = done, total = total)

    @Test fun theBannerFollowsTheQueueFlow() {
        val queue = MutableStateFlow(listOf(downloading(1_048_576, 4_194_304)))
        compose.setContent { MaterialTheme { OpeningBanner(LibraryState(), queue) } }

        compose.onNodeWithText("Downloading Shelf book (1 of 1): 1.0 MB of 4.0 MB", substring = true).assertIsDisplayed()
        compose.onNode(progress(0.25f)).assertIsDisplayed()

        compose.runOnIdle { queue.value = listOf(downloading(2_097_152, 4_194_304)) }

        compose.onNodeWithText("2.0 MB", substring = true).assertIsDisplayed()
        compose.onNode(progress(0.5f)).assertIsDisplayed()
    }

    /**
     * The banner's height is layout: while it shows, the shelf starts below it. Collecting the flow
     * inside the banner must not turn that into a stale `remember`.
     */
    @Test fun theShelfMovesWhenTheBannerAppearsAndGoes() {
        val queue = MutableStateFlow<List<DownloadQueueItem>>(emptyList())
        compose.setContent { MaterialTheme { Shelf(queue) } }
        val withoutBanner = compose.onNodeWithTag("shelf").getUnclippedBoundsInRoot().top

        compose.runOnIdle { queue.value = listOf(downloading(1_048_576, 4_194_304)) }
        compose.onNodeWithText("Downloading", substring = true).assertIsDisplayed()
        val withBanner = compose.onNodeWithTag("shelf").getUnclippedBoundsInRoot().top
        assertTrue("the banner must push the shelf down, got $withBanner vs $withoutBanner", withBanner > withoutBanner)

        compose.runOnIdle { queue.value = emptyList() }
        compose.onAllNodesWithText("Downloading", substring = true).assertCountEquals(0)
        assertTrue(
            "an idle queue must give the space back",
            compose.onNodeWithTag("shelf").getUnclippedBoundsInRoot().top <= withoutBanner,
        )
    }

    @Test fun openingABookStillWinsTheLine() {
        val queue = MutableStateFlow(listOf(downloading(1_048_576, 4_194_304)))
        val opening = LibraryState(openingBookId = "b", openingTitle = "Shelf book", openingDone = 2_097_152, openingTotal = 4_194_304)
        compose.setContent { MaterialTheme { OpeningBanner(opening, queue) } }

        compose.onNodeWithText("Opening Shelf book: 2.0 MB of 4.0 MB").assertIsDisplayed()
        compose.onAllNodesWithText("Downloading", substring = true).assertCountEquals(0)
    }

    @Composable
    private fun Shelf(queue: MutableStateFlow<List<DownloadQueueItem>>) {
        Column {
            OpeningBanner(LibraryState(), queue)
            Text("Shelf", Modifier.testTag("shelf"))
        }
    }

    private fun progress(fraction: Float) =
        SemanticsMatcher.expectValue(SemanticsProperties.ProgressBarRangeInfo, ProgressBarRangeInfo(fraction, 0f..1f))
}
