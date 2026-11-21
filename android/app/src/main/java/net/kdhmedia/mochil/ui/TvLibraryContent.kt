package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.Alignment
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.background
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.layout.Box
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.AudioGroup
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicLayout
import net.kdhmedia.mochil.audio.MusicState
import net.kdhmedia.mochil.audio.stableKey
import net.kdhmedia.mochil.audio.visibleTracks
import net.kdhmedia.mochil.catalogue.AuthorSummary
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.catalogue.SeriesSummary
import net.kdhmedia.mochil.podcast.PodcastShow
import net.kdhmedia.mochil.podcast.PodcastState
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoShow
import net.kdhmedia.mochil.video.VideoState

private fun podcastShowFocusKey(value: PodcastShow) = "podcast-show:${value.id}"
private fun podcastEpisodeFocusKey(value: AudioTrack) = "podcast-episode:${value.id}"
private fun bookFocusKey(value: BookSummary) = "book:${value.id}"
private fun authorFocusKey(value: AuthorSummary) = "author:${value.name}"
private fun seriesFocusKey(value: SeriesSummary) = "series:${value.name}"
private fun videoShowFocusKey(value: VideoShow) = "video-show:${value.name}"
private fun videoFocusKey(value: VideoItem) = "video:${value.id}"
private fun audioGroupFocusKey(value: AudioGroup) = "audio-group:${value.stableKey()}"
private fun audioTrackFocusKey(value: AudioTrack) = "audio-track:${value.id}"

internal fun tvHasVisibleFocusKey(
    key: String?,
    library: LibraryState,
    music: MusicState,
    video: VideoState,
    podcasts: PodcastState,
    visibleMusicTracks: List<AudioTrack> = music.visibleTracks(),
): Boolean {
    if (key.isNullOrBlank()) return false
    return when (library.selected?.kind) {
        "podcasts" -> if (podcasts.selectedShow == null)
            podcasts.shows.any { podcastShowFocusKey(it) == key }
        else podcasts.episodes.any { podcastEpisodeFocusKey(it) == key }
        "movies", "tv", "anime" -> if (video.selectedShow == null && video.shows.isNotEmpty())
            video.shows.any { videoShowFocusKey(it) == key }
        else video.items.any { videoFocusKey(it) == key }
        "music", "audiobooks" -> {
            val groups = tvAudioGroups(music)
            if (music.selectedGroup == null && groups.isNotEmpty() && music.grouping != MusicGrouping.Discover)
                groups.any { audioGroupFocusKey(it) == key }
            else (if (music.grouping == MusicGrouping.Discover) tvDiscoveryTracks(music) else visibleMusicTracks)
                .any { audioTrackFocusKey(it) == key }
        }
        else -> when {
            tvPublicationShowsBooks(library) -> tvPublicationBooks(library).any { bookFocusKey(it) == key }
            library.grouping == PublicationGrouping.Authors -> library.authors.any { authorFocusKey(it) == key }
            else -> library.series.any { seriesFocusKey(it) == key }
        }
    }
}

internal fun tvPublicationInsideGroup(state: LibraryState): Boolean =
    state.selectedCollection != null || state.selectedSeries != null || state.selectedAuthor != null

internal fun tvPublicationShowsBooks(state: LibraryState): Boolean =
    tvPublicationInsideGroup(state) || state.grouping in setOf(PublicationGrouping.Continue, PublicationGrouping.Books, PublicationGrouping.Recent)

internal fun tvPublicationBooks(state: LibraryState): List<BookSummary> =
    if (tvPublicationInsideGroup(state)) state.seriesBooks else state.books

@Composable
internal fun TvPodcastGrid(state: PodcastState, auth: String?, showArt: (PodcastShow) -> String, episodeArt: (AudioTrack) -> String, onShow: (PodcastShow) -> Unit, onEpisode: (AudioTrack) -> Unit, onSetPlayed: (AudioTrack, Boolean) -> Unit, onMore: () -> Unit, onFocus: (TvFocusDetail) -> Unit, positionKey: String, positions: TvGridPositionStore) {
    val paging = TvPagingState(state.nextCursor != null || state.nextOffset != null, state.loadingMore, onMore)
    var backdropArt by remember { mutableStateOf("") }
    TvBackdropEffect(backdropArt)
    if (state.selectedShow == null) {
        TvMediaGrid(paging, minColumnWidth = 130.dp, positionKey = positionKey, positions = positions, focusIndex = { key -> state.shows.indexOfFirst { podcastShowFocusKey(it) == key } }) {
            items(state.shows, key = PodcastShow::id, contentType = { "podcast-show" }) { value -> TvMediaCard(value.title, "${value.episodeCount} episodes", showArt(value), auth, TvArtworkRatio.Square, { onShow(value) }, focusIdentity = TvGridFocusIdentity(positionKey, podcastShowFocusKey(value), positions)) { backdropArt = showArt(value); onFocus(TvFocusDetail(value.title, "${value.episodeCount} episodes", tvFocusDescription(value.description))) } }
        }
    } else {
        TvMediaGrid(paging, minColumnWidth = 270.dp, positionKey = positionKey, positions = positions, focusIndex = { key -> state.episodes.indexOfFirst { podcastEpisodeFocusKey(it) == key } }) {
            items(state.episodes, key = AudioTrack::id, contentType = { "podcast-episode-row" }) { value ->
                var menu by remember(value.libraryId, value.id) { mutableStateOf(false) }
                val updating = value.identityKey in state.updatingPlayedIds
                val resolvingPlayback = state.resolvingPlaybackId == value.identityKey
                val progress = tvPodcastProgress(value)
                Box {
                    TvWideMediaCard(
                        value.title,
                        if (resolvingPlayback) "Opening from synced position…" else if (value.finished) "Played" else value.artist,
                        episodeArt(value), auth,
                        { onEpisode(value) },
                        progress = progress.fraction,
                        completed = progress.completed,
                        onLongClick = { menu = true },
                        focusIdentity = TvGridFocusIdentity(positionKey, podcastEpisodeFocusKey(value), positions),
                    ) { backdropArt = episodeArt(value); onFocus(TvFocusDetail(value.title, audioFocusContext(value))) }
                    TvContextDropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                        DropdownMenuItem(
                            text = { Text(tvPodcastPlayedActionLabel(value.finished, updating)) },
                            enabled = !updating,
                            onClick = { menu = false; onSetPlayed(value, !value.finished) },
                            modifier = androidx.compose.ui.Modifier.tvFocusBorder(),
                        )
                    }
                }
            }
        }
    }
}

internal fun tvPodcastPlayedActionLabel(finished: Boolean, updating: Boolean): String = when {
    updating -> "Updating…"
    finished -> "Mark as unplayed"
    else -> "Mark as played"
}

@Composable
internal fun TvPublicationGrid(state: LibraryState, auth: String?, bookArt: (BookSummary) -> String, seriesArt: (SeriesSummary) -> String, authorArt: (AuthorSummary) -> String, onSeries: (SeriesSummary) -> Unit, onAuthor: (AuthorSummary) -> Unit, onBook: (BookSummary) -> Unit, onMore: () -> Unit, onFocus: (TvFocusDetail) -> Unit, positionKey: String, positions: TvGridPositionStore) {
    val showingBooks = tvPublicationShowsBooks(state)
    val books = tvPublicationBooks(state)
    val insideGroup = tvPublicationInsideGroup(state)
    var backdropArt by remember { mutableStateOf("") }
    TvBackdropEffect(backdropArt)
    TvMediaGrid(TvPagingState(if (insideGroup) state.groupNextCursor != null else state.nextCursor != null, if (insideGroup) state.groupLoadingMore else state.loadingMore, onMore), minColumnWidth = 130.dp, positionKey = positionKey, positions = positions, focusIndex = { key ->
        when {
            showingBooks -> books.indexOfFirst { bookFocusKey(it) == key }
            state.grouping == PublicationGrouping.Authors -> state.authors.indexOfFirst { authorFocusKey(it) == key }
            else -> state.series.indexOfFirst { seriesFocusKey(it) == key }
        }
    }) {
        if (showingBooks) items(books, key = BookSummary::id, contentType = { "publication-cover" }) { value -> TvMediaCard(value.title, value.authors.joinToString(), bookArt(value), auth, TvArtworkRatio.Portrait, { onBook(value) }, progress = value.readingFraction?.toFloat() ?: 0f, completed = value.finished, focusIdentity = TvGridFocusIdentity(positionKey, bookFocusKey(value), positions)) { backdropArt = bookArt(value); onFocus(TvFocusDetail(value.title, publicationFocusContext(value), tvFocusDescription(value.description))) } }
        else if (state.grouping == PublicationGrouping.Authors) items(state.authors, key = AuthorSummary::name, contentType = { "publication-group-cover" }) { value -> TvMediaCard(value.name, "${value.count} ${if (value.count == 1) "book" else "books"}", authorArt(value), auth, TvArtworkRatio.Portrait, { onAuthor(value) }, focusIdentity = TvGridFocusIdentity(positionKey, authorFocusKey(value), positions)) { backdropArt = authorArt(value); onFocus(TvFocusDetail(value.name, "${value.count} ${if (value.count == 1) "book" else "books"}")) } }
        else items(state.series, key = SeriesSummary::name, contentType = { "publication-group-cover" }) { value -> TvMediaCard(value.name, "${value.count} ${if (value.count == 1) "book" else "books"}", seriesArt(value), auth, TvArtworkRatio.Portrait, { onSeries(value) }, focusIdentity = TvGridFocusIdentity(positionKey, seriesFocusKey(value), positions)) { backdropArt = seriesArt(value); onFocus(TvFocusDetail(value.name, listOfNotNull("${value.count} ${if (value.count == 1) "book" else "books"}", value.authors.joinToString().takeIf(String::isNotBlank)).joinToString(" · "))) } }
    }
}

@Composable
internal fun TvVideoGrid(state: VideoState, auth: String?, artwork: (String) -> String, onShow: (VideoShow) -> Unit, onVideo: (VideoItem) -> Unit, onSetWatched: (VideoItem, Boolean) -> Unit, onMore: () -> Unit, onFocus: (TvFocusDetail) -> Unit, positionKey: String, positions: TvGridPositionStore, onSeason: (Int) -> Unit, onDetails: (VideoShow?, VideoItem?) -> Unit = { _, _ -> }) {
    state.selectedShow?.let { show ->
        TvShowDetail(show, state, auth, artwork, onSeason, onVideo, onSetWatched, onMore, onFocus, onDetails)
        return
    }
    // Movies open a detail page first, like the other TV media apps; Back returns to the shelf.
    var openMovie by remember(positionKey) { mutableStateOf<VideoItem?>(null) }
    // Opening and closing the detail removes the focused card from the composition. With nothing focused, the
    // focus system falls back to a search that lands on the navigation rail and opens the panel over the film.
    // Take focus deliberately in both directions: the detail's Play control carries the content requester, and
    // entering the grid runs its own remembered-card restore.
    val contentFocus = LocalTvContentFocus.current
    var wasOpen by remember { mutableStateOf(false) }
    LaunchedEffect(openMovie) {
        if (openMovie != null || wasOpen) runCatching { contentFocus?.requestFocus() }
        wasOpen = openMovie != null
    }
    openMovie?.let { selected ->
        val movie = state.items.firstOrNull { it.id == selected.id } ?: selected
        androidx.activity.compose.BackHandler { openMovie = null }
        TvMovieDetail(movie, auth, artwork(movie.artworkPath), movie.backdropPath.takeIf(String::isNotBlank)?.let(artwork).orEmpty(), { onVideo(movie) }, { onSetWatched(movie, it) }, details = state.details["item:${movie.id}"], onDetails = { onDetails(null, movie) }, artwork = artwork)
        return
    }
    var spotlight by remember(positionKey) { mutableStateOf<TvSpotlight?>(null) }
    TvVideoPosters(state, auth, artwork, onShow, onVideo, onSetWatched, onMore, onFocus, positionKey, positions, spotlight, { spotlight = it }) { openMovie = it }
}


/**
 * How wide a card has to be on a television. A portrait poster wall at 118dp put six books or shows across a
 * 1080p set, where each cover was about an eighth of the screen height -- small enough that the wall read as a
 * contact sheet rather than as something to choose from. Leaning back, four across is the shape the covers
 * deserve; a shelf of wide stills stays narrower than a poster, because it has less to show.
 */
internal fun tvCardMinWidth(posterShelf: Boolean): Dp = if (posterShelf) 176.dp else 210.dp

/** The columns [tvCardMinWidth] actually produces, so the intent can be asserted instead of eyeballed. */
internal fun tvGridColumns(widthDp: Float, minColumnWidth: Dp, spacingDp: Float = 16f, contentPaddingDp: Float = 80f): Int {
    val usable = widthDp - contentPaddingDp
    val step = minColumnWidth.value + spacingDp
    return maxOf(1, ((usable + spacingDp) / step).toInt())
}

@Composable
private fun TvVideoPosters(state: VideoState, auth: String?, artwork: (String) -> String, onShow: (VideoShow) -> Unit, onVideo: (VideoItem) -> Unit, onSetWatched: (VideoItem, Boolean) -> Unit, onMore: () -> Unit, onFocus: (TvFocusDetail) -> Unit, positionKey: String, positions: TvGridPositionStore, spotlight: TvSpotlight?, onSpotlight: (TvSpotlight) -> Unit, onOpenMovie: (VideoItem) -> Unit) {
    val paging = TvPagingState(state.nextOffset != null && state.pagingError == null, state.loadingMore, onMore, state.pagingError, onMore)
    // The library level is always a poster wall like Plex and Emby, matched or not; the episodes inside a show stay wide stills.
    val posterShelf = state.selectedShow == null
    val ratio = if (posterShelf) TvArtworkRatio.Portrait else TvArtworkRatio.Landscape
    TvMediaGrid(paging, minColumnWidth = tvCardMinWidth(posterShelf), positionKey = positionKey, positions = positions, focusIndex = { key ->
        val index = if (state.selectedShow == null && state.shows.isNotEmpty()) state.shows.indexOfFirst { videoShowFocusKey(it) == key } else state.items.indexOfFirst { videoFocusKey(it) == key }
        if (index < 0) -1 else index + 1 // spotlight occupies the first grid slot
    }) {
        // The spotlight is the first row of the grid, so it scrolls away with it instead of clipping the cards under a fixed panel.
        item(key = "spotlight", span = { androidx.compose.foundation.lazy.grid.GridItemSpan(maxLineSpan) }, contentType = "spotlight") { TvSpotlightPanel(spotlight, auth, Modifier.fillMaxWidth().height(236.dp)) }
        if (state.selectedShow == null && state.shows.isNotEmpty()) items(state.shows, key = VideoShow::name, contentType = { "video-show-cover" }) { value -> TvMediaCard(tvCleanTitle(value.name), listOfNotNull(value.year.takeIf { it > 0 && posterShelf }?.toString(), tvCount(value.count, "episode")).joinToString(" · "), artwork(value.artworkPath), auth, ratio, { onShow(value) }, progress = value.watchFraction, completed = value.completed, focusIdentity = TvGridFocusIdentity(positionKey, videoShowFocusKey(value), positions)) { onSpotlight(TvSpotlight(tvCleanTitle(value.name), tvShowMeta(value), value.overview, artwork(value.artworkPath), value.backdropPath.takeIf(String::isNotBlank)?.let(artwork).orEmpty())); onFocus(TvFocusDetail(value.name, "")) } }
        else items(state.items, key = VideoItem::id, contentType = { "video-episode-cover" }) { value ->
            var menu by remember(value.libraryId, value.id) { mutableStateOf(false) }
            Box {
                TvMediaCard(
                    tvCleanTitle(value.displayTitle),
                    listOfNotNull(value.show.takeIf { it.isNotBlank() && state.selectedShow == null }, value.episode.takeIf { it > 0 }?.let { "S${value.season} · E$it" }, value.year.takeIf { it > 0 && posterShelf && value.episode <= 0 }?.toString()).joinToString(" · "),
                    artwork(value.artworkPath), auth,
                    ratio,
                    { if (state.libraryKind == "movies") onOpenMovie(value) else onVideo(value) },
                    progress = value.progressFraction,
                    completed = value.finished,
                    onLongClick = { menu = true },
                    focusIdentity = TvGridFocusIdentity(positionKey, videoFocusKey(value), positions),
                ) { onSpotlight(TvSpotlight(tvCleanTitle(value.displayTitle), tvMovieMeta(value), value.overview, artwork(value.artworkPath), value.backdropPath.takeIf(String::isNotBlank)?.let(artwork).orEmpty())); onFocus(TvFocusDetail(value.displayTitle, "")) }
                TvContextDropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                    DropdownMenuItem(
                        text = { Text(if (value.finished) "Mark as unwatched" else "Mark as watched") },
                        onClick = { menu = false; onSetWatched(value, !value.finished) },
                        modifier = androidx.compose.ui.Modifier.tvFocusBorder(),
                    )
                }
            }
        }
    }
}

@Composable
internal fun TvAudioGrid(state: MusicState, visibleTracks: List<AudioTrack>, auth: String?, groupArt: (AudioGroup) -> String, trackArt: (AudioTrack) -> String, onGroup: (AudioGroup) -> Unit, onTrack: (AudioTrack) -> Unit, onPlayNext: (AudioTrack) -> Unit, onAddToQueue: (AudioTrack) -> Unit, onToggleFavorite: (AudioTrack) -> Unit, onMore: () -> Unit, onFocus: (TvFocusDetail) -> Unit, positionKey: String, positions: TvGridPositionStore) {
    val paging = TvPagingState(state.nextCursor != null || state.nextOffset != null, state.loadingMore, onMore)
    var backdropArt by remember { mutableStateOf("") }
    TvBackdropEffect(backdropArt)
    val visibleGroups = tvAudioGroups(state)
    if (state.selectedGroup == null && visibleGroups.isNotEmpty() && state.grouping != MusicGrouping.Discover) {
        TvMediaGrid(paging, minColumnWidth = if (state.layout == MusicLayout.List) 270.dp else 130.dp, positionKey = positionKey, positions = positions, focusIndex = { key -> visibleGroups.indexOfFirst { audioGroupFocusKey(it) == key } }) {
            items(visibleGroups, key = AudioGroup::stableKey, contentType = { if (state.layout == MusicLayout.List) "audio-group-row" else "audio-group-cover" }) { value ->
                val focused = { backdropArt = groupArt(value); onFocus(TvFocusDetail(value.name, listOf(value.subtitle, "${value.count} items").filter(String::isNotBlank).joinToString(" · "))) }
                val focusIdentity = TvGridFocusIdentity(positionKey, audioGroupFocusKey(value), positions)
                if (state.layout == MusicLayout.List) TvWideMediaCard(value.name, value.subtitle, groupArt(value), auth, { onGroup(value) }, focusIdentity = focusIdentity, onFocused = focused)
                else TvMediaCard(value.name, value.subtitle, groupArt(value), auth, TvArtworkRatio.Square, { onGroup(value) }, focusIdentity = focusIdentity, onFocused = focused)
            }
        }
    } else {
        val tracks = remember(state.grouping, state.discoveryPlayed, state.discoveryRecent, state.favoriteTracks, visibleTracks) {
            if (state.grouping == MusicGrouping.Discover) tvDiscoveryTracks(state) else visibleTracks
        }
        TvMediaGrid(paging, minColumnWidth = if (state.layout == MusicLayout.List) 270.dp else 130.dp, positionKey = positionKey, positions = positions, focusIndex = { key -> tracks.indexOfFirst { audioTrackFocusKey(it) == key } }) {
            items(tracks, key = AudioTrack::id, contentType = { if (state.layout == MusicLayout.List) "audio-track-row" else "audio-track-cover" }) { value ->
                var menu by remember(value.libraryId, value.id) { mutableStateOf(false) }
                val focused = { backdropArt = trackArt(value); onFocus(TvFocusDetail(value.title, audioFocusContext(value))) }
                val progress = audiobookProgress(value, state.libraryKind)
                Box {
                    val openActions = if (value.libraryKind == "music") ({ menu = true }) else null
                    val focusIdentity = TvGridFocusIdentity(positionKey, audioTrackFocusKey(value), positions)
                    if (state.layout == MusicLayout.List) TvWideMediaCard(value.title, value.artist, trackArt(value), auth, { onTrack(value) }, progress.fraction, progress.completed, onLongClick = openActions, focusIdentity = focusIdentity, onFocused = focused)
                    else TvMediaCard(value.title, value.artist, trackArt(value), auth, TvArtworkRatio.Square, { onTrack(value) }, progress.fraction, progress.completed, onLongClick = openActions, focusIdentity = focusIdentity, onFocused = focused)
                    TvContextDropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                        tvMusicTrackActions(value, value.identityKey in state.favoriteIds).forEach { action ->
                            DropdownMenuItem(text = { Text(action.label) }, modifier = androidx.compose.ui.Modifier.tvFocusBorder(), onClick = {
                                menu = false
                                when (action.type) {
                                    TvMusicTrackActionType.PlayNext -> onPlayNext(value)
                                    TvMusicTrackActionType.AddToQueue -> onAddToQueue(value)
                                    TvMusicTrackActionType.ToggleFavorite -> onToggleFavorite(value)
                                }
                            })
                        }
                    }
                }
            }
        }
    }
}

internal enum class TvMusicTrackActionType { PlayNext, AddToQueue, ToggleFavorite }
internal data class TvMusicTrackAction(val type: TvMusicTrackActionType, val label: String)

internal fun tvMusicTrackActions(track: AudioTrack, favorite: Boolean): List<TvMusicTrackAction> = buildList {
    if (track.libraryKind != "music") return@buildList
    add(TvMusicTrackAction(TvMusicTrackActionType.PlayNext, "Play next"))
    add(TvMusicTrackAction(TvMusicTrackActionType.AddToQueue, "Add to queue"))
    add(TvMusicTrackAction(TvMusicTrackActionType.ToggleFavorite, if (favorite) "Remove from favorites" else "Add to favorites"))
}

internal fun tvAudioGroups(state: MusicState): List<AudioGroup> =
    if (state.selectedArtist != null && state.selectedGroup == null) state.artistAlbums else state.groups

/** A dense TV discovery feed: resume familiar music first, then surface new tracks. */
internal fun tvDiscoveryTracks(state: MusicState): List<AudioTrack> =
    sequenceOf(state.discoveryPlayed, state.discoveryRecent, state.favoriteTracks)
        .flatten()
        .distinctBy(AudioTrack::identityKey)
        .take(80)
        .toList()

internal fun tvPlaybackTracks(state: MusicState): List<AudioTrack> =
    if (state.grouping == MusicGrouping.Discover) tvDiscoveryTracks(state) else state.visibleTracks()

internal data class TvAudioProgress(val fraction: Float, val completed: Boolean)

internal fun tvPodcastProgress(track: AudioTrack): TvAudioProgress {
    val fraction = if (track.durationSeconds > 0.0)
        (track.resumePositionSeconds / track.durationSeconds).toFloat().coerceIn(0f, 1f)
    else 0f
    return TvAudioProgress(fraction, track.finished)
}

internal fun audiobookProgress(track: AudioTrack, libraryKind: String): TvAudioProgress {
    if (libraryKind != "audiobooks") return TvAudioProgress(0f, false)
    val fraction = if (track.durationSeconds > 0.0)
        (track.resumePositionSeconds / track.durationSeconds).toFloat().coerceIn(0f, 1f)
    else 0f
    return TvAudioProgress(fraction, track.finished)
}

private fun publicationFocusContext(book: BookSummary): String = listOfNotNull(
    book.authors.filterNot { it.equals("unknown", ignoreCase = true) }.joinToString().takeIf(String::isNotBlank),
    book.series.takeIf(String::isNotBlank)?.let { series -> book.seriesIndex?.takeIf { it > 0 }?.let { "$series · Volume ${it.compactNumber()}" } ?: series },
    book.format.uppercase().takeIf(String::isNotBlank),
).joinToString(" · ").ifBlank { "Open publication" }

private fun audioFocusContext(track: AudioTrack): String = listOfNotNull(
    track.artist.takeIf(String::isNotBlank), track.album.takeIf(String::isNotBlank),
    track.durationSeconds.takeIf { it > 0 }?.let(::formatTvDuration),
    track.resumePositionSeconds.takeIf { it > 0 && !track.finished }?.let { "Resume at ${formatTvDuration(it)}" },
).joinToString(" · ").ifBlank { "Play audio" }

private fun formatTvDuration(seconds: Double): String {
    val total = seconds.toLong().coerceAtLeast(0)
    val hours = total / 3600
    val minutes = total % 3600 / 60
    val remainingSeconds = total % 60
    return if (hours > 0) "%d:%02d:%02d".format(hours, minutes, remainingSeconds) else "%d:%02d".format(minutes, remainingSeconds)
}

private fun Double.compactNumber(): String = if (this % 1.0 == 0.0) toLong().toString() else toString().trimEnd('0').trimEnd('.')

internal data class TvSpotlight(val title: String, val meta: String, val overview: String, val art: String, val backdrop: String = "")

/** The focused poster's art spread behind its title, facts and synopsis, so a shelf reads like the other TV media apps. */
@Composable
private fun TvSpotlightPanel(spotlight: TvSpotlight?, auth: String?, modifier: Modifier) {
    val background = MaterialTheme.colorScheme.background
    Box(modifier) {
        spotlight?.let { current ->
            TvBackdropEffect(current.backdrop.ifBlank { current.art })
            Column(Modifier.fillMaxWidth(.5f).padding(start = 8.dp, top = 64.dp), verticalArrangement = Arrangement.spacedBy(5.dp)) {
                Text(current.title, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold, maxLines = 2, overflow = TextOverflow.Ellipsis)
                if (current.meta.isNotBlank()) Text(current.meta, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1)
                if (current.overview.isNotBlank()) Text(current.overview, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 2, overflow = TextOverflow.Ellipsis)
            }
        }
    }
}
