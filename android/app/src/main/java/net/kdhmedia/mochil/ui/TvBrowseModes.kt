package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.podcast.PodcastEpisodeSort
import net.kdhmedia.mochil.podcast.PodcastPlayedFilter

internal data class TvBrowseMode(val id: String, val label: String, val selected: Boolean)

internal fun tvSeasonModes(seasons: List<Int>, selectedSeason: Int?): List<TvBrowseMode> =
    seasons.distinct().sorted().map { season -> TvBrowseMode("season:$season", "Season $season", selectedSeason == season) }

internal fun tvPodcastEpisodeModes(sort: PodcastEpisodeSort, filter: PodcastPlayedFilter): List<TvBrowseMode> =
    PodcastEpisodeSort.entries.map { option -> TvBrowseMode("podcast-sort:${option.name}", "Sort · ${option.label}", sort == option) } +
        PodcastPlayedFilter.entries.map { option -> TvBrowseMode("podcast-filter:${option.name}", "Show · ${option.name}", filter == option) }

internal fun tvBrowseModes(
    libraryKind: String,
    publicationGrouping: PublicationGrouping,
    musicGrouping: MusicGrouping,
    atRoot: Boolean,
): List<TvBrowseMode> {
    if (!atRoot) return emptyList()
    return when (libraryKind) {
        "books", "lightnovels", "manga", "comics", "mixed", "calibre" -> listOf(
            TvBrowseMode(PublicationGrouping.Continue.name, "Continue", publicationGrouping == PublicationGrouping.Continue),
            TvBrowseMode(PublicationGrouping.Series.name, "Series", publicationGrouping == PublicationGrouping.Series),
            TvBrowseMode(PublicationGrouping.Authors.name, "Authors", publicationGrouping == PublicationGrouping.Authors),
            TvBrowseMode(PublicationGrouping.Books.name, "All", publicationGrouping == PublicationGrouping.Books),
            TvBrowseMode(PublicationGrouping.Recent.name, "Recent", publicationGrouping == PublicationGrouping.Recent),
        )
        "music" -> listOf(MusicGrouping.Discover to "Discover", MusicGrouping.RecentlyPlayed to "Recent", MusicGrouping.Albums to "Albums", MusicGrouping.Artists to "Artists", MusicGrouping.Favorites to "Favorites", MusicGrouping.Tracks to "Tracks")
            .map { (grouping, label) -> TvBrowseMode(grouping.name, label, musicGrouping == grouping) }
        "audiobooks" -> listOf(MusicGrouping.Continue to "Continue", MusicGrouping.Recent to "Recent", MusicGrouping.Series to "Series", MusicGrouping.Authors to "Authors", MusicGrouping.Tracks to "All")
            .map { (grouping, label) -> TvBrowseMode(grouping.name, label, musicGrouping == grouping) }
        else -> emptyList()
    }
}
