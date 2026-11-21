package net.kdhmedia.mochil.ui

import androidx.compose.foundation.border
import androidx.compose.foundation.background
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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.material.icons.filled.Headphones
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import net.kdhmedia.mochil.catalogue.readingStatus
import net.kdhmedia.mochil.catalogue.ReadingStatus
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import coil3.network.NetworkHeaders
import coil3.network.httpHeaders
import coil3.request.ImageRequest
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.catalogueItemIdentityKey
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

@Composable
internal fun BookDetail(
    book: BookSummary,
    coverUrl: String?,
    header: String?,
    onBack: () -> Unit,
    onQueue: (BookSummary) -> Unit,
    queued: Boolean,
) {
    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back to books") }
            Text("Book details", style = MaterialTheme.typography.titleLarge)
        }
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 20.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            PublicationArtwork(coverUrl, header, "Cover of ${book.title}", Modifier.size(width = 160.dp, height = 240.dp).align(Alignment.CenterHorizontally))
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(book.title, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
                if (book.authors.isNotEmpty()) Text(book.authors.joinToString(), style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            HorizontalDivider()
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                if (book.series.isNotBlank()) DetailValue("Series", book.series + book.seriesIndex?.let { " · ${formatSeriesIndex(it)}" }.orEmpty())
                if (book.format.isNotBlank()) DetailValue("Format", book.format.uppercase())
                if (book.language.isNotBlank()) DetailValue("Language", book.language)
                if (book.size > 0) DetailValue("File size", formatByteSize(book.size))
            }
            if (book.description.isNotBlank()) {
                HorizontalDivider()
                Text("About this book", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                Text(book.description, style = MaterialTheme.typography.bodyLarge)
            }
            // Queuing is separate from reading on purpose: the reader may be lining up three books to
            // fetch in the background rather than opening this one now.
            OutlinedButton(onClick = { onQueue(book) }, enabled = !queued, modifier = Modifier.fillMaxWidth()) {
                Text(if (queued) "In the download queue" else "Add to download queue")
            }
        }
    }
}

@Composable
private fun DetailValue(label: String, value: String) {
    Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, style = MaterialTheme.typography.bodyLarge)
    }
}

@Composable
internal fun BookGrid(books: List<BookSummary>, coverUrl: (BookSummary) -> String?, header: String?, state: LibraryState?, onRead: (BookSummary) -> Unit, onCollect: (BookSummary) -> Unit, onMore: () -> Unit, onRetry: () -> Unit) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(136.dp), contentPadding = PaddingValues(20.dp),
        horizontalArrangement = Arrangement.spacedBy(14.dp), verticalArrangement = Arrangement.spacedBy(22.dp),
    ) {
        items(books, key = { catalogueItemIdentityKey(it.libraryId, it.id) }) { book -> BookCard(book, coverUrl(book), header, onRead, onCollect) }
        if (state != null) publicationPaging(state, onMore, onRetry)
    }
}

@Composable
internal fun BookList(books: List<BookSummary>, coverUrl: (BookSummary) -> String?, header: String?, state: LibraryState?, onRead: (BookSummary) -> Unit, onCollect: (BookSummary) -> Unit, onMore: () -> Unit, onRetry: () -> Unit, table: Boolean = false, showAuthors: Boolean = true) {
    LazyColumn(Modifier.fillMaxSize()) {
        items(books, key = { catalogueItemIdentityKey(it.libraryId, it.id) }) { book ->
            if (table) BookTableRow(book, coverUrl(book), header, showAuthors, onRead, onCollect) else BookListRow(book, coverUrl(book), header, onRead, onCollect)
        }
        if (state != null) publicationPaging(state, onMore, onRetry)
    }
}

/**
 * What a card says under its cover, in the one or two words a reader scans for. It names the state or the
 * format, never a feature: "Narrated" is true of exactly those books that carry a recording, where the mock
 * ups this card follows had an "AI Whispersync ready" badge on everything.
 */
internal fun bookCardTag(book: BookSummary): String = when {
    // The foot of the cover already says where the reader is, so the pill carries the one fact it cannot.
    book.readAlong -> "Narrated"
    book.status == "on_hold" -> "On hold"
    book.status == "dropped" -> "Dropped"
    else -> ""
}

/** The line printed over the artwork's foot: what fraction is read, or that the book is finished. */
internal fun bookProgressLabel(book: BookSummary): String? {
    val fraction = book.readingFraction ?: 0.0
    if (book.finished) return "Read"
    if (fraction <= 0.0) return null
    // A book opened once sits at a fraction of a percent; "0% read" is a number that says nothing, and a bar
    // drawn at that width is a 1px sliver that reads as a fault. Under one percent the foot says "Started".
    return if ((fraction * 100).toInt() < 1) "Started" else "${(fraction * 100).toInt()}% read"
}

/** The bar under the foot, only once there is a width worth drawing. */
internal fun bookProgressBarShown(book: BookSummary): Boolean {
    if (book.finished) return true
    val fraction = book.readingFraction ?: 0.0
    return fraction > 0.0 && (fraction * 100).toInt() >= 1
}

/** Where the book sits in its series, and nothing else: "Book 3". The stored index only counts when the book
 *  actually belongs to a series (every row carries a 1 by default), otherwise a zero-padded title prefix is the
 *  only honest source. A number that is the title ("1984") is never a position. */
internal fun bookSeriesLabel(book: BookSummary): String {
    val stored = book.seriesIndex ?: 0.0
    val inSeries = book.series.trim().isNotEmpty() && !book.series.trim().equals(book.title.trim(), ignoreCase = true)
    if (inSeries && stored > 0 && stored % 1.0 == 0.0) return "Book ${stored.toInt()}"
    val match = Regex("""^0(\d{1,2})\s*[-–—:.]?\s+""").find(book.title) ?: return ""
    return "Book ${match.groupValues[1].toInt()}"
}

@Composable
private fun BookCard(book: BookSummary, coverUrl: String?, header: String?, onRead: (BookSummary) -> Unit, onCollect: (BookSummary) -> Unit) {
    val fraction = if (book.finished) 1f else (book.readingFraction ?: 0.0).toFloat()
    val progressLabel = bookProgressLabel(book)
    Column(
        Modifier.gridCard().combinedClickable(onClickLabel = "Read ${book.title}", role = Role.Button, onClick = { onRead(book) }, onLongClickLabel = "Add ${book.title} to collection", onLongClick = { onCollect(book) }).padding(4.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Box(Modifier.fillMaxWidth().aspectRatio(0.67f).clip(RoundedCornerShape(16.dp))) {
            PublicationArtwork(coverUrl, header, "Cover of ${book.title}", Modifier.fillMaxSize())
            Row(Modifier.align(Alignment.TopEnd).padding(6.dp), horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                if (book.readAlong) ReadAlongBadge()
                CollectButton("Add ${book.title} to a collection", { onCollect(book) })
            }
            // The foot of the artwork carries the progress, so the grid stays a wall of covers rather than a
            // wall of covers with a second column of text under each one.
            if (progressLabel != null) {
                Column(
                    Modifier.align(Alignment.BottomCenter).fillMaxWidth()
                        .background(Brush.verticalGradient(listOf(Color.Transparent, Color.Black.copy(alpha = .78f))))
                        .padding(start = 10.dp, end = 10.dp, top = 18.dp, bottom = 9.dp),
                    verticalArrangement = Arrangement.spacedBy(5.dp),
                ) {
                    Text(progressLabel, style = MaterialTheme.typography.labelMedium, color = Color.White)
                    if (bookProgressBarShown(book)) {
                        ThinProgress(progress = { fraction }, modifier = Modifier.fillMaxWidth().height(3.dp))
                    }
                }
            }
        }
        Text(book.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, minLines = 2, maxLines = 2, overflow = TextOverflow.Ellipsis)
        val series = bookSeriesLabel(book)
        if (series.isNotEmpty()) {
            Text(series, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary, maxLines = 1)
        }
        Text(book.authors.joinToString(), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
        val tag = bookCardTag(book)
        if (tag.isNotEmpty()) CardTagPill(tag)
    }
}

/** The small outlined pill under a card: state or format in one word, never a claim about the software. */
@Composable
private fun CardTagPill(label: String) {
    Text(
        label,
        style = MaterialTheme.typography.labelSmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        maxLines = 1,
        modifier = Modifier
            .clip(RoundedCornerShape(50))
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(50))
            .padding(horizontal = 8.dp, vertical = 3.dp),
    )
}

/** The words "Read-along" with the headphones, for rows where there is no cover corner to mark. */
@Composable
internal fun ReadAlongLabel(modifier: Modifier = Modifier) {
    Row(modifier, verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
        Icon(Icons.Default.Headphones, contentDescription = null, modifier = Modifier.size(14.dp), tint = MaterialTheme.colorScheme.primary)
        Text("Read-along", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
    }
}

/** Marks a book that carries its own recording, so it can be told from a plain one at a glance. */
@Composable
internal fun ReadAlongBadge(modifier: Modifier = Modifier) {
    Box(modifier.size(24.dp).background(Color.Black.copy(alpha = 0.62f), CircleShape), contentAlignment = Alignment.Center) {
        Icon(Icons.Default.Headphones, contentDescription = "Has its own recording", modifier = Modifier.size(15.dp), tint = Color.White)
    }
}

@Composable
internal fun PublicationArtwork(url: String?, header: String?, description: String, modifier: Modifier) {
    val context = LocalContext.current
    val model = remember(url, header) {
        val headers = NetworkHeaders.Builder().apply { if (header != null) set("Authorization", header) }.build()
        ImageRequest.Builder(context).data(url).httpHeaders(headers).build()
    }
    var loading by remember(model) { mutableStateOf(!url.isNullOrBlank()) }
    AsyncImage(
        model = model,
        contentDescription = description,
        contentScale = ContentScale.Crop,
        onState = { loading = it is coil3.compose.AsyncImagePainter.State.Loading || it is coil3.compose.AsyncImagePainter.State.Empty },
        modifier = modifier.clip(RoundedCornerShape(12.dp)).then(if (loading) Modifier.shimmer() else Modifier.background(MaterialTheme.colorScheme.surfaceVariant)),
    )
}

internal fun androidx.compose.foundation.lazy.grid.LazyGridScope.publicationPaging(state: LibraryState, onMore: () -> Unit, onRetry: () -> Unit) {
    if (state.error != null) item(key = "page-error") { InlineError(state.error, onRetry) }
    else if (state.nextCursor != null) item(key = "next-page") {
        LaunchedEffect(state.nextCursor) { onMore() }
        PagingIndicator()
    }
}

internal fun androidx.compose.foundation.lazy.LazyListScope.publicationPaging(state: LibraryState, onMore: () -> Unit, onRetry: () -> Unit) {
    if (state.error != null) item(key = "page-error") { InlineError(state.error, onRetry) }
    else if (state.nextCursor != null) item(key = "next-page") {
        LaunchedEffect(state.nextCursor) { onMore() }
        PagingIndicator()
    }
}

@Composable
private fun PagingIndicator() = Box(Modifier.fillMaxWidth().height(72.dp), contentAlignment = Alignment.Center) {
    if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(26.dp)) else CircularProgressIndicator(Modifier.size(26.dp))
}

@Composable
internal fun InlineError(message: String, onRetry: () -> Unit) {
    Column(Modifier.fillMaxWidth().padding(20.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(message, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Button(onClick = onRetry) { Text("Try again") }
    }
}

@Composable
internal fun PublicationEmpty(message: String) {
    Box(Modifier.fillMaxSize().padding(32.dp), contentAlignment = Alignment.Center) {
        Text(message, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

private fun formatSeriesIndex(index: Double): String =
    if (index % 1.0 == 0.0) "Volume ${index.toInt()}" else "Volume $index"
