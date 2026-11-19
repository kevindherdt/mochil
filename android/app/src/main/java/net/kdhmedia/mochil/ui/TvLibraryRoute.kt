package net.kdhmedia.mochil.ui

import androidx.activity.compose.BackHandler
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import net.kdhmedia.mochil.audio.MusicQueueViewModel
import net.kdhmedia.mochil.audio.canPersistMusicPlaybackQueue
import net.kdhmedia.mochil.audio.MusicState
import net.kdhmedia.mochil.audio.MusicViewModel
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.podcast.PodcastState
import net.kdhmedia.mochil.podcast.PodcastViewModel
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoState
import net.kdhmedia.mochil.video.VideoViewModel

/** Adapts each TV media kind to one shared, focus-aware library surface. */
@Composable
internal fun TvLibraryRoute(
    libraryViewModel: LibraryViewModel,
    musicViewModel: MusicViewModel,
    musicQueueViewModel: MusicQueueViewModel,
    playbackViewModel: PlaybackViewModel,
    podcastViewModel: PodcastViewModel,
    videoViewModel: VideoViewModel,
    gridPositions: TvGridPositionStore,
    onSelectLibrary: (LibrarySummary) -> Unit,
    onPlayVideo: (VideoItem) -> Unit,
    onHome: () -> Unit,
) {
    val library by libraryViewModel.state.collectAsStateWithLifecycle()
    val kind = library.selected?.kind.orEmpty()
    DisposableEffect(kind) {
        onDispose { podcastViewModel.cancelPlaybackResolution() }
    }
    val artwork = TvLibraryArtwork(
        book = { libraryViewModel.coverUrl(it).orEmpty() },
        series = { libraryViewModel.seriesCoverUrl(it).orEmpty() },
        author = { libraryViewModel.authorCoverUrl(it).orEmpty() },
        audioGroup = musicViewModel::artworkUrl,
        track = musicViewModel::artworkUrl,
        video = videoViewModel::artworkUrl,
        podcast = podcastViewModel::artworkUrl,
        episode = podcastViewModel::artworkUrl,
    )
    val publicationActions = TvPublicationActions(
        openSeries = libraryViewModel::openSeries,
        openAuthor = libraryViewModel::openAuthor,
        read = libraryViewModel::readBook,
        loadMore = libraryViewModel::loadMoreVisible,
        setGrouping = libraryViewModel::setGrouping,
        selectBucket = libraryViewModel::selectBucket,
    )
    val baseAudioActions = TvAudioActions(
        openGroup = musicViewModel::openGroup,
        loadMore = musicViewModel::loadMore,
        setGrouping = musicViewModel::setGrouping,
        setLayout = musicViewModel::setLayout,
        selectBucket = musicViewModel::jumpToBucket,
    )
    val videoActions = TvVideoActions(
        openShow = videoViewModel::openShow,
        play = onPlayVideo,
        setWatched = videoViewModel::setWatched,
        selectSeason = videoViewModel::selectSeason,
        loadMore = videoViewModel::loadMore,
        loadDetails = videoViewModel::loadDetails,
    )
    val basePodcastActions = TvPodcastActions(
        openShow = podcastViewModel::openShow,
        setPlayed = podcastViewModel::setPlayed,
        setSort = podcastViewModel::setEpisodeSort,
        setFilter = podcastViewModel::setPlayedFilter,
        loadMore = podcastViewModel::loadMore,
    )
    when (kind) {
        "movies", "tv", "anime" -> {
            val video by videoViewModel.state.collectAsStateWithLifecycle()
            val back = { if (video.selectedShow != null) videoViewModel.closeShow() else onHome() }
            BackHandler(onBack = back)
            TvLibraryScreen(
                library, MusicState(), video, PodcastState(), libraryViewModel.authorizationHeader(), artwork,
                TvLibraryActions(
                    TvNavigationActions(back, onSelectLibrary, videoViewModel::search, videoViewModel::retry),
                    publicationActions, baseAudioActions, videoActions, basePodcastActions,
                ),
                video.query,
                gridPositions,
            )
        }
        "music", "audiobooks" -> {
            val music by musicViewModel.state.collectAsStateWithLifecycle()
            val back = {
                when {
                    music.selectedGroup != null -> musicViewModel.closeGroup()
                    music.selectedArtist != null -> musicViewModel.closeArtist()
                    else -> onHome()
                }
            }
            BackHandler(onBack = back)
            val audioActions = baseAudioActions.copy(
                play = { track ->
                    val queue = tvPlaybackTracks(music)
                    if (canPersistMusicPlaybackQueue(queue)) musicQueueViewModel.replace(queue)
                    playbackViewModel.play(queue, track, musicViewModel::playbackUrl)
                },
                playNext = { track ->
                    if (track.libraryKind == "music") musicQueueViewModel.playNext(track, playbackViewModel.state.value.current)
                    playbackViewModel.playNext(track, musicViewModel::playbackUrl)
                },
                addToQueue = { track ->
                    if (track.libraryKind == "music") musicQueueViewModel.append(track)
                    playbackViewModel.addToQueue(track, musicViewModel::playbackUrl)
                },
                toggleFavorite = musicViewModel::toggleFavorite,
            )
            TvLibraryScreen(
                library, music, VideoState(), PodcastState(), libraryViewModel.authorizationHeader(), artwork,
                TvLibraryActions(
                    TvNavigationActions(back, onSelectLibrary, musicViewModel::search, musicViewModel::retry),
                    publicationActions, audioActions, videoActions, basePodcastActions,
                ),
                music.query,
                gridPositions,
            )
        }
        "podcasts" -> {
            val podcasts by podcastViewModel.state.collectAsStateWithLifecycle()
            val back = { if (podcasts.selectedShow != null) podcastViewModel.closeShow() else onHome() }
            BackHandler(onBack = back)
            val podcastActions = basePodcastActions.copy(
                play = { episode ->
                    podcastViewModel.resolveForPlayback(episode) { resolved ->
                        val queue = podcasts.episodes.map {
                            if (it.libraryId == resolved.libraryId && it.id == resolved.id) resolved else it
                        }
                        playbackViewModel.play(
                            queue,
                            resolved,
                            podcastViewModel::playbackUrl,
                            (resolved.resumePositionSeconds * 1_000).toLong(),
                        )
                    }
                },
            )
            TvLibraryScreen(
                library, MusicState(), VideoState(), podcasts, libraryViewModel.authorizationHeader(), artwork,
                TvLibraryActions(
                    TvNavigationActions(back, onSelectLibrary, podcastViewModel::search, podcastViewModel::retry),
                    publicationActions, baseAudioActions, videoActions, podcastActions,
                ),
                podcasts.query,
                gridPositions,
            )
        }
        else -> {
            val back = {
                when (tvPublicationBackTarget(
                    hasCollection = library.selectedCollection != null,
                    hasSeries = library.selectedSeries != null || library.expandedSeries != null,
                    hasAuthor = library.selectedAuthor != null,
                )) {
                    TvPublicationBackTarget.Collection -> libraryViewModel.closeCollection()
                    TvPublicationBackTarget.Series -> libraryViewModel.closeSeries()
                    TvPublicationBackTarget.Author -> libraryViewModel.closeAuthor()
                    TvPublicationBackTarget.Home -> onHome()
                }
            }
            BackHandler(onBack = back)
            TvLibraryScreen(
                library, MusicState(), VideoState(), PodcastState(), libraryViewModel.authorizationHeader(), artwork,
                TvLibraryActions(
                    TvNavigationActions(back, onSelectLibrary, libraryViewModel::search, libraryViewModel::retry),
                    publicationActions, baseAudioActions, videoActions, basePodcastActions,
                ),
                library.query,
                gridPositions,
            )
        }
    }
}
