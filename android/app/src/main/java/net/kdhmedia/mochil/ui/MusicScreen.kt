package net.kdhmedia.mochil.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ViewList
import androidx.compose.material.icons.filled.GridView
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.FilterList
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.DownloadDone
import androidx.compose.material.icons.filled.CloudDownload
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Shuffle
import androidx.compose.material3.*
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.key
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.*

@Composable
fun MusicScreen(
    state: MusicState,
    trackArtworkUrl: (AudioTrack) -> String,
    groupArtworkUrl: (AudioGroup) -> String,
    authorizationHeader: String?,
    onPlay: (AudioTrack) -> Unit,
    onOpenGroup: (AudioGroup) -> Unit,
    onOpenAlbum: (AudioGroup) -> Unit,
    onCloseGroup: () -> Unit,
    onCloseArtist: () -> Unit,
    onGrouping: (MusicGrouping) -> Unit,
    onLayout: (MusicLayout) -> Unit,
    onLoadMore: () -> Unit,
    onRetry: () -> Unit,
    onSearch: (String) -> Unit,
    onDownloadedOnly: (Boolean) -> Unit,
    onFavoritesOnly: (Boolean) -> Unit,
    onShowAllTracks: () -> Unit,
    onToggleDownload: (AudioTrack) -> Unit,
    onToggleFavorite: (AudioTrack) -> Unit,
    onPlayNext: (AudioTrack) -> Unit,
    onAddToQueue: (AudioTrack) -> Unit,
    onDownloadGroup: () -> Unit,
    playlists: MusicPlaylistBindings,
    onQueueGroup: (List<AudioTrack>, Boolean) -> Unit,
    onPlayGroup: (List<AudioTrack>, Boolean) -> Unit,
    onPlayArtist: (Boolean) -> Unit,
) {
    // Saved across a rotation: the open search panel is a place the viewer put themselves in, and plain `remember`
    // collapsed it back to the plain shelf every time the screen was recreated.
    var searchExpanded by rememberSaveable { mutableStateOf(false) }
    BoxWithConstraints(Modifier.fillMaxSize()) {
        val inlineGroup = state.usesInlineTrackGroup()
        val hasSelection = state.selectedArtist != null || state.selectedGroup != null && !inlineGroup
        val twoPane = usesMediaTwoPane(maxWidth.value.toInt(), hasSelection)
        if (twoPane) {
            BackHandler(onBack = if (state.selectedGroup != null) onCloseGroup else onCloseArtist)
            Row(Modifier.fillMaxSize()) {
                Box(Modifier.weight(.43f)) { MusicBrowse(state.copy(loading = false, error = null), trackArtworkUrl, groupArtworkUrl, authorizationHeader, onPlay, onOpenGroup, onCloseGroup, onGrouping, onLayout, onLoadMore, onRetry, onSearch, onDownloadedOnly, onFavoritesOnly, onShowAllTracks, onToggleDownload, onToggleFavorite, onPlayNext, onAddToQueue, playlists, searchExpanded) { searchExpanded = !searchExpanded } }
                VerticalDivider()
                Box(Modifier.weight(.57f)) {
                    if (state.selectedGroup != null) GroupTracks(state, groupArtworkUrl, authorizationHeader, onPlay, onCloseGroup, onRetry, onDownloadedOnly, onFavoritesOnly, onShowAllTracks, onToggleDownload, onToggleFavorite, onPlayNext, onAddToQueue, playlists, onDownloadGroup, onQueueGroup, onPlayGroup)
                    else ArtistAlbums(state, groupArtworkUrl, authorizationHeader, onOpenAlbum, onCloseArtist, onRetry, onPlayArtist)
                }
            }
        } else if (state.selectedGroup != null && !inlineGroup) {
            BackHandler(onBack = onCloseGroup)
            GroupTracks(state, groupArtworkUrl, authorizationHeader, onPlay, onCloseGroup, onRetry, onDownloadedOnly, onFavoritesOnly, onShowAllTracks, onToggleDownload, onToggleFavorite, onPlayNext, onAddToQueue, playlists, onDownloadGroup, onQueueGroup, onPlayGroup)
        } else if (state.selectedArtist != null) {
            BackHandler(onBack = onCloseArtist)
            ArtistAlbums(state, groupArtworkUrl, authorizationHeader, onOpenAlbum, onCloseArtist, onRetry, onPlayArtist)
        } else {
            MusicBrowse(state, trackArtworkUrl, groupArtworkUrl, authorizationHeader, onPlay, onOpenGroup, onCloseGroup, onGrouping, onLayout, onLoadMore, onRetry, onSearch, onDownloadedOnly, onFavoritesOnly, onShowAllTracks, onToggleDownload, onToggleFavorite, onPlayNext, onAddToQueue, playlists, searchExpanded) { searchExpanded = !searchExpanded }
        }
    }
}

@Composable
private fun MusicBrowse(
    state: MusicState,
    trackArtworkUrl: (AudioTrack) -> String,
    groupArtworkUrl: (AudioGroup) -> String,
    authorizationHeader: String?,
    onPlay: (AudioTrack) -> Unit,
    onOpenGroup: (AudioGroup) -> Unit,
    onCloseGroup: () -> Unit,
    onGrouping: (MusicGrouping) -> Unit,
    onLayout: (MusicLayout) -> Unit,
    onLoadMore: () -> Unit,
    onRetry: () -> Unit,
    onSearch: (String) -> Unit,
    onDownloadedOnly: (Boolean) -> Unit,
    onFavoritesOnly: (Boolean) -> Unit,
    onShowAllTracks: () -> Unit,
    onToggleDownload: (AudioTrack) -> Unit,
    onToggleFavorite: (AudioTrack) -> Unit,
    onPlayNext: (AudioTrack) -> Unit,
    onAddToQueue: (AudioTrack) -> Unit,
    playlists: MusicPlaylistBindings,
    searchExpanded: Boolean,
    onToggleSearch: () -> Unit,
) {
    Column(Modifier.fillMaxSize()) {
        MusicViewControls(state, onGrouping, onLayout, onDownloadedOnly, onFavoritesOnly, onShowAllTracks, searchExpanded || state.query.isNotBlank(), onToggleSearch)
        MotionVisibility(searchExpanded) {
            MusicSearch(state.query, state.libraryKind == "audiobooks", { query ->
                if (state.grouping == MusicGrouping.Discover || state.grouping == MusicGrouping.Playlists) onGrouping(MusicGrouping.Tracks)
                onSearch(query)
            }) { if (searchExpanded) onToggleSearch() }
        }
        state.downloadMessage?.let {
            Text(it, Modifier.padding(horizontal = 20.dp, vertical = 6.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
        }
        state.actionMessage?.let { MusicActionNotice(it) }
        if (state.offlineOnly) Row(Modifier.fillMaxWidth().padding(start = 20.dp, end = 12.dp), verticalAlignment = Alignment.CenterVertically) {
            Text("Offline · downloaded audio only", Modifier.weight(1f), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)
            TextButton(onClick = onRetry) { Text("Retry") }
        }
        when {
            state.usesInlineTrackGroup() && state.groups.isNotEmpty() -> MusicExpandableGroupList(state, groupArtworkUrl, authorizationHeader, onOpenGroup, onCloseGroup, onPlay, onLoadMore, onRetry, onToggleDownload, onToggleFavorite, onPlayNext, onAddToQueue, playlists)
            // Returning to the app re-reads favourites and downloads behind whatever is on screen, so a refresh can
            // arrive with the shelf already populated. Only a shelf with nothing on it is a load worth the whole
            // screen, the same rule the video shelf uses; otherwise the viewer keeps the rows they were reading.
            state.loading && musicShelfIsEmpty(state) -> AccessibleLoadingIndicator(if (state.libraryKind == "audiobooks") "Loading audiobooks" else "Loading music", Modifier.fillMaxSize())
            state.error != null && state.groups.isEmpty() && state.tracks.isEmpty() -> CenteredMusicStatus {
                Text(state.error, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Button(onClick = onRetry) { Text("Try again") }
            }
            state.grouping == MusicGrouping.Playlists -> MusicPlaylistLibrary(playlists, trackArtworkUrl, authorizationHeader)
            state.grouping == MusicGrouping.Discover -> MusicDiscovery(
                state, groupArtworkUrl, trackArtworkUrl, authorizationHeader,
                onOpenGroup = { grouping, group -> onGrouping(grouping); onOpenGroup(group) },
                onPlay = onPlay,
                onSeeAll = onGrouping,
                onSeeFavorites = { onGrouping(MusicGrouping.Favorites) },
                onSeeDownloads = { onGrouping(MusicGrouping.Tracks); onDownloadedOnly(true) },
            )
            state.grouping in setOf(MusicGrouping.Continue, MusicGrouping.Recent, MusicGrouping.Tracks, MusicGrouping.Favorites, MusicGrouping.RecentlyPlayed) -> TrackList(state, trackArtworkUrl, authorizationHeader, onPlay, onLoadMore, onRetry, onToggleDownload, onToggleFavorite, onPlayNext, onAddToQueue, playlists)
            state.groups.isEmpty() -> CenteredMusicStatus { Text(if (state.libraryKind == "audiobooks") "No audiobook groups were found." else "No grouped music metadata was found.", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            state.layout == MusicLayout.Covers -> GroupCoverGrid(state, groupArtworkUrl, authorizationHeader, onOpenGroup, onLoadMore, onRetry)
            else -> GroupList(state, groupArtworkUrl, authorizationHeader, onOpenGroup, onLoadMore, onRetry)
	    }
	}
}

@Composable
private fun MusicViewControls(state: MusicState, onGrouping: (MusicGrouping) -> Unit, onLayout: (MusicLayout) -> Unit, onDownloadedOnly: (Boolean) -> Unit, onFavoritesOnly: (Boolean) -> Unit, onShowAllTracks: () -> Unit, searchActive: Boolean, onToggleSearch: () -> Unit) {
    val groupings = if (state.libraryKind == "audiobooks") {
        listOf(MusicGrouping.Continue, MusicGrouping.Recent, MusicGrouping.Series, MusicGrouping.Authors, MusicGrouping.Tracks)
    } else {
        listOf(MusicGrouping.Discover, MusicGrouping.RecentlyPlayed, MusicGrouping.Albums, MusicGrouping.Artists, MusicGrouping.Favorites, MusicGrouping.Playlists, MusicGrouping.Tracks)
    }
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically,
    ) {
        BrowseModeMenu(state.grouping, groupings.map { BrowseModeOption(it, musicGroupingLabel(it, state.libraryKind)) }, if (state.libraryKind == "audiobooks") "Audiobook" else "Music", onGrouping)
        Spacer(Modifier.weight(1f))
        IconButton(onClick = onToggleSearch) {
            Icon(
                Icons.Default.Search,
                contentDescription = if (state.libraryKind == "audiobooks") "Search audiobooks" else "Search music",
                tint = if (searchActive) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        if (state.grouping == MusicGrouping.Tracks) {
            MusicFilterButton(state, onDownloadedOnly, onFavoritesOnly, onShowAllTracks)
        }
        if (state.grouping != MusicGrouping.Tracks && state.grouping != MusicGrouping.Discover) {
            IconButton(onClick = { onLayout(if (state.layout == MusicLayout.Covers) MusicLayout.List else MusicLayout.Covers) }) {
                Icon(
                    if (state.layout == MusicLayout.Covers) Icons.AutoMirrored.Filled.ViewList else Icons.Default.GridView,
                    contentDescription = if (state.layout == MusicLayout.Covers) "Use list view" else "Use cover view",
                )
            }
        }
    }
}

/**
 * True when there is nothing on the music shelf to keep on screen. A load with content behind it refreshes in place;
 * only an empty shelf gets the full-screen spinner, so a return to the app does not blank rows the viewer was reading.
 */
internal fun musicShelfIsEmpty(state: MusicState): Boolean =
    state.groups.isEmpty() && state.tracks.isEmpty() && state.offlineTracks.isEmpty() &&
        state.favoriteTracks.isEmpty() && state.discoveryAlbums.isEmpty() && state.discoveryArtists.isEmpty() &&
        state.discoveryRecent.isEmpty() && state.discoveryPlayed.isEmpty()

internal fun musicGroupingLabel(grouping: MusicGrouping, libraryKind: String) = when {
    grouping == MusicGrouping.Discover -> "Discover"
    grouping == MusicGrouping.Continue -> "Continue"
    grouping == MusicGrouping.Recent -> "Recent"
    grouping == MusicGrouping.RecentlyPlayed -> "Recently played"
    grouping == MusicGrouping.Tracks && libraryKind == "audiobooks" -> "All audiobooks"
    grouping == MusicGrouping.Tracks -> "Tracks"
    grouping == MusicGrouping.Albums -> "Albums"
    grouping == MusicGrouping.Artists -> "Artists"
    grouping == MusicGrouping.Favorites -> "Favorites"
    grouping == MusicGrouping.Playlists -> "Playlists"
    grouping == MusicGrouping.Series -> "Series"
    grouping == MusicGrouping.Authors -> "Authors"
    else -> grouping.name
}

@Composable
private fun MusicSearch(query: String, audiobooks: Boolean, onSearch: (String) -> Unit, onDone: () -> Unit) {
    var draft by remember(query) { mutableStateOf(query) }
    val focusManager = LocalFocusManager.current
    OutlinedTextField(
        value = draft,
        onValueChange = { draft = it },
        modifier = Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 4.dp),
        singleLine = true,
        label = { Text(if (audiobooks) "Search audiobooks" else "Search music") },
        leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
        trailingIcon = if (draft.isNotEmpty()) {
            { IconButton(onClick = { draft = ""; onSearch("") }) { Icon(Icons.Default.Close, contentDescription = "Clear search") } }
        } else null,
        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
        keyboardActions = KeyboardActions(onSearch = {
            onSearch(draft)
            focusManager.clearFocus()
            onDone()
        }),
    )
}

@Composable
private fun TrackList(state: MusicState, artworkUrl: (AudioTrack) -> String, header: String?, onPlay: (AudioTrack) -> Unit, onMore: () -> Unit, onRetry: () -> Unit, onToggleDownload: (AudioTrack) -> Unit, onToggleFavorite: (AudioTrack) -> Unit, onPlayNext: (AudioTrack) -> Unit, onAddToQueue: (AudioTrack) -> Unit, playlists: MusicPlaylistBindings) {
    val visibleTracks = rememberVisibleTracks(state)
    if (visibleTracks.isEmpty() && state.nextCursor == null) return CenteredMusicStatus {
        Text(
            when {
                state.grouping == MusicGrouping.RecentlyPlayed -> "Music you play will appear here."
                state.grouping == MusicGrouping.Favorites || state.favoritesOnly -> "Favorite a track and it will appear here."
                state.downloadedOnly -> "No downloaded audio on this shelf."
                else -> "This audio shelf is empty."
            },
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
    key(state.grouping, state.layout) {
        if (state.usesTrackCoverGrid()) {
            MusicTrackCoverGrid(state, visibleTracks, artworkUrl, header, onPlay, onToggleDownload, onToggleFavorite, onPlayNext, onAddToQueue, playlists)
        } else {
            val playlistNames = playlists.state.playlists.map { it.name }
            LazyColumn(Modifier.fillMaxSize()) {
                items(visibleTracks, key = AudioTrack::identityKey) { track -> MusicTrackRow(track, artworkUrl(track), header, downloaded = track.key() in state.downloadedIds, favorite = track.key() in state.favoriteIds, downloading = state.downloadingId == track.key(), onPlay = { onPlay(track) }, onToggleDownload = { onToggleDownload(track) }, onToggleFavorite = { onToggleFavorite(track) }, onPlayNext = { onPlayNext(track) }, onAddToQueue = { onAddToQueue(track) }, playlistNames = playlistNames, onAddToPlaylist = { playlists.add(track, it) }) }
                audioPaging(state, onMore, onRetry)
            }
        }
    }
}

@Composable
internal fun rememberVisibleTracks(state: MusicState): List<AudioTrack> = remember(
    state.grouping,
    state.libraryId,
    state.downloadedOnly,
    state.favoritesOnly,
    state.selectedGroup,
    state.selectedArtist,
    state.tracks,
    state.offlineTracks,
    state.favoriteTracks,
    state.discoveryPlayed,
) { state.visibleTracks() }

@Composable
private fun MusicFilterButton(state: MusicState, onDownloadedOnly: (Boolean) -> Unit, onFavoritesOnly: (Boolean) -> Unit, onShowAll: () -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    Box {
        IconButton(onClick = { expanded = true }) {
            Icon(Icons.Default.FilterList, contentDescription = "Filter tracks", tint = if (state.downloadedOnly || state.favoritesOnly) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
        }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            DropdownMenuItem(text = { Text("All tracks") }, onClick = { expanded = false; onShowAll() })
            DropdownMenuItem(text = { Text("Favorites") }, leadingIcon = { Icon(Icons.Default.Favorite, contentDescription = null) }, onClick = { expanded = false; onFavoritesOnly(true) })
            DropdownMenuItem(text = { Text("Downloaded") }, leadingIcon = { Icon(Icons.Default.DownloadDone, contentDescription = null) }, onClick = { expanded = false; onDownloadedOnly(true) })
        }
    }
}

private fun AudioTrack.key() = identityKey
