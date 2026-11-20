package net.kdhmedia.mochil.ui

import androidx.compose.foundation.focusGroup
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Tune
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.layout
import androidx.compose.ui.zIndex
import androidx.compose.foundation.layout.Spacer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRestorer
import androidx.compose.ui.focus.focusRequester
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import net.kdhmedia.mochil.audio.MusicState
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicLayout
import net.kdhmedia.mochil.audio.stableKey
import net.kdhmedia.mochil.audio.supportsMusicBucketNavigation
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.podcast.PodcastState
import net.kdhmedia.mochil.podcast.PodcastEpisodeSort
import net.kdhmedia.mochil.podcast.PodcastPlayedFilter
import net.kdhmedia.mochil.video.VideoState

@Composable
internal fun TvLibraryScreen(
    library: LibraryState,
    music: MusicState,
    video: VideoState,
    podcasts: PodcastState,
    authorizationHeader: String?,
    artwork: TvLibraryArtwork,
    actions: TvLibraryActions,
    query: String,
    gridPositions: TvGridPositionStore,
) {
    var searchOpen by remember(library.selected?.id) { mutableStateOf(false) }
    var searchDraft by remember(library.selected?.id) { mutableStateOf(query) }
    val focusedDetail = remember(library.selected?.id) { TvFocusDetailState() }
    val searchFocus = remember { FocusRequester() }
    val searchButtonFocus = remember { FocusRequester() }
    val contentFocus = remember { FocusRequester() }
    var restoreSearchButtonFocus by remember(library.selected?.id) { mutableStateOf(false) }
    val atBrowseRoot = library.selectedCollection == null && library.selectedSeries == null && library.selectedAuthor == null && music.selectedGroup == null && music.selectedArtist == null && video.selectedShow == null && podcasts.selectedShow == null
    val browseModes = tvBrowseModes(library.selected?.kind.orEmpty(), library.grouping, music.grouping, atBrowseRoot)
    val contextualModes = (video.selectedShow?.let { tvSeasonModes(it.seasons, video.selectedSeason) }
        ?: podcasts.selectedShow?.let { tvPodcastEpisodeModes(podcasts.episodeSort, podcasts.playedFilter) }
        ?: browseModes) +
        if (library.selected?.kind in setOf("music", "audiobooks")) MusicLayout.entries.map { TvBrowseMode("layout:${it.name}", "${it.name} view", music.layout == it) } else emptyList()
    val alphabetBuckets = when {
        !atBrowseRoot || query.isNotBlank() -> emptyList()
        library.selected?.kind == "music" -> if (music.supportsMusicBucketNavigation()) music.navigation.filter { it.offset >= 0 }.map { it.bucket } else emptyList()
        library.selected?.isPublicationLibraryForTv() == true && library.grouping != PublicationGrouping.Recent -> listOf("#") + ('A'..'Z').map(Char::toString)
        else -> emptyList()
    }
    val selectedBucket = if (library.selected?.kind == "music") music.bucket else library.bucket
    val visibleBucket = selectedBucket.ifBlank { focusedDetail.value?.navigationBucket.orEmpty() }
    val hierarchyKey = tvHierarchyKey(
        library.selectedCollection?.id,
        library.selectedSeries?.name,
        library.selectedAuthor?.name,
        music.selectedArtist?.stableKey(),
        music.selectedGroup?.stableKey(),
        video.selectedShow?.name,
        video.selectedSeason?.toString(),
        podcasts.selectedShow?.id,
    )
    val contentRouteKey = tvContentRouteKey(library.selected?.id, hierarchyKey)
    val positionKey = TvGridRouteKey(
        libraryId = library.selected?.id.orEmpty(),
        browseMode = "${library.grouping.name}:${music.grouping.name}",
        layout = music.layout.name,
        hierarchy = hierarchyKey,
        sort = podcasts.episodeSort.name,
        filter = podcasts.playedFilter.name,
        query = query,
        bucket = selectedBucket,
    ).stableKey()
    val visibleMusicTracks = rememberVisibleTracks(music)
    val rememberedFocusKey = gridPositions.focusedItem(positionKey)
    // Focus recovery needs one membership answer, not an allocated identity
    // set containing every loaded card.
    val rememberedFocusVisible = remember(
        library.selected?.kind, library.grouping, library.books, library.seriesBooks, library.series, library.authors,
        music.grouping, music.selectedGroup, music.selectedArtist, music.artistAlbums, music.groups, music.tracks, music.discoveryPlayed, music.discoveryRecent, music.favoriteTracks,
        video.selectedShow, video.shows, video.items, podcasts.selectedShow, podcasts.shows, podcasts.episodes,
        visibleMusicTracks, rememberedFocusKey,
    ) { tvHasVisibleFocusKey(rememberedFocusKey, library, music, video, podcasts, visibleMusicTracks) }
    val activeError = when (library.selected?.kind) {
        "music", "audiobooks" -> music.error
        "movies", "tv", "anime" -> video.error
        "podcasts" -> podcasts.error
        else -> if (tvPublicationInsideGroup(library)) library.groupError else library.error
    }
    val activeItemsEmpty = when (library.selected?.kind) {
        "music", "audiobooks" -> tvAudioItemsEmpty(music, visibleMusicTracks)
        "movies", "tv", "anime" -> if (video.selectedShow == null && video.shows.isNotEmpty()) false else video.items.isEmpty()
        "podcasts" -> if (podcasts.selectedShow == null) podcasts.shows.isEmpty() else podcasts.episodes.isEmpty()
        else -> when {
            tvPublicationInsideGroup(library) -> library.seriesBooks.isEmpty()
            library.grouping == PublicationGrouping.Authors -> library.authors.isEmpty()
            library.grouping == PublicationGrouping.Series -> library.series.isEmpty()
            else -> library.books.isEmpty()
        }
    }
    LaunchedEffect(searchOpen, restoreSearchButtonFocus) {
        when {
            searchOpen -> searchFocus.requestFocus()
            restoreSearchButtonFocus -> {
                searchButtonFocus.requestFocus()
                restoreSearchButtonFocus = false
            }
        }
    }
    LaunchedEffect(positionKey, rememberedFocusVisible) {
        if (rememberedFocusKey != null && !rememberedFocusVisible) focusedDetail.clear()
    }
    LaunchedEffect(positionKey) { focusedDetail.clear() }
    // Once a library or a series has loaded, put focus on its first card (or the one that was focused last time),
    // unless the person is typing a search.
    val contentReady = library.selected != null && activeError == null && !activeItemsEmpty && !tvLibraryLoading(library, music, video, podcasts)
    var autoFocused by remember(contentRouteKey) { mutableStateOf(false) }
    LaunchedEffect(contentRouteKey, contentReady, searchOpen) {
        if (autoFocused || !contentReady || searchOpen) return@LaunchedEffect
        repeat(4) {
            if (runCatching { contentFocus.requestFocus() }.getOrDefault(false)) { autoFocused = true; return@LaunchedEffect }
            kotlinx.coroutines.delay(80)
        }
        autoFocused = true
    }
    fun submitSearch() {
        searchOpen = false
        restoreSearchButtonFocus = true
        actions.navigation.search(searchDraft.trim())
    }
    fun selectMode(mode: TvBrowseMode) {
        when {
            mode.id.startsWith("season:") -> mode.id.substringAfter(':').toIntOrNull()?.let(actions.video.selectSeason)
            mode.id.startsWith("layout:") -> runCatching { MusicLayout.valueOf(mode.id.substringAfter(':')) }.getOrNull()?.let(actions.audio.setLayout)
            mode.id.startsWith("podcast-sort:") -> runCatching { PodcastEpisodeSort.valueOf(mode.id.substringAfter(':')) }.getOrNull()?.let(actions.podcasts.setSort)
            mode.id.startsWith("podcast-filter:") -> runCatching { PodcastPlayedFilter.valueOf(mode.id.substringAfter(':')) }.getOrNull()?.let(actions.podcasts.setFilter)
            library.selected?.kind in setOf("music", "audiobooks") -> runCatching { MusicGrouping.valueOf(mode.id) }.getOrNull()?.let(actions.audio.setGrouping)
            else -> runCatching { PublicationGrouping.valueOf(mode.id) }.getOrNull()?.let(actions.publications.setGrouping)
        }
    }
    // Video shelves open with their spotlight art behind the top bar, so the title is dropped and the buttons float over the art.
    val orderedLibraries = library.libraries.inTvOrder()
    val artBehindHeader = library.selected?.kind in setOf("movies", "tv", "anime")
    Column(Modifier.fillMaxSize().padding(top = if (artBehindHeader) 0.dp else 12.dp).onPreviewKeyEvent { event ->
        if (event.type != KeyEventType.KeyDown || event.nativeKeyEvent.repeatCount != 0) return@onPreviewKeyEvent false
        val direction = tvLibraryDirection(event.nativeKeyEvent.keyCode) ?: return@onPreviewKeyEvent false
        val current = orderedLibraries.indexOfFirst { it.id == library.selected?.id }
        val next = adjacentLibraryIndex(current, orderedLibraries.size, direction)
        if (next < 0) return@onPreviewKeyEvent false
        actions.navigation.selectLibrary(orderedLibraries[next])
        true
    }) {
        Row(
            Modifier.fillMaxWidth().padding(start = 12.dp, end = 44.dp)
                .then(if (artBehindHeader) Modifier.layout { measurable, constraints -> measurable.measure(constraints).let { placeable -> layout(placeable.width, 0) { placeable.place(0, 12.dp.roundToPx()) } } }.zIndex(1f) else Modifier),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (artBehindHeader) Spacer(Modifier.weight(1f)) else TvLibraryHeader(
                libraryName = library.selected?.name ?: "Libraries",
                browseContext = tvBrowseContext(library, music, video, podcasts),
                focusedDetail = focusedDetail,
                modifier = Modifier.weight(1f).padding(start = 28.dp),
            )
            if (contextualModes.isNotEmpty() && !searchOpen) {
                if (tvModesFitAsTabs(contextualModes)) TvBrowseModeTabs(contextualModes, ::selectMode)
                else TvBrowseModePicker(contextualModes, ::selectMode, compact = true)
            }
            if (searchOpen) {
                OutlinedTextField(
                    value = searchDraft,
                    onValueChange = { searchDraft = it.take(160) },
                    modifier = Modifier.width(430.dp).focusRequester(searchFocus).tvFocusBorder(),
                    placeholder = { Text("Search this library") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                    keyboardActions = KeyboardActions(onSearch = { submitSearch() }),
                    trailingIcon = { TvIconButton(Icons.Default.Search, "Run search", { submitSearch() }) },
                )
                TvIconButton(Icons.Default.Close, "Close search", {
                    searchOpen = false
                    restoreSearchButtonFocus = true
                    searchDraft = ""
                    actions.navigation.search("")
                })
            } else {
                TvIconButton(
                    Icons.Default.Search,
                    tvSearchButtonLabel(query),
                    { searchOpen = true },
                    Modifier.focusRequester(searchButtonFocus),
                    active = query.isNotBlank(),
                )
            }
        }
        if (alphabetBuckets.isNotEmpty()) TvAlphabetGrid(alphabetBuckets, visibleBucket) { bucket ->
            val nextBucket = toggledTvBucket(selectedBucket, bucket)
            if (library.selected?.kind == "music") actions.audio.selectBucket(nextBucket) else actions.publications.selectBucket(nextBucket)
        }
        Row(Modifier.fillMaxWidth().weight(1f)) {
        androidx.compose.runtime.CompositionLocalProvider(LocalTvContentFocus provides contentFocus) { Box(Modifier.weight(1f)) { when {
            library.selected == null -> AccessibleLoadingIndicator("Loading libraries", Modifier.fillMaxSize())
            library.selected.isPublicationLibraryForTv() && (library.initialLoading || library.groupLoading) -> AccessibleLoadingIndicator("Loading ${library.selected.name}", Modifier.fillMaxSize())
            library.selected.kind in setOf("movies", "tv", "anime") && video.loading -> AccessibleLoadingIndicator("Loading ${library.selected.name}", Modifier.fillMaxSize())
            library.selected.kind in setOf("music", "audiobooks") && music.loading -> AccessibleLoadingIndicator("Loading ${library.selected.name}", Modifier.fillMaxSize())
            library.selected.kind == "podcasts" && podcasts.loading -> AccessibleLoadingIndicator("Loading ${library.selected.name}", Modifier.fillMaxSize())
            activeError != null -> TvLibraryError(activeError, actions.navigation.retry)
            activeItemsEmpty -> TvEmptyLibrary(if (query.isBlank()) "Nothing is available in this view yet." else "No results for this search.")
            library.selected.isPublicationLibraryForTv() -> TvPublicationGrid(library, authorizationHeader, artwork.book, artwork.series, artwork.author, actions.publications.openSeries, actions.publications.openAuthor, actions.publications.read, actions.publications.loadMore, focusedDetail::show, positionKey, gridPositions)
            library.selected.kind in setOf("movies", "tv", "anime") -> TvVideoGrid(video, authorizationHeader, artwork.video, actions.video.openShow, actions.video.play, actions.video.setWatched, actions.video.loadMore, focusedDetail::show, positionKey, gridPositions, actions.video.selectSeason, actions.video.loadDetails)
            library.selected.kind in setOf("music", "audiobooks") -> TvAudioGrid(music, visibleMusicTracks, authorizationHeader, artwork.audioGroup, artwork.track, actions.audio.openGroup, actions.audio.play, actions.audio.playNext, actions.audio.addToQueue, actions.audio.toggleFavorite, actions.audio.loadMore, focusedDetail::show, positionKey, gridPositions)
            library.selected.kind == "podcasts" -> TvPodcastGrid(podcasts, authorizationHeader, artwork.podcast, artwork.episode, actions.podcasts.openShow, actions.podcasts.play, actions.podcasts.setPlayed, actions.podcasts.loadMore, focusedDetail::show, positionKey, gridPositions)
            else -> TvEmptyLibrary("This library type is not ready for TV browsing yet.")
        } } }
        }
    }
}

@Composable
private fun TvLibraryHeader(libraryName: String, browseContext: String, focusedDetail: TvFocusDetailState, modifier: Modifier = Modifier) {
    // Every grid publishes the focused card's title and metadata; without this the header only ever
    // named the library, so a viewer had to read a truncated one-line caption under each cover.
    val detail = focusedDetail.value
    val headline = detail?.title?.takeIf { it.isNotBlank() } ?: libraryName
    val supporting = detail?.metadata?.takeIf { it.isNotBlank() } ?: browseContext
    Row(modifier.height(62.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(18.dp)) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(1.dp)) {
            Text(headline, style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
            if (supporting.isNotBlank()) Text(supporting, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

/** A handful of plain views (Continue, Series, Authors…) read best as tabs; long lists of sorts and filters stay in the menu. */
internal fun tvModesFitAsTabs(modes: List<TvBrowseMode>): Boolean =
    modes.size in 2..6 && modes.none { it.id.startsWith("season:") || it.id.startsWith("podcast-") || it.id.startsWith("layout:") }

@Composable
private fun TvBrowseModeTabs(modes: List<TvBrowseMode>, onSelect: (TvBrowseMode) -> Unit) {
    Row(Modifier.padding(end = 10.dp).focusRestorer().focusGroup(), horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
        modes.forEach { mode -> TvPill(mode.label, selected = mode.selected, onClick = { onSelect(mode) }) }
    }
}

@Composable
private fun TvBrowseModePicker(modes: List<TvBrowseMode>, onSelect: (TvBrowseMode) -> Unit, compact: Boolean = false) {
    var expanded by remember { mutableStateOf(false) }
    val selected = modes.firstOrNull(TvBrowseMode::selected) ?: modes.first()
    val selectedFocus = remember(selected.id) { FocusRequester() }
    LaunchedEffect(expanded, selected.id) { if (expanded) selectedFocus.requestFocus() }
    Box {
        if (compact) TvIconButton(Icons.Default.Tune, "Browse mode: ${selected.label}", { expanded = true })
        else OutlinedButton(onClick = { expanded = true }, modifier = Modifier.height(48.dp).tvFocusBorder()) { Text(selected.label, maxLines = 1); androidx.compose.material3.Icon(Icons.Default.ArrowDropDown, contentDescription = null) }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            modes.forEach { mode ->
                DropdownMenuItem(
                    text = { Text(mode.label, fontWeight = if (mode.selected) FontWeight.SemiBold else FontWeight.Normal) },
                    onClick = { expanded = false; onSelect(mode) },
                    modifier = Modifier.then(if (mode.id == selected.id) Modifier.focusRequester(selectedFocus) else Modifier).tvFocusBorder(),
                )
            }
        }
    }
}

@Composable
private fun TvEmptyLibrary(message: String) = Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { Text(message, style = MaterialTheme.typography.titleLarge, color = MaterialTheme.colorScheme.onSurfaceVariant) }

@Composable
private fun TvLibraryError(message: String, onRetry: () -> Unit) {
    val retryFocus = remember { FocusRequester() }
    LaunchedEffect(message) { retryFocus.requestFocus() }
    Column(
        Modifier.fillMaxSize().semantics { liveRegion = LiveRegionMode.Polite },
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(18.dp, Alignment.CenterVertically),
    ) {
        Text(
            message,
            modifier = Modifier.fillMaxWidth(.72f).widthIn(max = 560.dp),
            style = MaterialTheme.typography.titleLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = androidx.compose.ui.text.style.TextAlign.Center,
            maxLines = 3,
            overflow = TextOverflow.Ellipsis,
        )
        Button(onClick = onRetry, modifier = Modifier.focusRequester(retryFocus).tvFocusBorder()) { Text("Try again") }
    }
}

private fun tvBrowseContext(library: LibraryState, music: MusicState, video: VideoState, podcasts: PodcastState): String = when (library.selected?.kind) {
    "books", "lightnovels", "manga", "comics", "mixed", "calibre" -> library.selectedCollection?.name ?: library.selectedSeries?.name ?: library.selectedAuthor?.name
    "movies", "tv", "anime" -> video.selectedShow?.name
    "music", "audiobooks" -> music.selectedGroup?.name
    "podcasts" -> podcasts.selectedShow?.title
    else -> null
}.orEmpty()

private fun LibrarySummary.isPublicationLibraryForTv() = kind in setOf("books", "lightnovels", "manga", "comics", "mixed", "calibre")

private fun tvLibraryLoading(library: LibraryState, music: MusicState, video: VideoState, podcasts: PodcastState): Boolean = when (library.selected?.kind) {
    "movies", "tv", "anime" -> video.loading
    "music", "audiobooks" -> music.loading
    "podcasts" -> podcasts.loading
    else -> library.initialLoading || library.groupLoading
}

internal fun toggledTvBucket(selected: String, requested: String): String = if (selected == requested) "" else requested

internal fun tvSearchButtonLabel(query: String): String =
    if (query.isBlank()) "Search this library" else "Edit search: ${query.trim().take(80)}"
