package net.kdhmedia.mochil.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import net.kdhmedia.mochil.audio.MusicPlaylistState
import net.kdhmedia.mochil.audio.MusicState
import net.kdhmedia.mochil.audio.MusicPlaylistViewModel
import net.kdhmedia.mochil.audio.MusicQueueViewModel
import net.kdhmedia.mochil.audio.MusicViewModel
import net.kdhmedia.mochil.audio.visibleTracks
import net.kdhmedia.mochil.catalogue.CollectionViewModel
import net.kdhmedia.mochil.catalogue.CollectionPickerState
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import net.kdhmedia.mochil.catalogue.downloadQueueItems
import net.kdhmedia.mochil.catalogue.queueBook
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.podcast.PodcastViewModel
import net.kdhmedia.mochil.podcast.PodcastState
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoState
import net.kdhmedia.mochil.video.VideoViewModel
import net.kdhmedia.mochil.wiki.WikiViewModel
import net.kdhmedia.mochil.wiki.WikiState

/** Owns media-catalogue state and actions while the Library destination is composed. */
@Composable
internal fun MobileLibraryRoute(
    libraryViewModel: LibraryViewModel,
    musicViewModel: MusicViewModel,
    podcastViewModel: PodcastViewModel,
    videoViewModel: VideoViewModel,
    wikiViewModel: WikiViewModel,
    collectionViewModel: CollectionViewModel,
    playlistViewModel: MusicPlaylistViewModel,
    queueViewModel: MusicQueueViewModel,
    playbackViewModel: PlaybackViewModel,
    playlistState: MusicPlaylistState,
    showCollections: Boolean,
    onOpenCollections: () -> Unit,
    onWatch: (VideoItem) -> Unit,
) {
    val libraryState by libraryViewModel.state.collectAsStateWithLifecycle()
    val selectedKind = libraryState.selected?.kind?.lowercase().orEmpty()
    val musicState = if (selectedKind in setOf("music", "audiobooks"))
        musicViewModel.state.collectAsStateWithLifecycle().value else MusicState()
    val podcastState = if (selectedKind == "podcasts")
        podcastViewModel.state.collectAsStateWithLifecycle().value else PodcastState()
    val videoState = if (selectedKind in setOf("movies", "tv", "anime", "video", "videos"))
        videoViewModel.state.collectAsStateWithLifecycle().value else VideoState()
    val wikiState = if (selectedKind == "vault")
        wikiViewModel.state.collectAsStateWithLifecycle().value else WikiState()
    val collectionState = if (selectedKind in setOf("books", "lightnovels", "manga", "comics", "mixed", "calibre"))
        collectionViewModel.state.collectAsStateWithLifecycle().value else CollectionPickerState()

    DisposableEffect(libraryState.selected?.kind) {
        onDispose { podcastViewModel.cancelPlaybackResolution() }
    }

    LaunchedEffect(collectionViewModel, libraryViewModel) {
        collectionViewModel.changes.collect(libraryViewModel::refreshCollections)
    }

    LaunchedEffect(selectedKind) {
        if (selectedKind == "music") queueViewModel.ensureLoaded()
    }

    LibraryScreen(
        state = libraryState,
        musicState = musicState,
        podcasts = PodcastBindings(
            state = podcastState,
            showArtworkUrl = podcastViewModel::artworkUrl,
            episodeArtworkUrl = podcastViewModel::artworkUrl,
            open = podcastViewModel::open,
            openShow = podcastViewModel::openShow,
            closeShow = podcastViewModel::closeShow,
            setLayout = podcastViewModel::setLayout,
            search = podcastViewModel::search,
            setPlayedFilter = podcastViewModel::setPlayedFilter,
            setEpisodeSort = podcastViewModel::setEpisodeSort,
            setPlayed = podcastViewModel::setPlayed,
            play = { episode ->
                podcastViewModel.resolveForPlayback(episode) { resolved ->
                    val queue = podcastState.episodes.map {
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
            toggleDownload = podcastViewModel::toggleDownload,
            loadMore = podcastViewModel::loadMore,
            retry = podcastViewModel::retry,
            subscribe = podcastViewModel::subscribe,
            unsubscribe = podcastViewModel::unsubscribe,
            discover = podcastViewModel::discover,
            clearDiscovery = podcastViewModel::clearDiscovery,
            discoveryArtworkUrl = podcastViewModel::artworkUrl,
            refreshShow = podcastViewModel::refreshShow,
            updatePolicy = podcastViewModel::updatePolicy,
            devicePolicy = podcastViewModel::devicePolicy,
            updateDevicePolicy = podcastViewModel::updateDevicePolicy,
        ),
        videos = VideoBindings(
            state = videoState,
            open = videoViewModel::open,
            artworkUrl = videoViewModel::artworkUrl,
            authorizationHeader = videoViewModel.authorizationHeader(),
            openShow = { show -> videoViewModel.openShow(show); videoViewModel.loadDetails(show, null) },
            closeShow = videoViewModel::closeShow,
            selectSeason = videoViewModel::selectSeason,
            setLayout = videoViewModel::setLayout,
            setSearchOpen = videoViewModel::setSearchOpen,
            search = videoViewModel::search,
            play = onWatch,
            dismissContinueWatching = videoViewModel::dismissContinueWatching,
            setWatched = videoViewModel::setWatched,
            loadMoreShows = videoViewModel::loadMoreShows,
            loadMoreItems = videoViewModel::loadMoreItems,
            retry = videoViewModel::retry,
            loadMovieDetails = { item -> videoViewModel.loadDetails(null, item) },
        ),
        wiki = WikiBindings(wikiState, wikiViewModel::open, wikiViewModel::refresh, wikiViewModel::search, wikiViewModel::toggle, wikiViewModel::select, wikiViewModel::edit, wikiViewModel::draft, wikiViewModel::save, wikiViewModel::toggleTask, wikiViewModel::closePage, wikiViewModel::create, wikiViewModel::move, wikiViewModel::openLink, wikiViewModel::dismissMissingLink, wikiViewModel::createMissingLink, wikiViewModel::consumeNavigationHeading, wikiViewModel::trash, wikiViewModel::saveOffline, wikiViewModel::assetUrl, libraryViewModel.authorizationHeader(), wikiViewModel::cancelAttachment),
        coverUrl = libraryViewModel::coverUrl,
        seriesCoverUrl = libraryViewModel::seriesCoverUrl,
        authorCoverUrl = libraryViewModel::authorCoverUrl,
        collectionCoverUrl = libraryViewModel::collectionCoverUrl,
        authorizationHeader = libraryViewModel.authorizationHeader(),
        onSelectLibrary = libraryViewModel::select,
        onPublicationGrouping = libraryViewModel::setGrouping,
        onPublicationSort = libraryViewModel::setSort,
        onPublicationLayout = libraryViewModel::setLayout,
        onOpenSeries = libraryViewModel::openSeries,
        onCloseSeries = libraryViewModel::closeSeries,
        onOpenAuthor = libraryViewModel::openAuthor,
        onCloseAuthor = libraryViewModel::closeAuthor,
        onOpenCollection = libraryViewModel::openCollection,
        // A collection reached from the combined Collections place goes back there, not to this one library's list.
        onCloseCollection = {
            val fromAll = libraryState.collectionFromAll
            libraryViewModel.closeCollection()
            if (fromAll) onOpenCollections()
        },
        onRetrySeries = libraryViewModel::retrySeriesBooks,
        onReadBook = libraryViewModel::readBook,
        onQueueBook = { libraryViewModel.queueBook(it) },
        downloadQueue = libraryViewModel.downloadQueueItems,
        onCloseBook = libraryViewModel::closeBook,
        onSearchPublications = libraryViewModel::search,
        onPublicationBucket = libraryViewModel::selectBucket,
        onDownloadedOnly = libraryViewModel::setDownloadedOnly,
        publicationCollections = PublicationCollectionActions(collectionViewModel::open, collectionViewModel::open, libraryViewModel::renameCollection, libraryViewModel::deleteCollection),
        onLoadMore = libraryViewModel::loadMoreVisible,
        onRetry = libraryViewModel::refresh,
        onOpenMusic = musicViewModel::open,
        audioArtworkUrl = musicViewModel::artworkUrl,
        audioGroupArtworkUrl = musicViewModel::artworkUrl,
        onOpenAudioGroup = musicViewModel::openGroup,
        onOpenAudioAlbum = musicViewModel::openAlbum,
        onCloseAudioGroup = musicViewModel::closeGroup,
        onCloseAudioArtist = musicViewModel::closeArtist,
        onMusicGrouping = musicViewModel::setGrouping,
        onMusicLayout = musicViewModel::setLayout,
        onSearchMusic = musicViewModel::search,
        onDownloadedMusicOnly = musicViewModel::setDownloadedOnly,
        onFavoriteMusicOnly = musicViewModel::setFavoritesOnly,
        onShowAllMusic = musicViewModel::showAllTracks,
        onToggleMusicDownload = musicViewModel::toggleDownload,
        onToggleMusicFavorite = musicViewModel::toggleFavorite,
        onPlayMusicNext = { track ->
            queueViewModel.playNext(track, playbackViewModel.state.value.current)
            playbackViewModel.playNext(track, musicViewModel::playbackUrl)
            musicViewModel.notifyAction("Playing next.")
        },
        onAddMusicToQueue = { track ->
            queueViewModel.append(track)
            playbackViewModel.addToQueue(track, musicViewModel::playbackUrl)
            musicViewModel.notifyAction("Added to queue.")
        },
        musicPlaylists = MusicPlaylistBindings(
            artworkUrl = playlistViewModel::artworkUrl,
            state = playlistState,
            select = playlistViewModel::select,
            play = { tracks, selected ->
                queueViewModel.replace(tracks)
                playbackViewModel.play(tracks, selected, musicViewModel::playbackUrl)
            },
            move = playlistViewModel::moveTrack,
            remove = playlistViewModel::removeTrack,
            rename = playlistViewModel::rename,
            delete = playlistViewModel::delete,
            add = { track, name -> playlistViewModel.addTrack(name, track) },
            offline = PlaylistOfflineBindings(
                downloadedIds = musicState.downloadedIds,
                busy = musicState.batchDownloadTotal > 0,
                completed = musicState.batchDownloadCompleted.takeIf {
                    musicState.batchDownloadLabel == "playlist"
                } ?: 0,
                total = musicState.batchDownloadTotal.takeIf {
                    musicState.batchDownloadLabel == "playlist"
                } ?: 0,
                download = { musicViewModel.downloadTracks(it, "playlist") },
                remove = { musicViewModel.removeDownloads(it, "playlist") },
            ),
        ),
        onDownloadMusicGroup = musicViewModel::downloadSelectedGroup,
        onQueueMusicGroup = { tracks, next ->
            if (next) {
                queueViewModel.playNextAll(tracks, playbackViewModel.state.value.current)
                playbackViewModel.playNext(tracks, musicViewModel::playbackUrl)
                musicViewModel.notifyAction("Group will play next.")
            } else {
                queueViewModel.appendAll(tracks)
                playbackViewModel.addToQueue(tracks, musicViewModel::playbackUrl)
                musicViewModel.notifyAction("Group added to queue.")
            }
        },
        onPlayMusicGroup = { tracks, shuffle ->
            val queue = if (shuffle) tracks.shuffled() else tracks
            if (queue.isNotEmpty()) {
                queueViewModel.replace(queue)
                playbackViewModel.play(queue, queue.first(), musicViewModel::playbackUrl)
            }
        },
        onPlayMusicArtist = { shuffle ->
            musicViewModel.artistQueue(shuffle) { queue ->
                if (queue.isNotEmpty()) {
                    queueViewModel.replace(queue)
                    playbackViewModel.play(queue, queue.first(), musicViewModel::playbackUrl)
                }
            }
        },
        onLoadMoreAudio = musicViewModel::loadMore,
        onRetryAudio = musicViewModel::retry,
        onPlayAudio = { track ->
            val visible = musicState.visibleTracks()
            val queue = visible.takeIf { tracks ->
                tracks.any { it.libraryId == track.libraryId && it.id == track.id }
            } ?: listOf(track)
            queueViewModel.replace(queue)
            playbackViewModel.play(queue, track, musicViewModel::playbackUrl)
        },
        showCollections = showCollections,
        onOpenCollections = onOpenCollections,
        onStatusFilter = libraryViewModel::setStatus,
    )

    CollectionPickerSheet(
        collectionState,
        collectionViewModel::toggle,
        collectionViewModel::create,
        libraryViewModel::showBookDetails,
        collectionViewModel::setReadingStatus,
        collectionViewModel::close,
    )
}
