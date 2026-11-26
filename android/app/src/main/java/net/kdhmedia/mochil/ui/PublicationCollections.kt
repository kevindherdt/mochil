package net.kdhmedia.mochil.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Surface
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.outlined.FolderOpen
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.grid.items as gridItems
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.runtime.Composable
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.PublicationCollection
import net.kdhmedia.mochil.catalogue.PublicationLayout
import net.kdhmedia.mochil.catalogue.currentLayout

@Composable
internal fun PublicationCollections(
    state: LibraryState,
    coverUrl: (BookSummary) -> String?,
    collectionCoverUrl: (PublicationCollection) -> String?,
    header: String?,
    onOpen: (PublicationCollection) -> Unit,
    onClose: () -> Unit,
    onRead: (BookSummary) -> Unit,
    onCollect: (BookSummary) -> Unit,
    onRename: (PublicationCollection, String) -> Unit = { _, _ -> },
    onDelete: (PublicationCollection) -> Unit = {},
) {
    val selected = state.selectedCollection
    if (selected == null) {
        if (state.collections.isEmpty()) return PublicationEmpty("No collections yet. Long-press a book or series to create one.")
        CollectionIndex(state.collections, state.currentLayout() == PublicationLayout.Covers, collectionCoverUrl, header, onOpen, onRename = onRename, onDelete = onDelete)
        return
    }
    BackHandler(onBack = onClose)
    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = onClose) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back to collections") }
            Text(selected.name, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
        when {
            state.groupLoading -> AccessibleLoadingIndicator("Loading collection", Modifier.fillMaxSize())
            state.groupError != null -> PublicationEmpty(state.groupError)
            state.seriesBooks.isEmpty() -> PublicationEmpty("This collection is empty.")
            state.currentLayout() == PublicationLayout.Covers -> BookGrid(state.seriesBooks, coverUrl, header, null, onRead, onCollect, {}, {}, kind = state.selected?.kind.orEmpty())
            else -> BookList(state.seriesBooks, coverUrl, header, null, onRead, onCollect, {}, {})
        }
    }
}
