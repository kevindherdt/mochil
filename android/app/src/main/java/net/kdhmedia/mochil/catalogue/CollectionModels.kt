package net.kdhmedia.mochil.catalogue

data class CollectionMember(
    val type: String,
    val libraryId: String,
    val bookId: String = "",
    val series: String = "",
)

data class PublicationCollection(
    val id: String,
    val name: String,
    val libraryId: String,
    val count: Int,
    val members: List<CollectionMember>,
    val covers: List<String> = emptyList(),
    /** A smart collection holds a search, not picked books; its contents cannot be edited by hand. */
    val smart: Boolean = false,
)

data class CollectionTarget(val label: String, val member: CollectionMember)

data class CollectionPickerState(
    val target: CollectionTarget? = null,
    val book: BookSummary? = null,
    val collections: List<PublicationCollection> = emptyList(),
    val collectionsEnabled: Boolean = true,
    val loading: Boolean = false,
    val busyId: String? = null,
    val creating: Boolean = false,
    val statusBusy: Boolean = false,
    val message: String? = null,
)

internal fun PublicationCollection.contains(member: CollectionMember): Boolean = members.any { it == member }

/** True when the book is in this collection only because its whole series was added. */
internal fun PublicationCollection.holdsThroughSeries(book: BookSummary?): Boolean =
    book != null && book.series.isNotBlank() && members.any { it.type == "series" && it.libraryId == book.libraryId && it.series == book.series }

internal fun LibraryState.acceptsCollectionRefresh(libraryId: String): Boolean =
    !offlineOnly && grouping == PublicationGrouping.Collections && selected?.id == libraryId
