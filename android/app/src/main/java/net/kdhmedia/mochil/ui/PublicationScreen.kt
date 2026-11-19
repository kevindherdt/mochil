package net.kdhmedia.mochil.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.relocation.bringIntoViewRequester
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Tune
import androidx.compose.material.icons.outlined.FolderOpen
import androidx.compose.material.icons.outlined.GridView
import androidx.compose.material.icons.outlined.Layers
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.Schedule
import androidx.compose.material.icons.automirrored.filled.ViewList
import androidx.compose.material.icons.filled.GridView
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.VerticalDivider
import androidx.compose.material3.FilterChip
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.draw.clip
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.border
import androidx.compose.foundation.background
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import kotlinx.coroutines.flow.StateFlow
import net.kdhmedia.mochil.catalogue.DownloadQueueItem
import net.kdhmedia.mochil.catalogue.isQueued
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.AuthorSummary
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.PublicationCollection
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.catalogue.PublicationLayout
import net.kdhmedia.mochil.catalogue.SeriesSummary
import net.kdhmedia.mochil.catalogue.currentLayout
import net.kdhmedia.mochil.catalogue.isGrouped

@Composable
fun PublicationScreen(
    state: LibraryState,
    coverUrl: (BookSummary) -> String?,
    seriesCoverUrl: (SeriesSummary) -> String?,
    authorCoverUrl: (AuthorSummary) -> String?,
    collectionCoverUrl: (PublicationCollection) -> String?,
    authorizationHeader: String?,
    onGrouping: (PublicationGrouping) -> Unit,
    onSort: (String) -> Unit,
    onLayout: (PublicationLayout) -> Unit,
    onOpenSeries: (SeriesSummary) -> Unit,
    onCloseSeries: () -> Unit,
    onOpenAuthor: (AuthorSummary) -> Unit,
    onCloseAuthor: () -> Unit,
    onOpenCollection: (net.kdhmedia.mochil.catalogue.PublicationCollection) -> Unit,
    onCloseCollection: () -> Unit,
    onRetrySeries: () -> Unit,
    onReadBook: (BookSummary) -> Unit,
    onQueueBook: (BookSummary) -> Unit,
    /** The live download queue; collected by the banner and the detail button, never stored in the state. */
    downloadQueue: StateFlow<List<DownloadQueueItem>>,
    onCloseBook: () -> Unit,
    onSearch: (String) -> Unit,
    onBucket: (String) -> Unit,
    onDownloadedOnly: (Boolean) -> Unit,
    collectionActions: PublicationCollectionActions,
    onLoadMore: () -> Unit,
    onRetry: () -> Unit,
) {
    var searchExpanded by rememberSaveable { mutableStateOf(false) }
    BoxWithConstraints(Modifier.fillMaxSize()) {
        val twoPane = usesPublicationTwoPane(maxWidth.value.toInt(), state.selectedBook != null || state.selectedSeries != null || state.selectedAuthor != null)
        if (twoPane) {
            BackHandler(onBack = if (state.selectedBook != null) onCloseBook else if (state.selectedAuthor != null) onCloseAuthor else onCloseSeries)
            Row(Modifier.fillMaxSize()) {
                Box(Modifier.weight(.43f)) {
                    PublicationBrowse(state, coverUrl, seriesCoverUrl, authorCoverUrl, collectionCoverUrl, authorizationHeader, onGrouping, onSort, onLayout, searchExpanded, { searchExpanded = !searchExpanded }, onOpenSeries, onOpenAuthor, onOpenCollection, onCloseCollection, onRetrySeries, onReadBook, downloadQueue, onSearch, onBucket, onDownloadedOnly, collectionActions, onLoadMore, onRetry)
                }
                VerticalDivider()
                Box(Modifier.weight(.57f)) {
                    state.selectedBook?.let {
                        QueuedBookDetail(it, coverUrl(it), authorizationHeader, onCloseBook, onQueueBook, downloadQueue)
                    }
                        ?: state.selectedSeries?.let { SeriesDetail(it, state, coverUrl, authorizationHeader, onCloseSeries, onRetrySeries, onLoadMore, onReadBook, collectionActions) }
                        ?: state.selectedAuthor?.let { AuthorDetail(it, state, coverUrl, authorizationHeader, onCloseAuthor, onRetrySeries, onLoadMore, onReadBook, collectionActions) }
                }
            }
        } else if (state.selectedBook != null) {
            BackHandler(onBack = onCloseBook)
            val open = requireNotNull(state.selectedBook)
            QueuedBookDetail(open, coverUrl(open), authorizationHeader, onCloseBook, onQueueBook, downloadQueue)
        } else if (state.selectedSeries != null) {
            BackHandler(onBack = onCloseSeries)
            SeriesDetail(requireNotNull(state.selectedSeries), state, coverUrl, authorizationHeader, onCloseSeries, onRetrySeries, onLoadMore, onReadBook, collectionActions)
        } else if (state.selectedAuthor != null) {
            BackHandler(onBack = onCloseAuthor)
            AuthorDetail(requireNotNull(state.selectedAuthor), state, coverUrl, authorizationHeader, onCloseAuthor, onRetrySeries, onLoadMore, onReadBook, collectionActions)
        } else {
            PublicationBrowse(state, coverUrl, seriesCoverUrl, authorCoverUrl, collectionCoverUrl, authorizationHeader, onGrouping, onSort, onLayout, searchExpanded, { searchExpanded = !searchExpanded }, onOpenSeries, onOpenAuthor, onOpenCollection, onCloseCollection, onRetrySeries, onReadBook, downloadQueue, onSearch, onBucket, onDownloadedOnly, collectionActions, onLoadMore, onRetry)
        }
    }
}

/**
 * The detail pane's queue button. It reads the queue flow itself so a progress tick recomposes this
 * button only; the shelf beside it — and the list of books behind the detail — keeps its frame.
 */
@Composable
private fun QueuedBookDetail(
    book: BookSummary,
    coverUrl: String?,
    header: String?,
    onBack: () -> Unit,
    onQueue: (BookSummary) -> Unit,
    queue: StateFlow<List<DownloadQueueItem>>,
) {
    val queued by queue.collectAsStateWithLifecycle()
    BookDetail(book, coverUrl, header, onBack, onQueue, isQueued(queued, book))
}

@Composable
private fun PublicationBrowse(
    state: LibraryState,
    coverUrl: (BookSummary) -> String?,
    seriesCoverUrl: (SeriesSummary) -> String?,
    authorCoverUrl: (AuthorSummary) -> String?,
    collectionCoverUrl: (PublicationCollection) -> String?,
    authorizationHeader: String?,
    onGrouping: (PublicationGrouping) -> Unit,
    onSort: (String) -> Unit,
    onLayout: (PublicationLayout) -> Unit,
    searchExpanded: Boolean,
    onToggleSearch: () -> Unit,
    onOpenSeries: (SeriesSummary) -> Unit,
    onOpenAuthor: (AuthorSummary) -> Unit,
    onOpenCollection: (net.kdhmedia.mochil.catalogue.PublicationCollection) -> Unit,
    onCloseCollection: () -> Unit,
    onRetrySeries: () -> Unit,
    onReadBook: (BookSummary) -> Unit,
    downloadQueue: StateFlow<List<DownloadQueueItem>>,
    onSearch: (String) -> Unit,
    onBucket: (String) -> Unit,
    onDownloadedOnly: (Boolean) -> Unit,
    collectionActions: PublicationCollectionActions,
    onLoadMore: () -> Unit,
    onRetry: () -> Unit,
) {
    val books = rememberVisibleBooks(state)
    Column(Modifier.fillMaxSize()) {
        PublicationControls(state, onGrouping, onSort, onLayout, searchExpanded, state.query.isNotBlank() || state.bucket.isNotBlank() || state.downloadedOnly) {
            onToggleSearch()
        }
        MotionVisibility(searchExpanded && state.grouping != PublicationGrouping.Collections) {
            Column {
                PublicationSearch(state, onSearch) { if (searchExpanded) onToggleSearch() }
                if (state.grouping !in setOf(PublicationGrouping.Continue, PublicationGrouping.Collections, PublicationGrouping.Recent)) AlphabetRail(state.bucket, onBucket)
                Row(Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 4.dp)) {
                    FilterChip(
                        selected = state.downloadedOnly,
                        onClick = { onDownloadedOnly(!state.downloadedOnly) },
                        label = { Text("Downloaded") },
                    )
                }
            }
        }
        OpeningBanner(state, downloadQueue)
        if (state.openError != null) {
            Text(state.openError, modifier = Modifier.padding(horizontal = 20.dp, vertical = 6.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
        }
        Box(Modifier.fillMaxSize()) { when {
            state.grouping == PublicationGrouping.Series && state.series.isEmpty() -> PublicationEmpty("No series were found on this shelf.")
            state.grouping == PublicationGrouping.Authors && state.authors.isEmpty() -> PublicationEmpty("No authors were found on this shelf.")
            publicationGroupingUsesBookEmptyState(state.grouping) && books.isEmpty() -> PublicationEmpty(if (state.downloadedOnly) "No downloaded books on this shelf." else when (state.grouping) { PublicationGrouping.Continue -> "Books you start will appear here."; PublicationGrouping.Recent -> "No recently added books were found."; else -> "This shelf has no books." })
            state.grouping == PublicationGrouping.Series && state.currentLayout() == PublicationLayout.Covers -> SeriesGrid(state, seriesCoverUrl, authorizationHeader, onOpenSeries, collectionActions.series, onLoadMore, onRetry)
            state.grouping == PublicationGrouping.Series -> SeriesList(state, coverUrl, authorizationHeader, onOpenSeries, onReadBook, collectionActions, onLoadMore, onRetry, onRetrySeries)
            state.grouping == PublicationGrouping.Authors -> AuthorBrowse(state, authorCoverUrl, authorizationHeader, onOpenAuthor, onLoadMore, onRetry)
            state.grouping == PublicationGrouping.Collections -> PublicationCollections(state, coverUrl, collectionCoverUrl, authorizationHeader, onOpenCollection, onCloseCollection, onReadBook, collectionActions.book, collectionActions.rename, collectionActions.delete)
            state.currentLayout() == PublicationLayout.Covers -> BookGrid(books, coverUrl, authorizationHeader, if (state.downloadedOnly) null else state, onReadBook, collectionActions.book, onLoadMore, onRetry)
            else -> BookList(books, coverUrl, authorizationHeader, if (state.downloadedOnly) null else state, onReadBook, collectionActions.book, onLoadMore, onRetry)
        } }
    }
}

internal fun publicationGroupingUsesBookEmptyState(grouping: PublicationGrouping): Boolean =
    grouping !in setOf(PublicationGrouping.Series, PublicationGrouping.Authors, PublicationGrouping.Collections)

@Composable
private fun PublicationSearch(state: LibraryState, onSearch: (String) -> Unit, onDone: () -> Unit) {
    var draft by remember(state.selected?.id, state.grouping) { mutableStateOf(state.query) }
    val focusManager = LocalFocusManager.current
    OutlinedTextField(
        value = draft,
        onValueChange = { draft = it },
        modifier = Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 4.dp),
        singleLine = true,
        label = { Text(when (state.grouping) { PublicationGrouping.Series -> "Search series"; PublicationGrouping.Authors -> "Search authors"; else -> "Search books" }) },
        leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
        trailingIcon = if (draft.isNotEmpty()) {
            {
                IconButton(onClick = {
                    draft = ""
                    onSearch("")
                }) { Icon(Icons.Default.Close, contentDescription = "Clear search") }
            }
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
private fun AlphabetRail(selected: String, onSelect: (String) -> Unit) {
    LazyRow(
        contentPadding = PaddingValues(horizontal = 20.dp, vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        items(listOf("") + listOf("#") + ('A'..'Z').map(Char::toString), key = { if (it.isEmpty()) "all" else it }) { bucket ->
            FilterChip(
                selected = selected == bucket,
                onClick = { onSelect(bucket) },
                label = { Text(if (bucket.isEmpty()) "All" else bucket) },
            )
        }
    }
}

@Composable
private fun PublicationControls(state: LibraryState, onGrouping: (PublicationGrouping) -> Unit, onSort: (String) -> Unit, onLayout: (PublicationLayout) -> Unit, searchExpanded: Boolean, filterActive: Boolean, onToggleSearch: () -> Unit) {
    // One row only: the library switcher, then search and the View button. No tab row, so the covers start right below.
    var sheet by remember { mutableStateOf(false) }
    val quiet = MaterialTheme.colorScheme.onSurfaceVariant
    HeaderActions(listOfNotNull(
        if (state.grouping != PublicationGrouping.Collections) HeaderAction(
            Icons.Default.Search, if (searchExpanded) "Close search and filters" else "Search and filter", onToggleSearch,
            if (searchExpanded || filterActive) MaterialTheme.colorScheme.primary else quiet,
        ) else null,
        HeaderAction(Icons.Outlined.Tune, "View options", { sheet = true }, quiet),
    ))
    if (sheet) PublicationViewSheet(state, onGrouping, onSort, onLayout) { sheet = false }
}


internal fun visibleBooks(state: LibraryState): List<BookSummary> {
    if (!state.downloadedOnly) return state.books
    val query = state.query.trim().lowercase()
    return state.downloadedBooks.asSequence()
        .filter { it.libraryId == state.selected?.id }
        .filter { query.isEmpty() || it.title.lowercase().contains(query) || it.authors.any { author -> author.lowercase().contains(query) } }
        .filter { book ->
            state.bucket.isEmpty() || when (state.bucket) {
                "#" -> book.title.firstOrNull()?.isLetter() != true
                else -> book.title.trim().firstOrNull()?.uppercase() == state.bucket
            }
        }
        .toList()
}

/**
 * The shelf's books, filtered once per composition. The empty-state test and the grid/list branch each used to call
 * [visibleBooks] themselves, so an offline shelf allocated its filtered list and a lowercased title per book two or
 * three times on the main thread for one frame. Keyed on exactly what the filter reads.
 */
@Composable
internal fun rememberVisibleBooks(state: LibraryState): List<BookSummary> = remember(
    state.downloadedOnly,
    state.books,
    state.downloadedBooks,
    state.query,
    state.bucket,
    state.selected?.id,
) { visibleBooks(state) }
