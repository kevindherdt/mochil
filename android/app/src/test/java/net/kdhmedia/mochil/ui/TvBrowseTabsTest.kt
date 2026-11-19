package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.podcast.PodcastEpisodeSort
import net.kdhmedia.mochil.podcast.PodcastPlayedFilter
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TvBrowseTabsTest {
    @Test
    fun fewPlainViewsAreTabsButLongSortAndFilterListsStayInTheMenu() {
        assertTrue(tvModesFitAsTabs(tvBrowseModes("books", PublicationGrouping.Series, MusicGrouping.Albums, atRoot = true)))
        assertTrue(tvModesFitAsTabs(tvBrowseModes("music", PublicationGrouping.Series, MusicGrouping.Albums, atRoot = true)))
        assertFalse(tvModesFitAsTabs(tvPodcastEpisodeModes(PodcastEpisodeSort.entries.first(), PodcastPlayedFilter.entries.first())))
        assertFalse(tvModesFitAsTabs(tvSeasonModes(listOf(1, 2), 1)))
    }
}
