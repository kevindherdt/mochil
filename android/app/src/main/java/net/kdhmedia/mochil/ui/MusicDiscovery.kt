package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.AudioGroup
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicState
import net.kdhmedia.mochil.audio.stableKey

@Composable
internal fun MusicDiscovery(
    state: MusicState,
    groupArtworkUrl: (AudioGroup) -> String,
    trackArtworkUrl: (AudioTrack) -> String,
    authorizationHeader: String?,
    onOpenGroup: (MusicGrouping, AudioGroup) -> Unit,
    onPlay: (AudioTrack) -> Unit,
    onSeeAll: (MusicGrouping) -> Unit,
    onSeeFavorites: () -> Unit,
    onSeeDownloads: () -> Unit,
) {
    val favorites = remember(state.favoriteTracks, state.libraryId) {
        state.favoriteTracks.asSequence().filter { it.libraryId == state.libraryId }.take(DISCOVERY_VISIBLE_TRACKS).toList()
    }
    val downloads = remember(state.offlineTracks, state.libraryId) {
        state.offlineTracks.asSequence().filter { it.libraryId == state.libraryId }.take(DISCOVERY_VISIBLE_TRACKS).toList()
    }
    val empty = state.discoveryPlayed.isEmpty() && state.discoveryRecent.isEmpty() &&
        favorites.isEmpty() && state.discoveryAlbums.isEmpty() && state.discoveryArtists.isEmpty() && downloads.isEmpty()
    LazyColumn(
        Modifier.fillMaxSize(),
        contentPadding = PaddingValues(bottom = 24.dp),
        verticalArrangement = Arrangement.spacedBy(22.dp),
    ) {
        if (empty) item {
            // Discover is the default grouping, so an empty shelf must explain itself instead of showing blank space.
            CenteredMusicStatus {
                Text(
                    if (state.libraryKind == "audiobooks") "Nothing here yet. Play an audiobook, or open All to browse the shelf."
                    else "Nothing here yet. Play something, or open All tracks to browse.",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        if (state.discoveryPlayed.isNotEmpty()) item {
            DiscoveryTrackSection("Recently played", state.discoveryPlayed, trackArtworkUrl, authorizationHeader, onPlay) { onSeeAll(MusicGrouping.RecentlyPlayed) }
        }
        if (state.discoveryRecent.isNotEmpty()) item {
            DiscoveryTrackSection("Recently added", state.discoveryRecent, trackArtworkUrl, authorizationHeader, onPlay) { onSeeAll(MusicGrouping.Tracks) }
        }
        if (favorites.isNotEmpty()) item {
            DiscoveryTrackSection("Favorites", favorites, trackArtworkUrl, authorizationHeader, onPlay, onSeeFavorites)
        }
        if (state.discoveryAlbums.isNotEmpty()) item {
            DiscoveryGroupSection("Albums", state.discoveryAlbums, groupArtworkUrl, authorizationHeader, { onOpenGroup(MusicGrouping.Albums, it) }) { onSeeAll(MusicGrouping.Albums) }
        }
        if (state.discoveryArtists.isNotEmpty()) item {
            DiscoveryGroupSection("Artists", state.discoveryArtists, groupArtworkUrl, authorizationHeader, { onOpenGroup(MusicGrouping.Artists, it) }) { onSeeAll(MusicGrouping.Artists) }
        }
        if (downloads.isNotEmpty()) item {
            DiscoveryTrackSection("Available offline", downloads, trackArtworkUrl, authorizationHeader, onPlay, onSeeDownloads)
        }
    }
}

@Composable
private fun DiscoveryGroupSection(
    title: String,
    groups: List<AudioGroup>,
    artworkUrl: (AudioGroup) -> String,
    header: String?,
    onOpen: (AudioGroup) -> Unit,
    onSeeAll: () -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        DiscoveryHeading(title, onSeeAll)
        LazyRow(contentPadding = PaddingValues(horizontal = 20.dp), horizontalArrangement = Arrangement.spacedBy(14.dp)) {
            items(groups, key = AudioGroup::stableKey) { group ->
                Column(
                    Modifier.width(142.dp).clickable(onClickLabel = "Open ${group.name}", role = Role.Button) { onOpen(group) },
                    verticalArrangement = Arrangement.spacedBy(7.dp),
                ) {
                    AuthenticatedArtwork(artworkUrl(group), header, group.name, Modifier.fillMaxWidth().aspectRatio(1f))
                    Text(group.name, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    Text(group.subtitle.ifBlank { "${group.count} tracks" }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
            }
        }
    }
}

@Composable
private fun DiscoveryTrackSection(
    title: String,
    tracks: List<AudioTrack>,
    artworkUrl: (AudioTrack) -> String,
    header: String?,
    onPlay: (AudioTrack) -> Unit,
    onSeeAll: () -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        DiscoveryHeading(title, onSeeAll)
        tracks.take(DISCOVERY_VISIBLE_TRACKS).forEach { track ->
            Row(
                Modifier.fillMaxWidth().clickable(onClickLabel = "Play ${track.title}", role = Role.Button) { onPlay(track) }.padding(horizontal = 20.dp, vertical = 5.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                AuthenticatedArtwork(artworkUrl(track), header, track.title, Modifier.size(46.dp))
                Column(Modifier.weight(1f)) {
                    Text(track.title, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    Text(track.artist.ifBlank { "Unknown artist" }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
                if (track.durationSeconds > 0) Text(formatTrackDuration(track.durationSeconds), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

@Composable
private fun DiscoveryHeading(title: String, onSeeAll: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(start = 20.dp, end = 8.dp), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(title, style = MaterialTheme.typography.titleLarge, modifier = Modifier.padding(top = 10.dp))
        TextButton(onClick = onSeeAll) { Text("See all") }
    }
}

internal const val DISCOVERY_VISIBLE_TRACKS = 5
