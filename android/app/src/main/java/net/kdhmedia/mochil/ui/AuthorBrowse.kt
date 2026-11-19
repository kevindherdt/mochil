package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.AuthorSummary
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.PublicationLayout
import net.kdhmedia.mochil.catalogue.currentLayout
import net.kdhmedia.mochil.catalogue.catalogueItemIdentityKey
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

@Composable
internal fun AuthorBrowse(state: LibraryState, artwork: (AuthorSummary) -> String?, header: String?, onOpen: (AuthorSummary) -> Unit, onMore: () -> Unit, onRetry: () -> Unit) {
    if (state.currentLayout() == PublicationLayout.Covers) {
        LazyVerticalGrid(
            columns = GridCells.Adaptive(136.dp),
            contentPadding = PaddingValues(20.dp),
            horizontalArrangement = Arrangement.spacedBy(14.dp),
            verticalArrangement = Arrangement.spacedBy(22.dp),
        ) {
            items(state.authors, key = { catalogueItemIdentityKey(it.libraryId, it.name) }) { author ->
                Column(Modifier.gridCard().clickable(onClickLabel = "Open ${author.name}") { onOpen(author) }.padding(10.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    PublicationArtwork(artwork(author), header, "Representative cover for ${author.name}", Modifier.fillMaxWidth().aspectRatio(.75f))
                    Text(author.name, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, minLines = 2, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    Text("${author.count} ${if (author.count == 1) "book" else "books"}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
            publicationPaging(state, onMore, onRetry)
        }
    } else {
        LazyColumn(Modifier.fillMaxSize()) {
            items(state.authors, key = { catalogueItemIdentityKey(it.libraryId, it.name) }) { author ->
                Row(
                    Modifier.homeCard().clickable(onClickLabel = "Open ${author.name}") { onOpen(author) }.padding(14.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(Modifier.weight(1f)) {
                        Text(author.name, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Medium)
                        Text("${author.count} ${if (author.count == 1) "book" else "books"}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
            publicationPaging(state, onMore, onRetry)
        }
    }
}

@Composable
internal fun AuthorDetail(author: AuthorSummary, state: LibraryState, coverUrl: (BookSummary) -> String?, header: String?, onBack: () -> Unit, onRetry: () -> Unit, onMore: () -> Unit, onRead: (BookSummary) -> Unit, collectionActions: PublicationCollectionActions) {
    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back to authors") }
            Column {
                Text(author.name, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
                Text("${author.count} ${if (author.count == 1) "book" else "books"}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        when {
            state.groupLoading -> Column(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) { if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }) else CircularProgressIndicator() }
            state.groupError != null && state.seriesBooks.isEmpty() -> Column(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) { Text(state.groupError, color = MaterialTheme.colorScheme.error); Button(onClick = onRetry) { Text("Try again") } }
            state.currentLayout() == PublicationLayout.Covers -> BookGrid(state.seriesBooks, coverUrl, header, state.groupPagingState(), onRead, collectionActions.book, onMore, onRetry)
            else -> BookList(state.seriesBooks, coverUrl, header, state.groupPagingState(), onRead, collectionActions.book, onMore, onRetry, table = true, showAuthors = false)
        }
    }
}
