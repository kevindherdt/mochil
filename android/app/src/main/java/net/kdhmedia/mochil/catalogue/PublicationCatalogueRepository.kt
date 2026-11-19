package net.kdhmedia.mochil.catalogue

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential

/** IO boundary for publication browsing; screen state and stale-request policy stay in the view model. */
internal class PublicationCatalogueRepository(
    private val catalogue: CatalogueClient = CatalogueClient(),
    private val collections: CollectionClient = CollectionClient(),
) {
    suspend fun libraries(credential: DeviceCredential, refresh: Boolean = false) = io {
        librarySummaries.load(credential, refresh) { catalogue.libraries(credential) }
    }
    suspend fun continuing(credential: DeviceCredential, libraryId: String, query: String) =
        io { catalogue.continuing(credential, libraryId, query) }
    suspend fun books(credential: DeviceCredential, libraryId: String, cursor: String?, query: String = "", bucket: String = "", sort: String = "") =
        io { catalogue.books(credential, libraryId, cursor, query, bucket, sort) }
    suspend fun seriesBooks(credential: DeviceCredential, libraryId: String, series: String, cursor: String? = null) =
        io { catalogue.books(credential, libraryId, series, cursor) }
    suspend fun authorBooks(credential: DeviceCredential, libraryId: String, author: String, cursor: String? = null) =
        io { catalogue.authorBooks(credential, libraryId, author, cursor) }
    suspend fun series(credential: DeviceCredential, libraryId: String, cursor: String?, query: String = "", bucket: String = "") =
        io { catalogue.series(credential, libraryId, cursor, query, bucket) }
    suspend fun authors(credential: DeviceCredential, libraryId: String, cursor: String?, query: String = "", bucket: String = "") =
        io { catalogue.authors(credential, libraryId, cursor, query, bucket) }
    suspend fun allCollections(credential: DeviceCredential) = io { collections.listAll(credential) }
    suspend fun collections(credential: DeviceCredential, libraryId: String) =
        io { collections.list(credential, libraryId) }
    suspend fun deleteCollection(credential: DeviceCredential, id: String) = io { collections.delete(credential, id) }
    suspend fun renameCollection(credential: DeviceCredential, id: String, name: String) = io { collections.rename(credential, id, name) }
    suspend fun collectionBooks(credential: DeviceCredential, collectionId: String) =
        io { collections.books(credential, collectionId) }
    suspend fun collectionSnapshot(credential: DeviceCredential, libraryId: String, selectedId: String?) = io {
        collections.list(credential, libraryId) to selectedId?.let { collections.books(credential, it) }
    }

    fun coverUrl(credential: DeviceCredential, book: BookSummary) = catalogue.coverUrl(credential, book)
    fun seriesCoverUrl(credential: DeviceCredential, series: SeriesSummary) = catalogue.seriesCoverUrl(credential, series)
    fun collectionCoverUrl(credential: DeviceCredential, collection: PublicationCollection) = catalogue.collectionCoverUrl(credential, collection)
    fun authorCoverUrl(credential: DeviceCredential, author: AuthorSummary) = catalogue.authorCoverUrl(credential, author)

    private suspend fun <T> io(block: () -> T): T = withContext(Dispatchers.IO) { block() }

    private companion object { val librarySummaries = LibrarySummaryRequestCache() }
}
