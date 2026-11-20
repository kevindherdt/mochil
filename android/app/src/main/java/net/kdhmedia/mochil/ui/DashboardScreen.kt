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
    onOpenSettings: () -> Unit,
    libraries: List<LibrarySummary>,
    selectedLibrary: LibrarySummary?,
    showCollections: Boolean,
    showSources: Boolean,
    onOpenCollections: () -> Unit,
    onOpenSources: () -> Unit,
    onSelectLibrary: (LibrarySummary) -> Unit,
) {
  Column(Modifier.fillMaxSize()) {
    if (!LocalExpandedMobileNavigation.current) Row(Modifier.fillMaxWidth().padding(horizontal = MOBILE_CONTENT_GUTTER)) {
        MobileLibraryRail(libraries, selectedLibrary, MobileShellDestination.Home, {}, showCollections, showSources, onOpenCollections, onOpenSources, onOpenSettings, onSelectLibrary)
    }
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
        if (state.loading && state.reading.isEmpty() && state.held.isEmpty() && state.finished.isEmpty() && state.listening.isEmpty() && state.watching.isEmpty()) {
            item { HomeSkeleton() }
        } else if (state.reading.isEmpty() && state.held.isEmpty() && state.finished.isEmpty() && state.listening.isEmpty() && state.watching.isEmpty()) {
            if (state.error == null) item { Text("Media you start will wait for you here.", Modifier.padding(20.dp), color = MaterialTheme.colorScheme.onSurfaceVariant) }
        } else {
            val kindOf = { libraryId: String -> libraries.firstOrNull { it.id == libraryId }?.kind.orEmpty() }
            val jumpBackIn = dashboardJumpBackIn(
                state.reading.map { item -> DashboardEntry("book:${item.book.libraryId}:${item.book.id}") { ReadingRow(item, kindOf(item.book.libraryId), coverUrl(item.book), authorizationHeader, { onRead(item.book) }) { onDismissReading(item.book) } } },
                state.watching.map { item -> DashboardEntry("video:${item.libraryId}:${item.id}") { WatchingRow(item, kindOf(item.libraryId), videoArtworkUrl(item), authorizationHeader, { onWatch(item) }) { onDismissWatching(item) } } },
                state.listening.map { item -> DashboardEntry("audio:${item.track.libraryId}:${item.track.id}") { ListeningRow(item, item.track.libraryKind.ifBlank { kindOf(item.track.libraryId) }, artworkUrl(item.track), authorizationHeader, { onListen(item.track) }) { onDismissListening(item) } } },
            )
            if (jumpBackIn.isNotEmpty()) {
                item { DashboardSectionTitle("Jump back in") }
                items(jumpBackIn, key = DashboardEntry::key) { entry -> entry.content() }
            }
            if (state.held.isNotEmpty()) {
                item { DashboardSectionTitle("On hold & dropped") }
                items(state.held, key = { "held:${it.book.libraryId}:${it.book.id}" }) { item ->
                    HeldRow(item, coverUrl(item.book), authorizationHeader, if (item.book.status == "dropped") "Dropped" else "On hold", "Resume") { onResumeHeld(item.book) }
                }
            }
            if (state.finished.isNotEmpty()) {
                item { DashboardSectionTitle("Finished") }
                items(state.finished, key = { "finished:${it.book.libraryId}:${it.book.id}" }) { item ->
                    HeldRow(item, coverUrl(item.book), authorizationHeader, "Read", "Restore") { onResumeHeld(item.book) }
                }
            }
        }
      }
    }
  }
}

/** One row of Jump back in, with the key that keeps it stable in the list. */
internal class DashboardEntry(val key: String, val content: @Composable () -> Unit)

/** Reading, watching and listening mixed in turn, so one kind does not push the others off the screen. Each list keeps its own order. */
internal fun <T> dashboardJumpBackIn(vararg lists: List<T>): List<T> {
    val longest = lists.maxOfOrNull { it.size } ?: 0
    return buildList { for (index in 0 until longest) lists.forEach { list -> list.getOrNull(index)?.let(::add) } }
}

/** A round icon in the library kind's own strong colour, so a book, a film and a recording can be told apart at a glance. */
@Composable
private fun MediaTypeBadge(kind: String) {
    val style = mediaKindStyle(
        kind,
        monochromeColor = if (LocalEInkPalette.current) MaterialTheme.colorScheme.onSurface else null,
    )
    val tint = style.tint(MaterialTheme.colorScheme.background)
    Box(Modifier.size(34.dp).clip(androidx.compose.foundation.shape.CircleShape).background(tint.copy(alpha = .20f)), contentAlignment = Alignment.Center) {
        Icon(style.icon, contentDescription = style.label, modifier = Modifier.size(20.dp), tint = tint)
    }
}

@Composable
private fun WatchingRow(item: VideoItem, kind: String, artworkUrl: String, authorizationHeader: String?, onClick: () -> Unit, onDismiss: () -> Unit) {
    var menuExpanded by remember { mutableStateOf(false) }
    Box {
    Row(
        Modifier.homeCard().combinedClickable(onClickLabel = "Continue watching ${item.displayTitle}", role = Role.Button, onClick = onClick, onLongClickLabel = "More options", onLongClick = { menuExpanded = true })
            .semantics { customActions = listOf(CustomAccessibilityAction("Remove from Home") { onDismiss(); true }) }.padding(12.dp),
        horizontalArrangement = Arrangement.spacedBy(14.dp), verticalAlignment = Alignment.CenterVertically,
    ) {
        AuthenticatedArtwork(artworkUrl, authorizationHeader, item.displayTitle, Modifier.size(width = 72.dp, height = 44.dp))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(5.dp)) {
            Text(item.displayTitle, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 1, overflow = TextOverflow.Ellipsis)
            val context = listOfNotNull(item.show.takeIf(String::isNotBlank), item.episode.takeIf { it > 0 }?.let { "S${item.season}E$it" }).joinToString(" · ")
            if (context.isNotBlank()) Text(context, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text("${(item.progressFraction * 100).toInt()}% watched", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        MediaTypeBadge(kind)
    }
        DropdownMenu(expanded = menuExpanded, onDismissRequest = { menuExpanded = false }) {
            DropdownMenuItem(text = { Text("Remove from Home") }, onClick = { menuExpanded = false; onDismiss() })
        }
    }
}

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
    Text(title, Modifier.padding(start = 20.dp, end = 20.dp, top = 18.dp, bottom = 4.dp), style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
}

@Composable
private fun ReadingRow(item: ContinueReading, kind: String, coverUrl: String, authorizationHeader: String?, onClick: () -> Unit, onDismiss: () -> Unit) {
    val fraction = item.fraction.takeIf(Double::isFinite)?.coerceIn(0.0, 1.0) ?: 0.0
    val title = dashboardReadingTitle(item.book)
    var menuExpanded by remember { mutableStateOf(false) }
    Box {
    Row(
        Modifier.homeCard().combinedClickable(onClickLabel = "Continue reading $title", role = Role.Button, onClick = onClick, onLongClickLabel = "More options", onLongClick = { menuExpanded = true })
            .semantics { customActions = listOf(CustomAccessibilityAction("Remove from Home") { onDismiss(); true }) }.padding(12.dp),
        horizontalArrangement = Arrangement.spacedBy(14.dp), verticalAlignment = Alignment.CenterVertically,
    ) {
        AuthenticatedArtwork(coverUrl, authorizationHeader, item.book.title, Modifier.size(width = 40.dp, height = 56.dp))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(5.dp)) {
            Text(title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text("${(fraction * 100).toInt()}% read", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            ThinProgress(progress = { fraction.toFloat() }, Modifier.fillMaxWidth().height(4.dp).clip(RoundedCornerShape(2.dp)))
            if (item.book.readAlong) ReadAlongLabel()
        }
        MediaTypeBadge(kind)
    }
        DropdownMenu(expanded = menuExpanded, onDismissRequest = { menuExpanded = false }) {
            DropdownMenuItem(text = { Text("Remove from Home") }, onClick = { menuExpanded = false; onDismiss() })
        }
    }
}

internal fun dashboardReadingTitle(book: BookSummary): String {
    val title = book.title.trim().ifBlank { "Untitled" }
    val series = book.series.trim()
    return if (series.isNotBlank() && title.startsWith("chapter ", ignoreCase = true)) "$series — $title" else title
}

@Composable
private fun ListeningRow(item: ContinueListening, kind: String, artworkUrl: String, authorizationHeader: String?, onClick: () -> Unit, onDismiss: () -> Unit) {
    val fraction = if (item.durationSeconds.isFinite() && item.durationSeconds > 0 && item.positionSeconds.isFinite()) (item.positionSeconds / item.durationSeconds).coerceIn(0.0, 1.0) else 0.0
    var menuExpanded by remember { mutableStateOf(false) }
    Box {
    Row(
        Modifier.homeCard().combinedClickable(onClickLabel = "Continue listening to ${item.track.title}", role = Role.Button, onClick = onClick, onLongClickLabel = "More options", onLongClick = { menuExpanded = true })
            .semantics { customActions = listOf(CustomAccessibilityAction("Remove from Home") { onDismiss(); true }) }.padding(12.dp),
        horizontalArrangement = Arrangement.spacedBy(14.dp), verticalAlignment = Alignment.CenterVertically,
    ) {
        AuthenticatedArtwork(artworkUrl, authorizationHeader, item.track.title, Modifier.size(48.dp))
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(5.dp)) {
            Text(item.track.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text(item.track.artist.ifBlank { "Unknown artist" }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text("${(fraction * 100).toInt()}% listened", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        MediaTypeBadge(kind)
    }
        DropdownMenu(expanded = menuExpanded, onDismissRequest = { menuExpanded = false }) {
            DropdownMenuItem(text = { Text("Remove from Home") }, onClick = { menuExpanded = false; onDismiss() })
        }
    }
}
