package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CollectionModelsTest {
    @Test fun membershipUsesTheCompleteStablePublicationIdentity() {
        val dune = CollectionMember("book", "books", bookId = "dune")
        val collection = PublicationCollection("favorites", "Favorites", "books", 1, listOf(dune))
        assertTrue(collection.contains(dune))
        assertFalse(collection.contains(dune.copy(libraryId = "other")))
        assertFalse(collection.contains(CollectionMember("series", "books", series = "Dune")))
    }

    @Test fun collectionRefreshOnlyTargetsTheOpenOnlineLibrary() {
        val manga = LibrarySummary("manga", "Manga", "manga", 35)
        val active = LibraryState(selected = manga, grouping = PublicationGrouping.Collections)
        assertTrue(active.acceptsCollectionRefresh("manga"))
        assertFalse(active.acceptsCollectionRefresh("books"))
        assertFalse(active.copy(grouping = PublicationGrouping.Series).acceptsCollectionRefresh("manga"))
        assertFalse(active.copy(offlineOnly = true).acceptsCollectionRefresh("manga"))
    }
}
