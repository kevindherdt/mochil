package net.kdhmedia.mochil.catalogue

import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredential

/**
 * What the Collections place changes on the server: loading every library's collections, renaming one, deleting one. The view
 * model keeps the navigation (opening and closing a collection, which share its request generations); this is only the edits and
 * the reconciliation of the lists they leave behind. Split out of [LibraryViewModel] to keep it within its size budget.
 */
internal class LibraryCollectionEdits(
    private val vm: LibraryViewModel,
    private val catalogue: PublicationCatalogueRepository,
) {
    /** Loads the collections of every library for the combined Collections place. */
    fun loadAll() {
        val saved = vm.credential ?: return
        vm.update { copy(allCollectionsLoading = true, allCollectionsError = null) }
        vm.viewModelScope.launch {
            runCatching { catalogue.allCollections(saved) }
                .onSuccess { items -> vm.update { copy(allCollections = items, allCollectionsLoading = false) } }
                .onFailure { failure -> vm.update { copy(allCollectionsLoading = false, allCollectionsError = failure.userMessage()) } }
        }
    }

    fun delete(collection: PublicationCollection) = change(collection) { saved -> catalogue.deleteCollection(saved, collection.id) }

    fun rename(collection: PublicationCollection, name: String) {
        val clean = name.trim().take(80)
        if (clean.isBlank() || clean == collection.name) return
        change(collection) { saved -> catalogue.renameCollection(saved, collection.id, clean) }
    }

    private fun change(collection: PublicationCollection, change: suspend (DeviceCredential) -> Unit) {
        val saved = vm.credential ?: return
        vm.viewModelScope.launch {
            runCatching { change(saved); catalogue.allCollections(saved) to catalogue.collections(saved, collection.libraryId) }
                .onSuccess { (everything, inLibrary) ->
                    vm.update {
                        val sameLibrary = selected?.id == collection.libraryId
                        copy(
                            allCollections = everything,
                            collections = if (sameLibrary) inLibrary else collections,
                            selectedCollection = selectedCollection?.let { open -> inLibrary.firstOrNull { it.id == open.id } },
                            groupError = null,
                        )
                    }
                }
                .onFailure { failure -> vm.update { copy(allCollectionsError = failure.userMessage(), groupError = failure.userMessage()) } }
        }
    }
}
