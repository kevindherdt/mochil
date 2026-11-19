package net.kdhmedia.mochil.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import net.kdhmedia.mochil.catalogue.LibraryViewModel

/** The Collections destination: it alone listens to the library state, so the app root does not recompose with it. */
@Composable
internal fun MobileCollectionsRoute(libraryViewModel: LibraryViewModel, onShowLibrary: () -> Unit) {
    val state by libraryViewModel.state.collectAsStateWithLifecycle()
    AllCollectionsScreen(
        state = state,
        collectionCoverUrl = libraryViewModel::collectionCoverUrl,
        header = libraryViewModel.authorizationHeader(),
        onLoad = libraryViewModel::loadAllCollections,
        onToggleLayout = libraryViewModel::toggleCollectionsLayout,
        onRename = libraryViewModel::renameCollection,
        onDelete = libraryViewModel::deleteCollection,
        onOpen = { collection -> libraryViewModel.openCollectionAnywhere(collection); onShowLibrary() },
    )
}
