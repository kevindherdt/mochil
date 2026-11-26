package net.kdhmedia.mochil.catalogue

data class LibraryState(
    val allLibraries: List<LibrarySummary> = emptyList(),
    val libraries: List<LibrarySummary> = emptyList(),
    val selected: LibrarySummary? = null,
    val grouping: PublicationGrouping = PublicationGrouping.Series,
    /** How the ungrouped list of all books is ordered: "" for title, or "author", "series", "recent". */
    val sort: String = "",
    val seriesLayout: PublicationLayout = PublicationLayout.Covers,
    val booksLayout: PublicationLayout = PublicationLayout.Covers,
    val query: String = "",
    val bucket: String = "",
    /** The reading-status chip over the plain list of books; All when no chip is chosen or the view has no such list. */
    val status: BookStatusFilter = BookStatusFilter.All,
    val books: List<BookSummary> = emptyList(),
    val series: List<SeriesSummary> = emptyList(),
    val authors: List<AuthorSummary> = emptyList(),
    val collections: List<PublicationCollection> = emptyList(),
    val allCollections: List<PublicationCollection> = emptyList(),
    val allCollectionsLoading: Boolean = false,
    /** True while the open collection was reached from the combined Collections place, so back returns there. */
    val collectionFromAll: Boolean = false,
    val allCollectionsError: String? = null,
    val selectedCollection: PublicationCollection? = null,
    val selectedSeries: SeriesSummary? = null,
    val selectedAuthor: AuthorSummary? = null,
    val selectedBook: BookSummary? = null,
    val expandedSeries: String? = null,
    val seriesBooks: List<BookSummary> = emptyList(),
    val initialLoading: Boolean = true,
    val loadingMore: Boolean = false,
    val groupLoading: Boolean = false,
    val groupLoadingMore: Boolean = false,
    val groupNextCursor: String? = null,
    val nextCursor: String? = null,
    val error: String? = null,
    val groupError: String? = null,
    val openingBookId: String? = null,
    val openingTitle: String = "",
    val openingDone: Long = 0,
    val openingTotal: Long = 0,
    val openError: String? = null,
    val downloadedOnly: Boolean = false,
    val downloadedBooks: List<BookSummary> = emptyList(),
    val offlineOnly: Boolean = false,
    val offlineDiagnostic: String? = null,
)

internal enum class LibraryRetryTarget { Discovery, Shelf, Group }

internal fun libraryRetryTarget(
    hasSelectedLibrary: Boolean,
    hasOpenGroup: Boolean,
): LibraryRetryTarget = when {
    hasOpenGroup -> LibraryRetryTarget.Group
    hasSelectedLibrary -> LibraryRetryTarget.Shelf
    else -> LibraryRetryTarget.Discovery
}

internal fun LibraryState.withRefreshedLibraries(
    all: List<LibrarySummary>,
    visible: List<LibrarySummary>,
): LibraryState {
    val refreshedSelection = selected?.let { old -> visible.firstOrNull { it.id == old.id } }
        ?: visible.firstOrNull { it.isPublicationLibrary() }
        ?: visible.firstOrNull()
    val changedShelf = selected?.id != refreshedSelection?.id
    return copy(
        allLibraries = all,
        libraries = visible,
        selected = refreshedSelection,
        offlineOnly = false,
        offlineDiagnostic = null,
        books = if (changedShelf) emptyList() else books,
        // A different library starts unfiltered, as if it had been chosen from the pills.
        status = if (changedShelf) BookStatusFilter.All else status,
        series = if (changedShelf) emptyList() else series,
        authors = if (changedShelf) emptyList() else authors,
    )
}

fun LibraryState.currentLayout(): PublicationLayout =
    if (grouping.usesGroupLayout()) seriesLayout else booksLayout

/** Series, Authors and Collections are all indexes of things, so they share one covers/list choice. */
fun PublicationGrouping.usesGroupLayout(): Boolean = isGrouped() || this == PublicationGrouping.Collections

fun PublicationGrouping.isGrouped(): Boolean =
    this == PublicationGrouping.Series || this == PublicationGrouping.Authors

/** The chips filter the plain list of books on the server, so they belong to that list online and nowhere else. */
fun LibraryState.showsStatusChips(keepDetailFilters: Boolean = false): Boolean =
    selected?.isPublicationLibrary() == true &&
        (grouping == PublicationGrouping.Books || grouping == PublicationGrouping.Recent) &&
        !downloadedOnly && !offlineOnly &&
        // On a wide screen the list stays on screen beside the open book, and the chips are the only sign of its filter.
        (keepDetailFilters || (selectedBook == null && selectedSeries == null && selectedAuthor == null))

internal fun LibraryState.bookSort(): String =
    when (grouping) { PublicationGrouping.Recent -> "recent"; PublicationGrouping.Books -> sort; else -> "" }

internal fun toggledPublicationBucket(current: String, selected: String): String =
    if (current == selected) "" else selected

fun LibrarySummary.isPublicationLibrary(): Boolean = kind.lowercase() in setOf(
    "books",
    "book",
    "lightnovels",
    "lightnovel",
    "comics",
    "comic",
    "manga",
    "mixed",
    "calibre",
)
