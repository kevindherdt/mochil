package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.podcast.PodcastEpisodeSort
import net.kdhmedia.mochil.podcast.PodcastPlayedFilter
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class TvBrowseModesTest {
    @Test fun `publication roots offer continue series authors all and recent`() {
        val modes = tvBrowseModes("books", PublicationGrouping.Series, MusicGrouping.Albums, atRoot = true)
        assertEquals(listOf("Continue", "Series", "Authors", "All", "Recent"), modes.map(TvBrowseMode::label))
        assertEquals("Series", modes.single(TvBrowseMode::selected).label)
    }

    @Test fun `audiobooks use media-appropriate group names`() {
        assertEquals(
            listOf("Continue", "Recent", "Series", "Authors", "All"),
            tvBrowseModes("audiobooks", PublicationGrouping.Series, MusicGrouping.Authors, atRoot = true).map(TvBrowseMode::label),
        )
    }

    @Test fun `drill-down hides root browse controls`() {
        assertTrue(tvBrowseModes("music", PublicationGrouping.Series, MusicGrouping.Albums, atRoot = false).isEmpty())
    }

    @Test fun `music discovery is the first remote destination`() {
        val modes = tvBrowseModes("music", PublicationGrouping.Series, MusicGrouping.Discover, atRoot = true)
        assertEquals(listOf("Discover", "Recent", "Albums", "Artists", "Favorites", "Tracks"), modes.map(TvBrowseMode::label))
        assertEquals("Discover", modes.single(TvBrowseMode::selected).label)
    }

    @Test fun `show drill-down exposes sorted distinct seasons and active season`() {
        val modes = tvSeasonModes(listOf(3, 1, 2, 2), selectedSeason = 2)
        assertEquals(listOf("Season 1", "Season 2", "Season 3"), modes.map(TvBrowseMode::label))
        assertEquals("season:2", modes.single(TvBrowseMode::selected).id)
    }

    @Test fun `podcast drill-down exposes ordering and played filters`() {
        val modes = tvPodcastEpisodeModes(PodcastEpisodeSort.Oldest, PodcastPlayedFilter.Unplayed)
        assertEquals(
            listOf("Sort · Newest first", "Sort · Oldest first", "Sort · Title A–Z", "Show · All", "Show · Unplayed", "Show · Played", "Show · Downloaded"),
            modes.map(TvBrowseMode::label),
        )
        assertTrue(modes.single { it.id == "podcast-sort:Oldest" }.selected)
        assertTrue(modes.single { it.id == "podcast-filter:Unplayed" }.selected)
    }
}
