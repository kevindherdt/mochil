package net.kdhmedia.mochil.video

import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

class VideoWatchedUpdatesTest {
    private val episode = VideoItem("one", "anime", "One", "Original", 1, 1, 100.0,
        "", "/stream", "", 20.0, emptyList())
    private fun show(name: String) = VideoShow(name, name, "one", 10, listOf(1), "")

    @Test fun switchedShowDoesNotReceivePreviousShowsCount() {
        val other = show("Other")
        val state = VideoState(libraryId = "anime", selectedShow = other,
            shows = listOf(show("Original"), other), continueWatching = listOf(episode))
        val result = state.withConfirmedWatched(episode, true)
        assertSame(other, result.selectedShow)
        assertEquals(1, result.shows.first().watchedCount)
        assertEquals(0, result.shows.last().watchedCount)
        assertTrue(result.continueWatching.isEmpty())
    }

    @Test fun switchedLibraryIsUntouchedEvenWhenNamesMatch() {
        val state = VideoState(libraryId = "tv", selectedShow = show("Original"), error = "Current error")
        assertSame(state, state.withConfirmedWatched(episode, true))
    }

    @Test fun repeatedConfirmationUsesCurrentItemInsteadOfOldActionSnapshot() {
        val state = VideoState(libraryId = "anime", items = listOf(episode),
            selectedShow = show("Original"), shows = listOf(show("Original")))
        val once = state.withConfirmedWatched(episode, true)
        val twice = once.withConfirmedWatched(episode, true)
        assertEquals(1, twice.selectedShow?.watchedCount)
        assertEquals(1, twice.shows.single().watchedCount)
        assertTrue(twice.items.single().finished)
        val cleared = twice.withConfirmedWatched(episode, false)
        assertEquals(0, cleared.selectedShow?.watchedCount)
        assertEquals(0.0, cleared.items.single().resumeSeconds, 0.0)
    }
}
