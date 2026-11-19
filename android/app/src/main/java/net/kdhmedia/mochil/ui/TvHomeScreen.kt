package net.kdhmedia.mochil.ui

import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.relocation.BringIntoViewRequester
import androidx.compose.foundation.relocation.bringIntoViewRequester
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.gestures.snapping.rememberSnapFlingBehavior
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.focusRestorer
import androidx.compose.foundation.focusGroup
import androidx.compose.ui.semantics.CustomAccessibilityAction
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.customActions
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.catalogueItemIdentityKey
import net.kdhmedia.mochil.dashboard.ContinueListening
import net.kdhmedia.mochil.dashboard.DashboardState
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.video.VideoItem

@Composable
internal fun TvHomeScreen(
    state: DashboardState,
    artworkUrl: (AudioTrack) -> String,
    videoArtworkUrl: (VideoItem) -> String,
    videoBackdropUrl: (VideoItem) -> String = { "" },
    authorizationHeader: String?,
    onListen: (ContinueListening) -> Unit,
    onWatch: (VideoItem) -> Unit,
    onDismissListening: (ContinueListening) -> Unit,
    onDismissWatching: (VideoItem) -> Unit,
    onRetry: () -> Unit,
    focusRail: () -> Unit,
) {
    val retryFocus = remember { FocusRequester() }
    val firstFocus = remember { FocusRequester() }
    val empty = state.listening.isEmpty() && state.watching.isEmpty()
    // Open on the first thing to resume; with nothing to resume (or a failed load) the rail is the place to be.
    LaunchedEffect(state.error, state.loading, empty) {
        when {
            state.loading && empty -> Unit
            state.error != null -> retryFocus.requestFocus()
            empty -> focusRail()
            else -> runCatching { firstFocus.requestFocus() }
        }
    }
    val firstWatching = state.watching.firstOrNull()
    val firstListening = state.listening.firstOrNull()
    val initial = when {
        firstWatching != null -> TvHomeFocus("Continue watching", tvCleanTitle(firstWatching.displayTitle), videoContext(firstWatching), videoArtworkUrl(firstWatching), firstWatching.progressFraction, firstWatching.overview, videoBackdropUrl(firstWatching))
        firstListening != null -> TvHomeFocus("Continue listening", firstListening.track.title, firstListening.track.artist.ifBlank { "Unknown artist" }, artworkUrl(firstListening.track), listeningFraction(firstListening))
        else -> null
    }
    var focus by remember { mutableStateOf<TvHomeFocus?>(null) }
    val shown = focus ?: initial
    val background = MaterialTheme.colorScheme.background
    Box(Modifier.fillMaxSize()) {
        // The thing under the remote becomes the screen backdrop, so Home always looks like what you are about to watch.
        TvBackdropEffect(shown?.let { it.backdrop.ifBlank { it.art } }.orEmpty())
        Column(Modifier.fillMaxSize().padding(start = 4.dp, end = 44.dp, top = 28.dp)) {
            Column(Modifier.fillMaxWidth(.7f).padding(start = 8.dp, bottom = 12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                if (shown != null) {
                    Text(shown.title, style = MaterialTheme.typography.headlineSmall.copy(fontSize = 24.sp, lineHeight = 28.sp), fontWeight = FontWeight.SemiBold, minLines = 2, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    val line = listOf(shown.detail, shown.progress.takeIf { it >= .01f }?.let { "${(it * 100).toInt()}% watched" }.orEmpty()).filter(String::isNotBlank).joinToString(" · ")
                    if (line.isNotBlank()) Text(line, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
            }
            LazyColumn(
                Modifier.weight(1f).focusRestorer().focusGroup(),
                contentPadding = PaddingValues(bottom = 40.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                if (state.error != null) {
                    item {
                        Column(
                            Modifier.fillMaxWidth().semantics { liveRegion = LiveRegionMode.Polite },
                            horizontalAlignment = Alignment.CenterHorizontally,
                            verticalArrangement = Arrangement.spacedBy(14.dp),
                        ) {
                            Text(state.error, style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            Button(onClick = onRetry, modifier = Modifier.focusRequester(retryFocus).tvFocusBorder()) { Text("Try again") }
                        }
                    }
                }
                if (state.loading && empty) {
                    item { AccessibleLoadingIndicator("Loading Home", Modifier.fillMaxWidth().height(180.dp)) }
                } else if (empty) {
                    if (state.error == null) item { Text("Start something on Mochil and it will appear here.", style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                } else {
                    if (state.watching.isNotEmpty()) item(key = "watching") {
                        TvSection("Continue watching", state.watching, { catalogueItemIdentityKey(it.libraryId, it.id) }, authorizationHeader, onWatch, onDismissWatching, firstFocus = firstFocus.takeIf { firstWatching != null },
                            art = { videoArtworkUrl(it) }, name = { tvCleanTitle(it.displayTitle) }, detail = { videoContext(it) }, progress = VideoItem::progressFraction,
                            onFocused = { focus = TvHomeFocus("Continue watching", tvCleanTitle(it.displayTitle), videoContext(it), videoArtworkUrl(it), it.progressFraction, it.overview, videoBackdropUrl(it)) })
                    }
                    if (state.listening.isNotEmpty()) item(key = "listening") {
                        TvSection("Continue listening", state.listening, { it.track.identityKey }, authorizationHeader, onListen, onDismissListening, firstFocus = firstFocus.takeIf { firstWatching == null },
                            art = { artworkUrl(it.track) }, name = { it.track.title }, detail = { it.track.artist.ifBlank { "Unknown artist" } }, progress = ::listeningFraction, square = true,
                            onFocused = { focus = TvHomeFocus("Continue listening", it.track.title, it.track.artist.ifBlank { "Unknown artist" }, artworkUrl(it.track), listeningFraction(it)) })
                    }
                }
            }
        }
    }
}

private data class TvHomeFocus(val label: String, val title: String, val detail: String, val art: String, val progress: Float, val overview: String = "", val backdrop: String = "")

@Composable
private fun <T> TvSection(
    title: String,
    values: List<T>,
    itemKey: (T) -> Any,
    authorizationHeader: String?,
    onOpen: (T) -> Unit,
    onDismiss: ((T) -> Unit)?,
    firstFocus: FocusRequester?,
    art: (T) -> String,
    name: (T) -> String,
    detail: (T) -> String,
    progress: (T) -> Float,
    onFocused: (T) -> Unit,
    square: Boolean = false,
) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold, modifier = Modifier.padding(start = 8.dp))
        LazyRow(Modifier.focusRestorer().focusGroup(), contentPadding = PaddingValues(start = 8.dp, end = 24.dp, top = 6.dp, bottom = 6.dp), horizontalArrangement = Arrangement.spacedBy(14.dp)) {
            itemsIndexed(values, key = { _, value -> itemKey(value) }) { index, value ->
                val tileTitle = name(value)
                // Focus scrolling asks for the focused frame only, so on a short window at a large text size the tile's
                // title stayed below the fold and its ring lost the room it grows into. Ask for the whole tile instead.
                val tile = remember { BringIntoViewRequester() }
                val scope = rememberCoroutineScope()
                var tileSize by remember { mutableStateOf(IntSize.Zero) }
                val ring = with(LocalDensity.current) { 8.dp.toPx() }
                // Same card as the library shelves: the art on its own, the title and detail beneath it.
                Column(
                    Modifier.width(if (square) 132.dp else 176.dp).bringIntoViewRequester(tile).onSizeChanged { tileSize = it },
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    TvResumeFrame(
                        tileTitle, { onOpen(value) }, onDismiss?.let { dismiss -> { dismiss(value) } },
                        Modifier.fillMaxWidth().aspectRatio(if (square) 1f else 16f / 9f).then(if (index == 0 && firstFocus != null) Modifier.focusRequester(firstFocus) else Modifier),
                        RoundedCornerShape(12.dp),
                        onFocused = {
                            onFocused(value)
                            val size = tileSize
                            if (size != IntSize.Zero) scope.launch {
                                tile.bringIntoView(Rect(-ring, -ring, size.width + ring, size.height + ring))
                            }
                        },
                    ) {
                        Box(Modifier.fillMaxSize()) {
                            AuthenticatedArtwork(art(value), authorizationHeader, tileTitle, Modifier.fillMaxSize(), cornerRadius = 0.dp)
                            if (progress(value) > 0f) TvProgressBar(progress(value), Modifier.fillMaxWidth().align(Alignment.BottomCenter), 4.dp)
                        }
                    }
                    Column(Modifier.padding(horizontal = 4.dp)) {
                        Text(tileTitle, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        Text(detail(value).ifBlank { " " }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                }
            }
        }
    }
}

/** One focusable Home tile: opens on click, offers "Remove from Home" on long press and as an accessibility action. */
@Composable
private fun TvResumeFrame(title: String, onClick: () -> Unit, onDismiss: (() -> Unit)?, modifier: Modifier, shape: androidx.compose.ui.graphics.Shape, onFocused: () -> Unit = {}, content: @Composable () -> Unit) {
    var focused by remember { mutableStateOf(false) }
    var menuExpanded by remember { mutableStateOf(false) }
    androidx.compose.foundation.layout.Box {
        Card(
            modifier = modifier.onFocusChanged { focused = it.isFocused; if (it.isFocused) onFocused() }
                .tvRemoteLongPress(onDismiss != null) { menuExpanded = true }
                .tvFocusVisual(focused, shape),
            shape = shape,
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
        ) {
            androidx.compose.foundation.layout.Box(
                Modifier.fillMaxSize().combinedClickable(
                    onClickLabel = "Open $title",
                    onLongClickLabel = if (onDismiss == null) null else "More options",
                    role = Role.Button,
                    onClick = onClick,
                    onLongClick = onDismiss?.let { { menuExpanded = true } },
                ).semantics {
                    if (onDismiss != null) customActions = listOf(CustomAccessibilityAction("Remove from Home") {
                        onDismiss()
                        true
                    })
                },
            ) { content() }
        }
        if (onDismiss != null) TvContextDropdownMenu(expanded = menuExpanded, onDismissRequest = { menuExpanded = false }) {
            DropdownMenuItem(text = { Text("Remove from Home") }, modifier = Modifier.tvFocusBorder(), onClick = {
                menuExpanded = false
                onDismiss()
            })
        }
    }
}

private fun listeningFraction(item: ContinueListening): Float =
    if (item.durationSeconds > 0) (item.positionSeconds / item.durationSeconds).toFloat().coerceIn(0f, 1f) else 0f

private fun videoContext(item: VideoItem): String = listOfNotNull(
    item.show.takeIf(String::isNotBlank)?.let(::tvCleanTitle),
    item.episode.takeIf { it > 0 }?.let { "S${item.season}E$it" },
).joinToString(" · ")
