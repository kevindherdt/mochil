package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ViewList
import androidx.compose.material.icons.filled.GridView
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.PublicationCollection
import net.kdhmedia.mochil.catalogue.PublicationLayout

/**
 * Every collection from every library in one place, each marked with the icon and colour of the library it belongs to.
 * Opening one goes into that library's collection, so reading, adding and removing work exactly as they do there.
 */
@Composable
internal fun AllCollectionsScreen(
    state: LibraryState,
    collectionCoverUrl: (PublicationCollection) -> String?,
    header: String?,
    onLoad: () -> Unit,
    onOpen: (PublicationCollection) -> Unit,
    onToggleLayout: () -> Unit,
    onRename: (PublicationCollection, String) -> Unit,
    onDelete: (PublicationCollection) -> Unit,
) {
    val covers = state.seriesLayout == PublicationLayout.Covers
    HeaderActions(listOf(
        HeaderAction(if (covers) Icons.AutoMirrored.Filled.ViewList else Icons.Default.GridView, if (covers) "Show as list" else "Show as covers", onToggleLayout, MaterialTheme.colorScheme.onSurfaceVariant),
    ))
    LaunchedEffect(Unit) { onLoad() }
    val libraries = state.libraries.associateBy { it.id }
    when {
        state.allCollections.isEmpty() && state.allCollectionsLoading -> AccessibleLoadingIndicator("Loading collections", Modifier.fillMaxSize())
        state.allCollections.isEmpty() && state.allCollectionsError != null -> InlineError(state.allCollectionsError, onLoad)
        state.allCollections.isEmpty() -> PublicationEmpty("No collections yet. Long-press a book or series to create one.")
        else -> Box(Modifier.fillMaxSize()) {
            CollectionIndex(
                state.allCollections,
                covers,
                collectionCoverUrl, header, onOpen,
                libraryOf = { libraries[it.libraryId] },
                onRename = onRename, onDelete = onDelete,
            )
        }
    }
}
