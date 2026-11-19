package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.AudioGroup
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicLayout
import net.kdhmedia.mochil.catalogue.AuthorSummary
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.catalogue.SeriesSummary
import net.kdhmedia.mochil.podcast.PodcastEpisodeSort
import net.kdhmedia.mochil.podcast.PodcastPlayedFilter
import net.kdhmedia.mochil.podcast.PodcastShow
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoShow

internal data class TvLibraryArtwork(
    val book: (BookSummary) -> String,
    val series: (SeriesSummary) -> String,
    val author: (AuthorSummary) -> String,
    val audioGroup: (AudioGroup) -> String,
    val track: (AudioTrack) -> String,
    val video: (String) -> String,
    val podcast: (PodcastShow) -> String,
    val episode: (AudioTrack) -> String,
)

internal data class TvNavigationActions(
    val back: () -> Unit,
    val selectLibrary: (LibrarySummary) -> Unit,
    val search: (String) -> Unit,
    val retry: () -> Unit,
)

internal data class TvPublicationActions(
    val openSeries: (SeriesSummary) -> Unit = {},
    val openAuthor: (AuthorSummary) -> Unit = {},
    val read: (BookSummary) -> Unit = {},
    val loadMore: () -> Unit = {},
    val setGrouping: (PublicationGrouping) -> Unit = {},
    val selectBucket: (String) -> Unit = {},
)

internal data class TvAudioActions(
    val openGroup: (AudioGroup) -> Unit = {},
    val play: (AudioTrack) -> Unit = {},
    val playNext: (AudioTrack) -> Unit = {},
    val addToQueue: (AudioTrack) -> Unit = {},
    val toggleFavorite: (AudioTrack) -> Unit = {},
    val loadMore: () -> Unit = {},
    val setGrouping: (MusicGrouping) -> Unit = {},
    val setLayout: (MusicLayout) -> Unit = {},
    val selectBucket: (String) -> Unit = {},
)

internal data class TvVideoActions(
    val openShow: (VideoShow) -> Unit = {},
    val play: (VideoItem) -> Unit = {},
    val setWatched: (VideoItem, Boolean) -> Unit = { _, _ -> },
    val selectSeason: (Int) -> Unit = {},
    val loadMore: () -> Unit = {},
    val loadDetails: (VideoShow?, VideoItem?) -> Unit = { _, _ -> },
)

internal data class TvPodcastActions(
    val openShow: (PodcastShow) -> Unit = {},
    val play: (AudioTrack) -> Unit = {},
    val setPlayed: (AudioTrack, Boolean) -> Unit = { _, _ -> },
    val setSort: (PodcastEpisodeSort) -> Unit = {},
    val setFilter: (PodcastPlayedFilter) -> Unit = {},
    val loadMore: () -> Unit = {},
)

internal data class TvLibraryActions(
    val navigation: TvNavigationActions,
    val publications: TvPublicationActions = TvPublicationActions(),
    val audio: TvAudioActions = TvAudioActions(),
    val video: TvVideoActions = TvVideoActions(),
    val podcasts: TvPodcastActions = TvPodcastActions(),
)
