package net.kdhmedia.mochil.ui

import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.background
import androidx.compose.material3.Icon
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Headphones
import androidx.compose.material.icons.automirrored.outlined.MenuBook
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.draw.clip
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.CustomAccessibilityAction
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.customActions
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.dashboard.ContinueListening
import net.kdhmedia.mochil.dashboard.ContinueReading
import net.kdhmedia.mochil.dashboard.DashboardState
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.ui.theme.LocalEInkPalette

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    state: DashboardState,
    coverUrl: (BookSummary) -> String,
    artworkUrl: (AudioTrack) -> String,
    videoArtworkUrl: (VideoItem) -> String,
    authorizationHeader: String?,
    onRead: (BookSummary) -> Unit,
    onDismissReading: (BookSummary) -> Unit,
    onResumeHeld: (BookSummary) -> Unit,
    onListen: (AudioTrack) -> Unit,
    onDismissListening: (ContinueListening) -> Unit,
    onWatch: (VideoItem) -> Unit,
    onDismissWatching: (VideoItem) -> Unit,
    onRefresh: () -> Unit,
    libraries: List<LibrarySummary>,
) {
  // No drop-down: the dock switches places and the library screen's pills switch shelves, as on the web.
  Column(Modifier.fillMaxSize()) {
    PullToRefreshBox(isRefreshing = state.loading, onRefresh = onRefresh, modifier = Modifier.weight(1f)) {
      LazyColumn(Modifier.fillMaxSize(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        if (state.error != null) {
            item {
                Column(
                    Modifier.fillMaxWidth().padding(20.dp).semantics { liveRegion = LiveRegionMode.Polite },
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Text(state.error, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Button(onClick = onRefresh) { Text("Try again") }
                }
            }
        }
        if (state.loading && state.reading.isEmpty() && state.held.isEmpty() && state.listening.isEmpty() && state.watching.isEmpty() && state.recent.isEmpty()) {
            item { HomeSkeleton() }
        } else if (state.reading.isEmpty() && state.held.isEmpty() && state.listening.isEmpty() && state.watching.isEmpty() && state.recent.isEmpty()) {
            if (state.error == null) item { Text("Media you start will wait for you here.", Modifier.padding(20.dp), color = MaterialTheme.colorScheme.onSurfaceVariant) }
        } else {
            val kindOf = { libraryId: String -> libraries.firstOrNull { it.id == libraryId }?.kind.orEmpty() }
            item { DashboardRail("Continue reading", state.reading, { it.book.libraryId to it.book.id }) { item ->
                BookCard(item.book, kindOf(item.book.libraryId), coverUrl(item.book), authorizationHeader, onRead, {}, onDismiss = { onDismissReading(item.book) })
            } }
            item { DashboardRail("Continue listening", state.listening, { it.track.libraryId to it.track.id }) { item ->
                MusicAlbumCard(
                    title = item.track.title, subtitle = item.track.artist.ifBlank { item.track.album },
                    art = { AuthenticatedArtwork(artworkUrl(item.track), authorizationHeader, item.track.title, Modifier.fillMaxSize()) },
                    onClick = { onListen(item.track) },
                    onDismiss = { onDismissListening(item) },
                )
            } }
            item { DashboardRail("Continue watching", state.watching, { it.libraryId to it.id }) { item ->
                VideoCard(
                    art = { AuthenticatedArtwork(videoArtworkUrl(item), authorizationHeader, item.displayTitle, it) },
                    aspect = if (item.episode > 0) 16f / 9f else 2f / 3f,
                    title = item.displayTitle, caption = item.caption(),
                    formatLabel = videoFormatLabel(item.videoHeight, item.directPlay),
                    // Long press removes it from Discover, the same gesture the rows had; never a visible button.
                    modifier = Modifier.combinedClickable(
                        onClickLabel = "Open ${item.displayTitle}", role = Role.Button, onClick = { onWatch(item) },
                        onLongClickLabel = "Remove ${item.displayTitle} from Discover", onLongClick = { onDismissWatching(item) },
                    ),
                )
            } }
            item { DashboardRail("Recently added", state.recent, { it.libraryId to it.id }) { book ->
                BookCard(book, kindOf(book.libraryId), coverUrl(book), authorizationHeader, onRead, {})
            } }
            if (state.held.isNotEmpty()) {
                item { DashboardSectionTitle("On hold & dropped") }
                items(state.held, key = { "held:${it.book.libraryId}:${it.book.id}" }) { item ->
                    HeldRow(item, coverUrl(item.book), authorizationHeader, if (item.book.status == "dropped") "Dropped" else "On hold", "Resume") { onResumeHeld(item.book) }
                }
            }
        }
      }
    }
  }
}

/**
 * One rail of Discover: a heading and a sideways row of the same cards the library draws, as the web's Discover does
 * (web/src/features/discover/screens/DiscoverScreen.tsx). A rail with nothing in it is left out entirely rather than
 * shown empty; the screen's own empty state covers "nothing at all".
 */
@Composable
internal fun <T> DashboardRail(title: String, entries: List<T>, key: (T) -> Pair<String, String>, card: @Composable (T) -> Unit) {
    if (entries.isEmpty()) return
    Column(verticalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(vertical = 4.dp)) {
        DashboardSectionTitle(title)
        LazyRow(
            contentPadding = PaddingValues(horizontal = MOBILE_CONTENT_GUTTER),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(entries, key = { item -> key(item).first + "\u001f" + key(item).second }) { item ->
                // A grid cell's width, so a rail card is the same size as a card in a shelf.
                Box(Modifier.width(148.dp)) { card(item) }
            }
        }
    }
}

/** A book the reader parked, with the one action that brings it back. */
@Composable
private fun HeldRow(item: ContinueReading, coverUrl: String, authorizationHeader: String?, label: String, action: String, onAction: () -> Unit) {
    val title = dashboardReadingTitle(item.book)
    Row(Modifier.homeCard().padding(12.dp), horizontalArrangement = Arrangement.spacedBy(14.dp), verticalAlignment = Alignment.CenterVertically) {
        AuthenticatedArtwork(coverUrl, authorizationHeader, item.book.title, Modifier.size(width = 40.dp, height = 56.dp))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(5.dp)) {
            Text(title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text(label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        TextButton(onClick = onAction) { Text(action) }
    }
}

@Composable
private fun DashboardSectionTitle(title: String) {
    Text(title, Modifier.padding(start = MOBILE_CONTENT_GUTTER, end = MOBILE_CONTENT_GUTTER, top = 18.dp, bottom = 4.dp), style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
}

internal fun dashboardReadingTitle(book: BookSummary): String {
    val title = book.title.trim().ifBlank { "Untitled" }
    val series = book.series.trim()
    return if (series.isNotBlank() && title.startsWith("chapter ", ignoreCase = true)) "$series — $title" else title
}

