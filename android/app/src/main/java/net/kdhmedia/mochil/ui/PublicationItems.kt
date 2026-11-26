package net.kdhmedia.mochil.ui

import androidx.compose.material3.TextButton
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
import androidx.compose.ui.graphics.Color
import androidx.compose.material.icons.filled.Headphones
import androidx.compose.material.icons.filled.Check
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
import net.kdhmedia.mochil.ui.kit.LucideIcons
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
internal fun BookGrid(books: List<BookSummary>, coverUrl: (BookSummary) -> String?, header: String?, state: LibraryState?, onRead: (BookSummary) -> Unit, onCollect: (BookSummary) -> Unit, onMore: () -> Unit, onRetry: () -> Unit, kind: String = "") {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(136.dp), contentPadding = PaddingValues(20.dp),
        horizontalArrangement = Arrangement.spacedBy(14.dp), verticalArrangement = Arrangement.spacedBy(22.dp),
    ) {
        val libraryKind = kind.ifEmpty { state?.selected?.kind.orEmpty() }
        items(books, key = { catalogueItemIdentityKey(it.libraryId, it.id) }) { book -> BookCard(book, libraryKind, coverUrl(book), header, onRead, onCollect) }
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

/** The bar is the only reading state on a card now, so it is drawn from the first sliver: a book just marked "Reading"
 *  sits at 0.1%, and the old card said "Started" exactly there. A finished book has the tick instead. */
internal fun bookProgressBarShown(book: BookSummary): Boolean =
    !book.finished && (book.readingFraction ?: 0.0) > 0.0

/** Where the book sits in its series, as the web's plate: "Vol. 3". Only a stored index that is a real position counts,
 *  and a number that is the title ("1984") is never one. */
internal fun bookVolumePlate(book: BookSummary): String? {
    val stored = book.seriesIndex ?: 0.0
    val inSeries = book.series.trim().isNotEmpty() && !book.series.trim().equals(book.title.trim(), ignoreCase = true)
    if (inSeries && stored > 0 && stored % 1.0 == 0.0) return "Vol. ${stored.toInt()}"
    // A zero-padded title prefix ("01 Dies the Fire") is a position too, as the web's titles.ts still reads it.
    val match = Regex("""^0(\d{1,2})\s*[-–—:.]?\s+(?=\S)""").find(book.title) ?: return null
    return "Vol. ${match.groupValues[1].toInt()}"
}

@Composable
internal fun BookCard(
    book: BookSummary, kind: String, coverUrl: String?, header: String?,
    onRead: (BookSummary) -> Unit, onCollect: (BookSummary) -> Unit,
    /** A synthetic cover for goldens; the shelf passes null and the real artwork is loaded. */
    art: (@Composable (Modifier) -> Unit)? = null,
    /** Long press removes this card from Discover; null means the long press adds it to a collection instead. */
    onDismiss: (() -> Unit)? = null,
) {
    val glyph = remember(kind, book.libraryId) { libraryGlyph(kind, book.libraryId) }
    val fraction = if (book.finished) 1f else (book.readingFraction ?: 0.0).toFloat()
    Column(
        Modifier.gridCard().combinedClickable(
            onClickLabel = "Read ${book.title}", role = Role.Button, onClick = { onRead(book) },
            onLongClickLabel = if (onDismiss != null) "Remove ${book.title} from Discover" else "Add ${book.title} to collection",
            onLongClick = { if (onDismiss != null) onDismiss() else onCollect(book) },
        ).padding(8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        // The poster is inset in the card at radius 8, as on the web; the marks sit on it, never on a text line below.
        Box(Modifier.fillMaxWidth().aspectRatio(0.67f).clip(RoundedCornerShape(8.dp))) {
            if (art != null) art(Modifier.fillMaxSize()) else PublicationArtwork(coverUrl, header, "Cover of ${book.title}", Modifier.fillMaxSize())
            ArtworkKindBadge(glyph, modifier = Modifier.align(Alignment.TopStart).padding(8.dp))
            // Stacked, as the web's .topRight is: a row of three would run into the kind badge from about 1.3x font.
            Column(Modifier.align(Alignment.TopEnd).padding(8.dp), horizontalAlignment = Alignment.End, verticalArrangement = Arrangement.spacedBy(4.dp)) {
                bookVolumePlate(book)?.let { ArtworkPlate(it) }
                // On the same dark plate as the kind glyph: a bare white icon disappeared on a pale cover in light themes.
                if (book.readAlong) ArtworkKindBadge(LucideIcons.Headphones, "Has its own recording")
                CollectButton("Add ${book.title} to a collection", { onCollect(book) })
            }
            if (book.finished) BookFinishedBadge(Modifier.align(Alignment.BottomStart).padding(8.dp))
            else if (bookProgressBarShown(book)) ArtworkProgressScrim({ fraction }, "${(fraction * 100).toInt()}% read", Modifier.align(Alignment.BottomCenter))
        }
        // Title two lines and author one, so every author in a row lands on the same line without a fixed height;
        // 4 between them, as the web's .textBlock gap, and 8 was the poster-to-block gap above.
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(book.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, minLines = 2, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text(book.authors.joinToString().ifEmpty { " " }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, minLines = 1, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
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

/**
 * A finished book's one mark: an 18dp solid green tick in the cover's free corner, matching the web's
 * `.cover-done` (book-cards.js, card-surface.css, 0.1.83). It replaces both a "Read" pill and a 100%-filled
 * progress strip, which on a shelf of mostly-finished books read as a wall of green bars for a fact that needs
 * one glance. Opaque (not the read-along badge's translucent black), because green-on-black would wash out and
 * the state is important enough to want full contrast.
 */
@Composable
internal fun BookFinishedBadge(modifier: Modifier = Modifier) {
    Box(modifier.size(18.dp).background(Color(0xFF2FAE5F), CircleShape), contentAlignment = Alignment.Center) {
        Icon(Icons.Default.Check, contentDescription = "Finished", modifier = Modifier.size(12.dp), tint = Color(0xFF04211F))
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
internal fun PublicationEmpty(message: String, actionLabel: String? = null, onAction: () -> Unit = {}) {
    Box(Modifier.fillMaxSize().padding(32.dp), contentAlignment = Alignment.Center) {
        // One muted line, and the one thing that can be done about it.
        Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(message, color = MaterialTheme.colorScheme.onSurfaceVariant)
            if (actionLabel != null) TextButton(onClick = onAction) { Text(actionLabel) }
        }
    }
}

private fun formatSeriesIndex(index: Double): String =
    if (index % 1.0 == 0.0) "Volume ${index.toInt()}" else "Volume $index"
