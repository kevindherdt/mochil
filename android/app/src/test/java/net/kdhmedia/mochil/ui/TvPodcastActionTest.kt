package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class TvPodcastActionTest {
    @Test
    fun actionReflectsCurrentEpisodeStateAndSuppressesDuplicateMutation() {
        assertEquals("Mark as played", tvPodcastPlayedActionLabel(finished = false, updating = false))
        assertEquals("Mark as unplayed", tvPodcastPlayedActionLabel(finished = true, updating = false))
        assertEquals("Updating…", tvPodcastPlayedActionLabel(finished = false, updating = true))
        assertEquals("Updating…", tvPodcastPlayedActionLabel(finished = true, updating = true))
    }
}
