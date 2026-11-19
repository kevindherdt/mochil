package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items as rowItems
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ViewList
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.GridView
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.VerticalDivider
import androidx.activity.compose.BackHandler
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.CustomAccessibilityAction
import androidx.compose.ui.semantics.customActions
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.catalogue.catalogueItemIdentityKey
import net.kdhmedia.mochil.video.VideoLayout
import net.kdhmedia.mochil.video.VideoShow
import net.kdhmedia.mochil.video.VideoState

/** The identity the open movie is remembered by, so the detail survives a rotation without keeping a stale copy. */
internal fun videoMovieSelectionKey(item: VideoItem): String = catalogueItemIdentityKey(item.libraryId, item.id)

/**
 * The movie to show, rebuilt from the shelf the ViewModel owns. Only the key is saved across recreation, so a
 * refreshed item (progress, watched state) is what the reopened detail draws, and a movie that has left the shelf
 * closes instead of lingering as a copy nothing updates.
 */
internal fun videoSelectedMovie(items: List<VideoItem>, key: String?): VideoItem? =
    key?.let { wanted -> items.firstOrNull { catalogueItemIdentityKey(it.libraryId, it.id) == wanted } }

@Composable
fun VideoScreen(
    state: VideoState,
    artworkUrl: (String) -> String,
    authorizationHeader: String?,
    onOpenShow: (VideoShow) -> Unit,
    onCloseShow: () -> Unit,
    onSeason: (Int) -> Unit,
    onLayout: (VideoLayout) -> Unit,
    onSearchOpen: (Boolean) -> Unit,
    onSearch: (String) -> Unit,
    onPlay: (VideoItem) -> Unit,
    onDismissContinueWatching: (VideoItem) -> Unit,
    onSetWatched: (VideoItem, Boolean) -> Unit,
    onLoadMoreShows: () -> Unit,
    onLoadMoreItems: () -> Unit,
    onRetry: () -> Unit,
    onLoadMovieDetails: (VideoItem) -> Unit = {},
) {
    // Only the movie's catalogue key is saved across a rotation: the item itself is looked up from the shelf again
    // (right below), so the reopened detail shows the progress the ViewModel has refreshed rather than a stale copy.
    // A `remember` here dropped the selection on every rotation, since the activity declares no configChanges.
    var selectedMovieKey by rememberSaveable(state.libraryId) { mutableStateOf<String?>(null) }
    val movie = videoSelectedMovie(state.items, selectedMovieKey)
    if (movie != null) {
        BackHandler { selectedMovieKey = null }
        LaunchedEffect(movie.libraryId, movie.id) { onLoadMovieDetails(movie) }
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                IconButton(onClick = { selectedMovieKey = null }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back to movies") }
                Text(movie.displayTitle, style = MaterialTheme.typography.titleLarge)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(16.dp), verticalAlignment = Alignment.Top) {
                AuthenticatedArtwork(movie.artworkPath.takeIf(String::isNotBlank)?.let(artworkUrl).orEmpty(), authorizationHeader, movie.displayTitle, Modifier.width(110.dp).aspectRatio(2f / 3f))
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    val details = state.details["item:${movie.id}"]
                    val facts = tvFactsLine(movie.year, details).ifBlank { tvMovieMeta(movie) }
                    if (facts.isNotBlank()) Text(facts, style = MaterialTheme.typography.bodyMedium)
                    details?.certification?.takeIf(String::isNotBlank)?.let { Text(it, style = MaterialTheme.typography.labelMedium) }
                    tvScoreLabel(details)?.let { Text("$it TMDB", style = MaterialTheme.typography.bodyMedium) }
                }
            }
            if (movie.overview.isNotBlank()) Text(movie.overview)
            Button(onClick = { onPlay(movie) }, enabled = movie.playable) { Text(if (movie.resumeSeconds > 0 && !movie.finished) "Resume" else "Play") }
            VideoCastRow(state.details["item:${movie.id}"], artworkUrl, authorizationHeader, television = false)
        }
        return
    }
    val openOrPlay: (VideoItem) -> Unit = { item ->
        if (state.libraryKind == "movies" && state.selectedShow == null) selectedMovieKey = videoMovieSelectionKey(item) else onPlay(item)
    }
    val openShowWithoutHiddenSearch: (VideoShow) -> Unit = { show ->
        if (state.query.isNotBlank()) onSearch("")
        onOpenShow(show)
    }
    BoxWithConstraints(Modifier.fillMaxSize()) {
        if (usesMediaTwoPane(maxWidth.value.toInt(), state.selectedShow != null)) {
            BackHandler(onBack = onCloseShow)
            Row(Modifier.fillMaxSize()) {
                Column(Modifier.weight(.43f)) {
                    Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text("Shows", Modifier.weight(1f), style = MaterialTheme.typography.titleLarge)
                        IconButton(onClick = { onLayout(if (state.layout == VideoLayout.Covers) VideoLayout.List else VideoLayout.Covers) }) {
                            Icon(if (state.layout == VideoLayout.Covers) Icons.AutoMirrored.Filled.ViewList else Icons.Default.GridView, if (state.layout == VideoLayout.Covers) "Use list view" else "Use cover view")
                        }
                    }
                    VideoRoot(state.copy(selectedShow = null, selectedSeason = null, items = emptyList(), continueWatching = emptyList(), loading = false, error = null), artworkUrl, authorizationHeader, openShowWithoutHiddenSearch, onPlay, onDismissContinueWatching, onSetWatched, onLoadMoreShows, onLoadMoreItems)
                }
                VerticalDivider()
                Box(Modifier.weight(.57f)) { VideoSinglePane(state, artworkUrl, authorizationHeader, onOpenShow, onCloseShow, onSeason, onLayout, onSearchOpen, onSearch, onPlay, onDismissContinueWatching, onSetWatched, onLoadMoreShows, onLoadMoreItems, onRetry) }
            }
        } else {
            VideoSinglePane(state, artworkUrl, authorizationHeader, openShowWithoutHiddenSearch, onCloseShow, onSeason, onLayout, onSearchOpen, onSearch, openOrPlay, onDismissContinueWatching, onSetWatched, onLoadMoreShows, onLoadMoreItems, onRetry, onPlay)
        }
    }
}

@Composable
private fun VideoSinglePane(
    state: VideoState,
    artworkUrl: (String) -> String,
    authorizationHeader: String?,
    onOpenShow: (VideoShow) -> Unit,
    onCloseShow: () -> Unit,
    onSeason: (Int) -> Unit,
    onLayout: (VideoLayout) -> Unit,
    onSearchOpen: (Boolean) -> Unit,
    onSearch: (String) -> Unit,
    onPlay: (VideoItem) -> Unit,
    onDismissContinueWatching: (VideoItem) -> Unit,
    onSetWatched: (VideoItem, Boolean) -> Unit,
    onLoadMoreShows: () -> Unit,
    onLoadMoreItems: () -> Unit,
    onRetry: () -> Unit,
    onContinuePlay: (VideoItem) -> Unit = onPlay,
) {
    state.selectedShow?.let { show ->
        BackHandler(onBack = onCloseShow)
        MobileShowDetail(show, state, artworkUrl, authorizationHeader, onCloseShow, onSeason, onPlay, onSetWatched, onLoadMoreItems, onRetry)
        return
    }
    Column(Modifier.fillMaxSize()) {
        VideoToolbar(state, onLayout, onSearchOpen, onSearch)
        // Returning to the app re-reads progress behind whatever is on screen, so a refresh can arrive with the
        // shelf already populated. Only an actually empty screen is a load in progress (or a failure worth owning
        // the whole screen): otherwise the viewer keeps the grid they were on instead of a spinner and a reset scroll.
        val nothingToShow = state.shows.isEmpty() && state.items.isEmpty()
        when {
            state.loading && nothingToShow -> AccessibleLoadingIndicator("Loading videos", Modifier.fillMaxSize())
            state.error != null && nothingToShow -> Column(Modifier.fillMaxSize().padding(32.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterVertically)) {
                Text(state.error, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Button(onClick = onRetry) { Text("Try again") }
            }
            state.selectedShow == null && (state.shows.isNotEmpty() || state.items.isNotEmpty()) -> VideoRoot(state, artworkUrl, authorizationHeader, onOpenShow, onPlay, onDismissContinueWatching, onSetWatched, onLoadMoreShows, onLoadMoreItems, onContinuePlay)
            state.items.isEmpty() -> Column(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
                Text("No videos found.", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            else -> VideoItems(state, artworkUrl, authorizationHeader, onPlay, onSetWatched, onLoadMoreItems, onLoadMoreItems)
        }
    }
}

@Composable
private fun MobileShowDetail(
    show: VideoShow, state: VideoState, artworkUrl: (String) -> String, auth: String?,
    onBack: () -> Unit, onSeason: (Int) -> Unit, onPlay: (VideoItem) -> Unit,
    onSetWatched: (VideoItem, Boolean) -> Unit, onMore: () -> Unit, onRetry: () -> Unit,
) {
    val details = state.details["show:${show.sourceName}"]

    LazyColumn(Modifier.fillMaxSize()) {
        item(key = "show-header") {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back to shows") }
                    Text("Shows", style = MaterialTheme.typography.titleMedium)
                }
                Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                    AuthenticatedArtwork(show.artworkPath.takeIf(String::isNotBlank)?.let(artworkUrl).orEmpty(), auth, show.name, Modifier.width(110.dp).aspectRatio(2f / 3f))
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(show.name, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                        Text(tvShowMeta(show), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        details?.let { value ->
                            val facts = tvFactsLine(0, value)
                            if (facts.isNotBlank()) Text(facts, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            value.certification.takeIf(String::isNotBlank)?.let { Text(it, style = MaterialTheme.typography.labelMedium) }
                        }

                    }
                }
                if (show.overview.isNotBlank()) Text(show.overview, Modifier.padding(horizontal = 16.dp), style = MaterialTheme.typography.bodyMedium)
                VideoCastRow(details, artworkUrl, auth, television = false)
                if (show.seasons.size > 1) LazyRow(Modifier.fillMaxWidth().padding(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    rowItems(show.seasons) { season -> FilterChip(selected = state.selectedSeason == season, onClick = { onSeason(season) }, label = { Text(tvSeasonLabel(season)) }) }
                }
                Text(state.selectedSeason?.let { tvSeasonLabel(it) } ?: "Episodes", Modifier.padding(horizontal = 16.dp), style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
            }
        }
        when {
            // Same rule as the shelf: a refresh behind an episode list that is already on screen must not replace it.
            state.loading && state.items.isEmpty() -> item { AccessibleLoadingIndicator("Loading videos", Modifier.fillMaxWidth().height(100.dp)) }
            state.error != null && state.items.isEmpty() -> item { VideoPagingError(state.error, onRetry) }
            state.items.isEmpty() -> item { Text("No videos found.", Modifier.padding(16.dp), color = MaterialTheme.colorScheme.onSurfaceVariant) }
            else -> {
                rowItems(state.items, key = { catalogueItemIdentityKey(it.libraryId, it.id) }) { episode ->
                    VideoItemActionSurface(episode, onPlay, onSetWatched, openDetails = false) { actionModifier ->
                        Row(actionModifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp), horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.Top) {
                            Box(Modifier.width(128.dp).aspectRatio(16f / 9f)) {
                                AuthenticatedArtwork(artworkUrl(episode.artworkPath), auth, episode.displayTitle, Modifier.fillMaxSize())
                                episode.caption().fraction?.let { ThinProgress({ it }, Modifier.fillMaxWidth().height(3.dp).align(Alignment.BottomCenter)) }
                            }
                            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
                                Text(episode.displayTitle, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, maxLines = 2, overflow = TextOverflow.Ellipsis)
                                Text(listOf(tvEpisodeLabel(episode), episode.caption().text.takeIf { episode.finished || episode.resumeSeconds > 0 || !episode.playable }.orEmpty()).filter(String::isNotBlank).joinToString(" · "), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                if (episode.overview.isNotBlank()) Text(episode.overview, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 3, overflow = TextOverflow.Ellipsis)
                            }
                        }
                    }
                }
                videoPaging(state, onMore, onMore)
            }
        }
    }
}

@Composable
private fun VideoToolbar(state: VideoState, onLayout: (VideoLayout) -> Unit, onSearchOpen: (Boolean) -> Unit, onSearch: (String) -> Unit) {
    val covers = state.layout == VideoLayout.Covers
    HeaderActions(listOfNotNull(
        if (!state.searchOpen) HeaderAction(Icons.Default.Search, "Search videos", { onSearchOpen(true) }) else null,
        HeaderAction(if (covers) Icons.AutoMirrored.Filled.ViewList else Icons.Default.GridView, if (covers) "Use list view" else "Use cover view", { onLayout(if (covers) VideoLayout.List else VideoLayout.Covers) }),
    ))
    if (state.searchOpen) {
        // A local draft submitted on the IME action, the same shape as the book and music shelves. Sending every
        // keystroke through the ViewModel meant this field was re-written from a query that arrives a frame late, so a
        // fast typist's next character was replaced by the previous one before it reached the state.
        var draft by remember(state.libraryId, state.searchOpen) { mutableStateOf(state.query) }
        val focusManager = LocalFocusManager.current
        OutlinedTextField(
            value = draft,
            onValueChange = { draft = it },
            modifier = Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 4.dp),
            singleLine = true,
            placeholder = { Text("Search this library") },
            trailingIcon = { IconButton(onClick = { onSearchOpen(false) }) { Icon(Icons.Default.Close, "Close search") } },
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
            keyboardActions = KeyboardActions(onSearch = {
                onSearch(draft)
                focusManager.clearFocus()
            }),
        )
    }
}

@Composable
private fun VideoRoot(state: VideoState, artworkUrl: (String) -> String, authorizationHeader: String?, onOpen: (VideoShow) -> Unit, onPlay: (VideoItem) -> Unit, onDismissContinueWatching: (VideoItem) -> Unit, onSetWatched: (VideoItem, Boolean) -> Unit, onLoadMoreShows: () -> Unit, onLoadMoreItems: () -> Unit, onContinuePlay: (VideoItem) -> Unit = onPlay) {
    Column(Modifier.fillMaxSize()) {
        if (state.shows.isNotEmpty()) {
            if (state.layout == VideoLayout.Covers) VideoShowGrid(state, artworkUrl, authorizationHeader, onOpen, onLoadMoreShows, onLoadMoreShows, Modifier.weight(1f))
            else VideoShowList(state, artworkUrl, authorizationHeader, onOpen, onLoadMoreShows, onLoadMoreShows, Modifier.weight(1f))
        } else {
            VideoItems(state, artworkUrl, authorizationHeader, onPlay, onSetWatched, onLoadMoreItems, onLoadMoreItems, Modifier.weight(1f))
        }
    }
}

@Composable
private fun VideoShowGrid(state: VideoState, artworkUrl: (String) -> String, authorizationHeader: String?, onOpen: (VideoShow) -> Unit, onLoadMore: () -> Unit, onRetry: () -> Unit, modifier: Modifier = Modifier) {
    LazyVerticalGrid(GridCells.Adaptive(132.dp), modifier.fillMaxSize().padding(horizontal = 14.dp), horizontalArrangement = Arrangement.spacedBy(12.dp), verticalArrangement = Arrangement.spacedBy(18.dp)) {
        items(state.shows, key = { it.sourceName }) { show ->
            VideoCard(
                art = { AuthenticatedArtwork(artworkUrl(show.artworkPath), authorizationHeader, show.name, it) }, aspect = 2f / 3f,
                title = show.name, caption = show.caption(),
                modifier = Modifier.gridCard().clickable(onClickLabel = "Open ${show.name}", role = Role.Button) { onOpen(show) }.padding(10.dp),
            )
        }
        videoPaging(state, onLoadMore, onRetry)
    }
}

@Composable
private fun VideoShowList(state: VideoState, artworkUrl: (String) -> String, authorizationHeader: String?, onOpen: (VideoShow) -> Unit, onLoadMore: () -> Unit, onRetry: () -> Unit, modifier: Modifier = Modifier) {
    LazyColumn(modifier.fillMaxSize()) {
        rowItems(state.shows, key = { it.sourceName }) { show ->
            val seasons = if (show.seasons.size == 1) "1 season" else "${show.seasons.size} seasons"
            val caption = show.caption().let { it.copy(text = "${it.text} · $seasons") }
            VideoListRow(
                art = { AuthenticatedArtwork(artworkUrl(show.artworkPath), authorizationHeader, show.name, Modifier.size(40.dp, 60.dp)) },
                artWidth = 40.dp, artHeight = 60.dp, title = show.name, caption = caption,
                modifier = Modifier.clickable(onClickLabel = "Open ${show.name}", role = Role.Button) { onOpen(show) },
            )
        }
        videoPaging(state, onLoadMore, onRetry)
    }
}

@Composable
private fun VideoItems(state: VideoState, artworkUrl: (String) -> String, authorizationHeader: String?, onPlay: (VideoItem) -> Unit, onSetWatched: (VideoItem, Boolean) -> Unit, onLoadMore: () -> Unit, onRetry: () -> Unit, modifier: Modifier = Modifier) {
    if (state.layout == VideoLayout.List) {
        VideoItemList(state, artworkUrl, authorizationHeader, onPlay, onSetWatched, onLoadMore, onRetry, modifier)
    } else {
        LazyVerticalGrid(GridCells.Adaptive(132.dp), modifier.fillMaxSize().padding(horizontal = 14.dp), horizontalArrangement = Arrangement.spacedBy(12.dp), verticalArrangement = Arrangement.spacedBy(18.dp)) {
            items(state.items, key = { catalogueItemIdentityKey(it.libraryId, it.id) }) { item ->
                VideoItemActionSurface(item, onPlay, onSetWatched, openDetails = state.libraryKind == "movies" && state.selectedShow == null) { actionModifier ->
                    VideoCard(
                        art = { AuthenticatedArtwork(artworkUrl(item.artworkPath), authorizationHeader, item.displayTitle, it) },
                        aspect = if (item.episode > 0) 16f / 9f else 2f / 3f,
                        title = item.displayTitle, caption = item.caption(),
                        modifier = actionModifier.gridCard().padding(10.dp),
                    )
                }
            }
            videoPaging(state, onLoadMore, onRetry)
        }
    }
}

@Composable
private fun VideoItemList(state: VideoState, artworkUrl: (String) -> String, authorizationHeader: String?, onPlay: (VideoItem) -> Unit, onSetWatched: (VideoItem, Boolean) -> Unit, onLoadMore: () -> Unit, onRetry: () -> Unit, modifier: Modifier = Modifier) {
    LazyColumn(modifier.fillMaxSize()) {
        rowItems(state.items, key = { catalogueItemIdentityKey(it.libraryId, it.id) }) { item ->
            val episode = item.episode > 0
            val width = if (episode) 80.dp else 40.dp
            val height = if (episode) 45.dp else 60.dp
            val code = if (episode) "S${item.season.toString().padStart(2, '0')}E${item.episode.toString().padStart(2, '0')}" else null
            val caption = item.caption().let { c -> if (code == null) c else c.copy(text = listOf(code, c.text.removePrefix("Episode ${item.episode}").trim(' ', '·')).filter(String::isNotBlank).joinToString(" · ")) }
            VideoItemActionSurface(item, onPlay, onSetWatched, openDetails = state.libraryKind == "movies" && state.selectedShow == null) { actionModifier ->
                VideoListRow(
                    art = { AuthenticatedArtwork(artworkUrl(item.artworkPath), authorizationHeader, item.displayTitle, Modifier.size(width, height)) },
                    artWidth = width, artHeight = height, title = item.displayTitle, caption = caption, modifier = actionModifier,
                )
            }
        }
        videoPaging(state, onLoadMore, onRetry)
    }
}

private fun androidx.compose.foundation.lazy.grid.LazyGridScope.videoPaging(state: VideoState, onLoadMore: () -> Unit, onRetry: () -> Unit) {
    val error = state.pagingError
    if (error != null) item(key = "video-page-error", span = { androidx.compose.foundation.lazy.grid.GridItemSpan(maxLineSpan) }) {
        VideoPagingError(error, onRetry)
    } else if (state.nextOffset != null) item(key = "video-page-${state.nextOffset}", span = { androidx.compose.foundation.lazy.grid.GridItemSpan(maxLineSpan) }) {
        LaunchedEffect(state.nextOffset) { onLoadMore() }
        VideoPagingSpinner()
    }
}

private fun androidx.compose.foundation.lazy.LazyListScope.videoPaging(state: VideoState, onLoadMore: () -> Unit, onRetry: () -> Unit) {
    val error = state.pagingError
    if (error != null) item(key = "video-page-error") { VideoPagingError(error, onRetry) }
    else if (state.nextOffset != null) item(key = "video-page-${state.nextOffset}") {
        LaunchedEffect(state.nextOffset) { onLoadMore() }
        VideoPagingSpinner()
    }
}

@Composable
private fun VideoPagingSpinner() = AccessibleLoadingIndicator("Loading more videos", Modifier.fillMaxWidth().height(72.dp), size = 26.dp)

@Composable
private fun VideoPagingError(message: String, onRetry: () -> Unit) {
    Column(Modifier.fillMaxWidth().padding(16.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(message, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Button(onClick = onRetry) { Text("Retry") }
    }
}

@Composable
private fun VideoItemActionSurface(item: VideoItem, onPlay: (VideoItem) -> Unit, onSetWatched: (VideoItem, Boolean) -> Unit, openDetails: Boolean, content: @Composable (Modifier) -> Unit) {
    var menuExpanded by remember(item.libraryId, item.id) { mutableStateOf(false) }
    Box {
        content(
            Modifier.combinedClickable(
                onClickLabel = if (item.playable || openDetails) "${if (openDetails) "Open" else "Play"} ${item.displayTitle}" else null,
                onLongClickLabel = "Video options",
                role = Role.Button,
                onClick = { if (openDetails || item.playable) onPlay(item) },
                onLongClick = { menuExpanded = true },
            ).semantics {
                customActions = listOf(CustomAccessibilityAction(if (item.finished) "Mark as unwatched" else "Mark as watched") {
                    onSetWatched(item, !item.finished)
                    true
                })
            },
        )
        DropdownMenu(expanded = menuExpanded, onDismissRequest = { menuExpanded = false }) {
            DropdownMenuItem(
                text = { Text(if (item.finished) "Mark as unwatched" else "Mark as watched") },
                onClick = { menuExpanded = false; onSetWatched(item, !item.finished) },
            )
        }
    }
}
