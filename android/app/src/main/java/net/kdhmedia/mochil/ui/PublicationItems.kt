package net.kdhmedia.mochil.ui

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

@Composable
private fun BookCard(book: BookSummary, coverUrl: String?, header: String?, onRead: (BookSummary) -> Unit, onCollect: (BookSummary) -> Unit) {
    Column(Modifier.gridCard().combinedClickable(onClickLabel = "Read ${book.title}", role = Role.Button, onClick = { onRead(book) }, onLongClickLabel = "Add ${book.title} to collection", onLongClick = { onCollect(book) }).padding(10.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Box {
            PublicationArtwork(coverUrl, header, "Cover of ${book.title}", Modifier.fillMaxWidth().aspectRatio(0.67f))
            if (book.readAlong) ReadAlongBadge(Modifier.align(Alignment.BottomStart).padding(start = 6.dp, bottom = 10.dp))
            CollectButton("Add ${book.title} to a collection", { onCollect(book) }, Modifier.align(Alignment.TopEnd).padding(6.dp))
            if (book.readingFraction != null || book.finished) ThinProgress(
                progress = { if (book.finished) 1f else book.readingFraction?.toFloat() ?: 0f },
                modifier = Modifier.fillMaxWidth().height(4.dp).align(Alignment.BottomCenter),
            )
        }
        Text(book.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, minLines = 2, maxLines = 2, overflow = TextOverflow.Ellipsis)
        Text(book.authors.joinToString(), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
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
