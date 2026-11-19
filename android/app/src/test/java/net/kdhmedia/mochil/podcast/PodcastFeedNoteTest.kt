package net.kdhmedia.mochil.podcast

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class PodcastFeedNoteTest {
    private val show = PodcastShow("id", "Show", "", 12)

    @Test fun aHealthyFeedSaysNothing() {
        assertNull(show.feedNote())
        assertNull(show.copy(lastError = "  ").feedNote())
    }

    @Test fun aFailedRefreshReassuresThatSavedEpisodesRemain() {
        assertEquals("Feed refresh failed · saved episodes are still available", show.copy(lastError = "HTTP 503").feedNote())
    }
}
