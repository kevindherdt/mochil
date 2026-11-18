package net.kdhmedia.mochil.ui

import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.ExpandLess
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.border
import androidx.compose.foundation.background
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.SeriesSummary
import net.kdhmedia.mochil.catalogue.catalogueItemIdentityKey

@Composable
internal fun SeriesGrid(state: LibraryState, coverUrl: (SeriesSummary) -> String?, header: String?, onOpen: (SeriesSummary) -> Unit, onCollect: (SeriesSummary) -> Unit, onMore: () -> Unit, onRetry: () -> Unit) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(136.dp), contentPadding = PaddingValues(20.dp),
        horizontalArrangement = Arrangement.spacedBy(14.dp), verticalArrangement = Arrangement.spacedBy(22.dp),
    ) {
        items(state.series, key = { catalogueItemIdentityKey(it.libraryId, it.name) }) { series ->
            Column(Modifier.gridCard().combinedClickable(onClickLabel = "Open ${series.name}", role = Role.Button, onClick = { onOpen(series) }, onLongClickLabel = "Add ${series.name} to collection", onLongClick = { onCollect(series) }).padding(10.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                // The outer box keeps every card's title on one line; a stack sits centred inside it with room for the tilt.
                Box(Modifier.fillMaxWidth().aspectRatio(0.75f), contentAlignment = Alignment.Center) { Box(Modifier.fillMaxWidth(if (series.count > 1) .88f else 1f)) {
                    if (series.count > 1) SeriesStack(coverUrl(series), header, series.name, Modifier.fillMaxWidth().aspectRatio(0.75f))
                    PublicationArtwork(coverUrl(series), header, "Cover of ${series.name}", Modifier.fillMaxWidth().aspectRatio(0.75f))
                    CollectButton("Add ${series.name} to a collection", { onCollect(series) }, Modifier.align(Alignment.TopEnd).padding(6.dp))
                    Surface(Modifier.align(Alignment.BottomEnd).padding(8.dp), shape = RoundedCornerShape(8.dp), color = MaterialTheme.colorScheme.surface, contentColor = MaterialTheme.colorScheme.onSurface) {
                        Text(seriesCountLabel(series.count), Modifier.padding(horizontal = 7.dp, vertical = 4.dp), style = MaterialTheme.typography.labelSmall)
                    }
                }
                }
                Text(series.name, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, minLines = 2, maxLines = 2, overflow = TextOverflow.Ellipsis)
                series.authors.let { Text(it.joinToString(), minLines = 1, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis) }
            }
        }
        publicationPaging(state, onMore, onRetry)
    }
}

@Composable
internal fun SeriesList(state: LibraryState, coverUrl: (BookSummary) -> String?, header: String?, onOpen: (SeriesSummary) -> Unit, onReadBook: (BookSummary) -> Unit, collectionActions: PublicationCollectionActions, onMore: () -> Unit, onRetry: () -> Unit, onRetrySeries: () -> Unit) {
    LazyColumn(Modifier.fillMaxSize()) {
        state.series.forEach { series ->
            val expanded = state.expandedSeries == series.name
            item(key = "series:${series.libraryId}:${series.name}") {
                Row(Modifier.fillMaxWidth().combinedClickable(onClickLabel = if (expanded) "Collapse ${series.name}" else "Expand ${series.name}", role = Role.Button, onClick = { onOpen(series) }, onLongClickLabel = "Add ${series.name} to collection", onLongClick = { collectionActions.series(series) }).padding(horizontal = 20.dp, vertical = 12.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Icon(if (expanded) Icons.Default.ExpandLess else Icons.Default.ExpandMore, contentDescription = if (expanded) "Collapse ${series.name}" else "Expand ${series.name}")
                    Column(Modifier.weight(1f)) {
                        Text(series.name, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium)
                        Text(seriesCaption(series), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                MotionVisibility(expanded) { ExpandedSeries(state, coverUrl, header, onReadBook, collectionActions.book, onMore, onRetrySeries) }
            }
        }
        publicationPaging(state, onMore, onRetry)
    }
}

@Composable
private fun ExpandedSeries(state: LibraryState, coverUrl: (BookSummary) -> String?, header: String?, onReadBook: (BookSummary) -> Unit, onCollect: (BookSummary) -> Unit, onMore: () -> Unit, onRetry: () -> Unit) {
    val seriesAuthors = state.series.firstOrNull { it.name == state.expandedSeries }?.authors.orEmpty()
    when {
        state.groupLoading -> AccessibleLoadingIndicator("Loading series", Modifier.fillMaxWidth().height(72.dp), size = 26.dp)
        state.groupError != null && state.seriesBooks.isEmpty() -> InlineError(state.groupError, onRetry)
        else -> Column {
            state.seriesBooks.forEach { book -> BookTableRow(book, coverUrl(book), header, shouldShowBookAuthors(book.authors, seriesAuthors), onReadBook, onCollect) }
            state.groupError?.let { InlineError(it, onRetry) }
            if (state.groupNextCursor != null) Button(onClick = onMore, modifier = Modifier.align(Alignment.CenterHorizontally), enabled = !state.groupLoadingMore) { Text(if (state.groupLoadingMore) "Loading…" else "Load more") }
        }
    }
}

@Composable
internal fun SeriesDetail(series: SeriesSummary, state: LibraryState, coverUrl: (BookSummary) -> String?, header: String?, onBack: () -> Unit, onRetry: () -> Unit, onMore: () -> Unit, onReadBook: (BookSummary) -> Unit, collectionActions: PublicationCollectionActions) {
    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back to series") }
            Column(Modifier.weight(1f)) {
                Text(series.name, style = MaterialTheme.typography.titleLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(seriesCaption(series), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        when {
            state.groupLoading -> AccessibleLoadingIndicator("Loading series", Modifier.fillMaxSize())
            state.groupError != null && state.seriesBooks.isEmpty() -> InlineError(state.groupError, onRetry)
            else -> BookGrid(state.seriesBooks, coverUrl, header, state.groupPagingState(), onReadBook, collectionActions.book, onMore, onRetry)
        }
    }
}

internal fun LibraryState.groupPagingState() = copy(nextCursor = groupNextCursor, loadingMore = groupLoadingMore, error = groupError)

private fun seriesCaption(series: SeriesSummary): String = buildList {
    if (series.authors.isNotEmpty()) add(series.authors.joinToString())
    add("${series.count} ${if (series.count == 1) "book" else "books"}")
}.joinToString(" · ")

internal fun seriesCountLabel(count: Int) = count.toString()

internal fun shouldShowBookAuthors(bookAuthors: List<String>, seriesAuthors: List<String>): Boolean {
    if (bookAuthors.isEmpty()) return false
    if (seriesAuthors.isEmpty()) return true
    return bookAuthors.map(::normalizedAuthor).toSet() != seriesAuthors.map(::normalizedAuthor).toSet()
}

private fun normalizedAuthor(value: String) = value.trim().lowercase()

/** Two more copies of the cover peek out behind the front one, tilted either way and darkened, as on the web. The image is already loaded, so this costs no extra request. */
@Composable
internal fun SeriesStack(url: String?, header: String?, name: String, modifier: Modifier) {
    Box(modifier) {
        StackedCover(url, header, name, Modifier.matchParentSize(), rotation = -4f, shiftX = -7f, dim = .45f)
        StackedCover(url, header, name, Modifier.matchParentSize(), rotation = 3.5f, shiftX = 7f, dim = .3f)
    }
}

@Composable
private fun StackedCover(url: String?, header: String?, name: String, modifier: Modifier, rotation: Float, shiftX: Float, dim: Float) {
    Box(modifier.graphicsLayer { rotationZ = rotation; translationX = shiftX; translationY = -3f; scaleX = .95f; scaleY = .95f }) {
        PublicationArtwork(url, header, "Cover of $name", Modifier.matchParentSize())
        Box(Modifier.matchParentSize().clip(RoundedCornerShape(12.dp)).background(androidx.compose.ui.graphics.Color.Black.copy(alpha = dim)))
    }
}
