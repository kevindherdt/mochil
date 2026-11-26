package net.kdhmedia.mochil.catalogue

import android.app.Application
import android.os.SystemClock
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.auth.DeviceSessionEvents

class LibraryViewModel @JvmOverloads constructor(
    application: Application,
    /**
     * Where a queued download comes from. Production goes through the opener; a test that watches
     * queue progress supplies its own, so it can exercise a tick and a completion without a server.
     */
    queueFetch: (suspend (BookSummary, (done: Long, total: Long) -> Unit) -> Result<Unit>)? = null,
) : AndroidViewModel(application) {
    private val navigation = LibraryNavigationStore(application)
    internal val credential = DeviceCredentialStore(application).load()
    private val catalogue = PublicationCatalogueRepository()
    private val collectionEdits = LibraryCollectionEdits(this, catalogue)
    internal val opener = PublicationOpener(application)
    internal val downloads = PublicationDownloadStore(application)
    private val visibility = LibraryVisibilityStore(application)
    private val contentRequests = LibraryRequestGeneration()
    // Offline fallback can temporarily force downloads; retain the user's independent choice.
    private var userDownloadedOnly = false
    private var offlineGrouping: PublicationGrouping? = null
    private var summaryRefreshInFlight = false
    private var summaryRetryPending = false
    internal val mutableState = MutableStateFlow(LibraryState())
    val state: StateFlow<LibraryState> = mutableState.asStateFlow()
    private val layoutListener = navigation.listenLayout { layout -> mutableState.value = mutableState.value.copy(seriesLayout = layout, booksLayout = layout) }
	private var lastLibraryRefreshAt = 0L

    init {
        refreshDownloadedBooks()
        refresh()
    }

    fun refresh() {
        val saved = credential ?: return update { copy(initialLoading = false, error = "This device is no longer connected.") }
        val generation = contentRequests.advanceShelf()
        contentRequests.advanceGroup()
        update { copy(initialLoading = true, error = null, books = emptyList(), series = emptyList(), authors = emptyList(), collections = emptyList(), selectedCollection = null, nextCursor = null) }
        viewModelScope.launch {
            runCatching { catalogue.libraries(saved) }
                .onSuccess { libraries ->
                    if (!contentRequests.isCurrentShelf(generation)) return@onSuccess
                    lastLibraryRefreshAt = SystemClock.elapsedRealtime()
                    navigation.saveLibraries(libraries)
                    val visibleLibraries = visibility.filter(libraries)
                    val selected = navigation.selected(visibleLibraries)
                    val grouping = navigation.grouping(visibility.deviceClass)
                    val sort = navigation.sort()
                    val seriesLayout = navigation.seriesLayout()
                    val booksLayout = navigation.booksLayout()
                    update { copy(allLibraries = libraries, libraries = visibleLibraries, selected = selected, grouping = grouping, sort = sort, seriesLayout = seriesLayout, booksLayout = booksLayout, initialLoading = false, downloadedOnly = if (offlineOnly) userDownloadedOnly else downloadedOnly, offlineOnly = false, offlineDiagnostic = null) }
                    if (selected?.isPublicationLibrary() == true) loadFirstPage(selected)
                }
                .onFailure { failure ->
                    if (!contentRequests.isCurrentShelf(generation)) return@onFailure
                    if (failure.isExpiredDeviceSession()) {
                        DeviceSessionEvents.expired()
                        return@onFailure update { copy(initialLoading = false, error = failure.userMessage()) }
                    }
                    val cached = navigation.cachedLibraries()
                    if (cached.isEmpty()) update { copy(initialLoading = false, error = failure.userMessage()) }
                    else {
                        val visibleLibraries = visibility.filter(cached)
                        val selected = navigation.selected(visibleLibraries)
                        val offlineBooks = downloads.list().map(DownloadedPublication::asBookSummary)
                        val hasOfflinePublications = selected?.isPublicationLibrary() == true && offlineBooks.any { it.libraryId == selected.id }
                        if (hasOfflinePublications && offlineGrouping == null) offlineGrouping = navigation.grouping(visibility.deviceClass)
                        update { copy(
                            allLibraries = cached, libraries = visibleLibraries, selected = selected, initialLoading = false, error = null, offlineOnly = true, offlineDiagnostic = failure.offlineDiagnostic(),
                            downloadedBooks = offlineBooks, downloadedOnly = hasOfflinePublications,
                            grouping = if (hasOfflinePublications) PublicationGrouping.Books else grouping,
                            // The downloaded list is local: a server-side status has nothing to say about it.
                            status = BookStatusFilter.All, loadingMore = false,
                        ) }
                    }
                }
        }
    }

    /** Refresh server-owned labels without allowing overlapping foreground requests. */
    fun refreshLibrarySummaries() = refreshLibrarySummaries(force = false)

    private fun refreshLibrarySummaries(force: Boolean) {
        val current = mutableState.value
        if (summaryRefreshInFlight) {
            if (force) summaryRetryPending = true
            return
        }
        if (!force && !current.offlineOnly && current.initialLoading) return
        if (!force && SystemClock.elapsedRealtime() - lastLibraryRefreshAt < 15_000L) return
        val saved = credential ?: return
        val shelfGeneration = contentRequests.captureShelf()
        val selectedId = current.selected?.id
        summaryRefreshInFlight = true
        lastLibraryRefreshAt = SystemClock.elapsedRealtime()
        viewModelScope.launch {
            try {
                runCatching { catalogue.libraries(saved, refresh = true) }
                    .onSuccess { refreshed ->
                        // A later shelf request owns its navigation and content.
                        val shelfStillOwned = contentRequests.isCurrentShelf(shelfGeneration)
                        val navigationChanged = !shelfStillOwned || mutableState.value.selected?.id != selectedId
                        navigation.saveLibraries(refreshed)
                        val visible = visibility.filter(refreshed)
                        val previousID = mutableState.value.selected?.id
                        val wasOffline = mutableState.value.offlineOnly
                        val restoredGrouping = if (wasOffline && !navigationChanged && !userDownloadedOnly) offlineGrouping else null
                        offlineGrouping = null
                        update {
                            withRefreshedLibraries(refreshed, visible).let {
                                if (wasOffline) it.copy(downloadedOnly = userDownloadedOnly, grouping = restoredGrouping ?: it.grouping) else it
                            }
                        }
                        val selected = mutableState.value.selected
                        val latest = mutableState.value
                        val detailOpen = latest.selectedBook != null || latest.selectedSeries != null || latest.selectedAuthor != null || latest.selectedCollection != null || latest.expandedSeries != null
                        if (((wasOffline && shelfStillOwned && !detailOpen) || previousID != selected?.id) && selected?.isPublicationLibrary() == true) loadFirstPage(selected)
                    }
                    .onFailure { failure ->
                        if (failure.isExpiredDeviceSession()) {
                            summaryRetryPending = false
                            DeviceSessionEvents.expired()
                        }
                        else if (mutableState.value.offlineOnly) update { copy(offlineDiagnostic = failure.offlineDiagnostic()) }
                    }
            } finally {
                summaryRefreshInFlight = false
                if (summaryRetryPending) {
                    summaryRetryPending = false
                    refreshLibrarySummaries(force = true)
                }
            }
        }
    }

    fun select(library: LibrarySummary) {
        if (mutableState.value.selected?.id == library.id) return
        contentRequests.advanceGroup()
        update {
            copy(
                selected = library, books = emptyList(), series = emptyList(), authors = emptyList(), selectedSeries = null, selectedAuthor = null, selectedBook = null,
                expandedSeries = null, seriesBooks = emptyList(), groupNextCursor = null, query = "", bucket = "", nextCursor = null, error = null,
                loadingMore = false, grouping = grouping.startingView(visibility.deviceClass.defaultGrouping()), status = BookStatusFilter.All,
            )
        }
        navigation.select(library.id)
        navigation.setGrouping(mutableState.value.grouping)
        if (mutableState.value.offlineOnly) {
            val available = downloads.list().map(DownloadedPublication::asBookSummary)
            update { copy(downloadedBooks = available, downloadedOnly = library.isPublicationLibrary() && available.any { it.libraryId == library.id }, grouping = if (library.isPublicationLibrary()) PublicationGrouping.Books else grouping) }
            return
        }
        if (library.isPublicationLibrary()) loadFirstPage(library)
    }

    fun setLibraryVisible(library: LibrarySummary, visible: Boolean) {
        visibility.setVisible(library, visible)
        val previous = mutableState.value.selected
        val libraries = visibility.filter(mutableState.value.allLibraries)
        val selected = previous?.takeIf { current -> libraries.any { it.id == current.id } } ?: navigation.selected(libraries)
        val selectionChanged = previous?.id != selected?.id
        if (selectionChanged) {
            contentRequests.advanceShelf()
            contentRequests.advanceGroup()
        }
        update { copy(libraries = libraries, selected = selected, books = if (selectionChanged) emptyList() else books, series = if (selectionChanged) emptyList() else series) }
        if (selectionChanged && selected != null) {
            navigation.select(selected.id)
            if (!mutableState.value.offlineOnly && selected.isPublicationLibrary()) loadFirstPage(selected)
        }
    }

    fun deviceClass(): ClientDeviceClass = visibility.deviceClass

    fun setGrouping(grouping: PublicationGrouping) {
        val library = mutableState.value.selected ?: return
        if (mutableState.value.grouping == grouping) return
        contentRequests.advanceGroup()
        update { copy(grouping = grouping, status = BookStatusFilter.All, selectedSeries = null, selectedAuthor = null, selectedCollection = null, selectedBook = null, expandedSeries = null, seriesBooks = emptyList(), groupNextCursor = null) }
        navigation.setGrouping(grouping)
        loadFirstPage(library)
    }

    /** Shows every book in one list ordered by title, author, series or when it was added. */
    fun setSort(sort: String) {
        val library = mutableState.value.selected ?: return
        if (mutableState.value.grouping == PublicationGrouping.Books && mutableState.value.sort == sort) return
        contentRequests.advanceGroup()
        update { copy(grouping = PublicationGrouping.Books, sort = sort, selectedSeries = null, selectedAuthor = null, selectedCollection = null, selectedBook = null, expandedSeries = null, seriesBooks = emptyList(), groupNextCursor = null) }
        navigation.setGrouping(PublicationGrouping.Books)
        navigation.setSort(sort)
        loadFirstPage(library)
    }

    fun setLayout(layout: PublicationLayout) {
        val grouping = mutableState.value.grouping
        update {
            // Collections keep the collection that is open: changing the layout must not empty what is on screen.
            if (grouping == PublicationGrouping.Collections) copy(seriesLayout = layout, booksLayout = layout)
            else if (grouping.isGrouped()) copy(seriesLayout = layout, booksLayout = layout, selectedSeries = null, selectedAuthor = null, expandedSeries = null, seriesBooks = emptyList(), groupError = null)
            else copy(seriesLayout = layout, booksLayout = layout, selectedSeries = null, selectedAuthor = null, expandedSeries = null, seriesBooks = emptyList(), groupError = null)
        }
        navigation.setLayout(grouping, layout)
    }

    /** The covers/list choice of the combined Collections place; the same one every Collections view uses. */
    fun setCollectionsLayout(layout: PublicationLayout) {
        update { copy(seriesLayout = layout, booksLayout = layout) }
        navigation.setLayout(PublicationGrouping.Collections, layout)
    }

    /** Flips covers/list from whatever is current at the moment of the tap, so a stale button can never re-apply the old choice. */
    fun toggleCollectionsLayout() = setCollectionsLayout(
        if (mutableState.value.seriesLayout == PublicationLayout.Covers) PublicationLayout.List else PublicationLayout.Covers,
    )

    fun deleteCollection(collection: PublicationCollection) = collectionEdits.delete(collection)

    fun renameCollection(collection: PublicationCollection, name: String) = collectionEdits.rename(collection, name)

    fun openSeries(series: SeriesSummary) {
        if (mutableState.value.currentLayout() == PublicationLayout.Covers) {
            update { copy(selectedSeries = series, selectedAuthor = null, expandedSeries = null, seriesBooks = emptyList(), groupNextCursor = null) }
            loadSeriesBooks(series)
            return
        }
        if (mutableState.value.expandedSeries == series.name) {
            contentRequests.advanceGroup()
            update { copy(expandedSeries = null, seriesBooks = emptyList(), groupNextCursor = null, groupError = null) }
        } else {
            update { copy(expandedSeries = series.name, selectedSeries = null, selectedAuthor = null, seriesBooks = emptyList(), groupNextCursor = null) }
            loadSeriesBooks(series)
        }
    }

    private var pendingCollectionOpen: String? = null

    /** Loads the collections of every library for the combined Collections place. */
    fun loadAllCollections() = collectionEdits.loadAll()

    /** Opens a collection from the combined place: goes into its library, to Collections, and opens it once the list is in. */
    fun openCollectionAnywhere(collection: PublicationCollection) {
        val library = mutableState.value.libraries.firstOrNull { it.id == collection.libraryId } ?: return
        pendingCollectionOpen = collection.id
        select(library)
        if (mutableState.value.grouping == PublicationGrouping.Collections) loadFirstPage(library) else setGrouping(PublicationGrouping.Collections)
    }

    fun openCollection(collection: PublicationCollection) {
        val saved = credential ?: return
        val generation = contentRequests.advanceGroup()
        update { copy(selectedCollection = collection, collectionFromAll = false, groupLoading = true, groupError = null, seriesBooks = emptyList()) }
        viewModelScope.launch {
            runCatching { catalogue.collectionBooks(saved, collection.id) }
                .onSuccess { books -> if (contentRequests.isCurrentGroup(generation)) update { copy(seriesBooks = books, groupLoading = false) } }
                .onFailure { failure -> if (contentRequests.isCurrentGroup(generation)) update { copy(groupLoading = false, groupError = failure.userMessage()) } }
        }
    }

    /** Reconciles a picker edit without closing or replacing the active Collections view. */
    fun refreshCollections(libraryId: String) {
        val saved = credential ?: return
        val current = mutableState.value
        if (!current.acceptsCollectionRefresh(libraryId)) return
        val selectedId = current.selectedCollection?.id
        val shelfGeneration = contentRequests.advanceShelf()
        val groupGeneration = if (selectedId != null) contentRequests.advanceGroup() else contentRequests.captureGroup()
        viewModelScope.launch {
            runCatching { catalogue.collectionSnapshot(saved, libraryId, selectedId) }.onSuccess { (collections, books) ->
                if (!contentRequests.isCurrentShelf(shelfGeneration) || !mutableState.value.acceptsCollectionRefresh(libraryId)) return@onSuccess
                if (selectedId != null && !contentRequests.isCurrentGroup(groupGeneration)) return@onSuccess
                val selected = selectedId?.let { id -> collections.firstOrNull { it.id == id } }
                // This refresh owns the shelf generation now, so a first page that was still in flight
                // will be dropped: the spinner it started has to be cleared here or it never is.
                update { copy(collections = collections, selectedCollection = selected, seriesBooks = books ?: seriesBooks, groupError = null, initialLoading = false) }
            }.onFailure { failure ->
                if (contentRequests.isCurrentShelf(shelfGeneration) && mutableState.value.acceptsCollectionRefresh(libraryId)) {
                    update { copy(groupError = if (selectedId != null) failure.userMessage() else groupError) }
                }
            }
        }
    }

    fun closeCollection() {
        contentRequests.advanceGroup()
        update { copy(selectedCollection = null, collectionFromAll = false, seriesBooks = emptyList(), groupLoading = false, groupError = null) }
    }

    fun closeSeries() {
        contentRequests.advanceGroup()
        update {
            copy(
                selectedSeries = null,
                expandedSeries = null,
                seriesBooks = emptyList(),
                groupNextCursor = null,
                groupError = null,
            )
        }
    }

    fun openAuthor(author: AuthorSummary) {
        update { copy(selectedAuthor = author, selectedSeries = null, seriesBooks = emptyList(), groupNextCursor = null) }
        loadAuthorBooks(author)
    }

    fun closeAuthor() { contentRequests.advanceGroup(); update { copy(selectedAuthor = null, seriesBooks = emptyList(), groupNextCursor = null, groupError = null) } }

    fun showBookDetails(book: BookSummary) = update { copy(selectedBook = book) }
    fun closeBook() = update { copy(selectedBook = null) }

    fun readBook(book: BookSummary) {
        val saved = credential ?: return update { copy(openError = "This device is no longer connected.") }
        if (mutableState.value.openingBookId != null) return
        update { copy(openingBookId = book.id, openingTitle = book.title, openingDone = 0, openingTotal = 0, openError = null) }
        viewModelScope.launch {
            var reported = 0L
            runCatching {
                opener.prepare(saved, book, nextBookInState(mutableState.value, book)) { done, total ->
                    // A few updates a second are plenty for a progress line.
                    val now = System.currentTimeMillis()
                    if (now - reported >= 250) { reported = now; update { copy(openingDone = done, openingTotal = total) } }
                }
            }
                .onSuccess { intent ->
                    refreshDownloadedBooks()
                    runCatching { getApplication<Application>().startActivity(intent) }
                        .onFailure { update { copy(openError = "No compatible reading app could open this ${book.format.uppercase()} file.") } }
                    update { copy(openingBookId = null) }
                }
                .onFailure { update { copy(openingBookId = null, openError = it.message ?: "Could not open this book.") } }
        }
    }

    /** Everything about the download queue lives in its own file; these are the entry points. */
    internal val downloadQueue = LibraryDownloads(this, queueFetch)

    fun search(query: String) {
        val library = mutableState.value.selected ?: return
        val normalized = query.trim()
        if (mutableState.value.query == normalized && mutableState.value.bucket.isEmpty()) return
        update { copy(query = normalized, bucket = "", selectedBook = null, selectedSeries = null, selectedAuthor = null, expandedSeries = null) }
        loadFirstPage(library)
    }

    /**
     * The status chip above the plain list of books. It is ignored where it cannot apply (a grouped view, the downloaded list,
     * offline), so a stale tap never filters a shelf the screen is not showing a chip for.
     */
    fun setStatus(status: BookStatusFilter) {
        val library = mutableState.value.selected ?: return
        val current = mutableState.value
        // All is always allowed: it is the way out of a filter whose chips are gone (offline, or a book open on a tablet).
        if (current.status == status || (status != BookStatusFilter.All && !current.showsStatusChips())) return
        update { copy(status = status, selectedBook = null, selectedSeries = null, selectedAuthor = null, expandedSeries = null) }
        loadFirstPage(library)
    }

    fun selectBucket(bucket: String) {
        val library = mutableState.value.selected ?: return
        val normalized = bucket.uppercase().takeIf { it == "#" || it.singleOrNull() in 'A'..'Z' }.orEmpty()
        val selected = toggledPublicationBucket(mutableState.value.bucket, normalized)
        update { copy(bucket = selected, selectedBook = null, selectedSeries = null, selectedAuthor = null, expandedSeries = null) }
        loadFirstPage(library)
    }

    fun setDownloadedOnly(enabled: Boolean) {
        userDownloadedOnly = enabled
        refreshDownloadedBooks()
        update {
            copy(
                downloadedOnly = enabled,
                // The downloaded list is local, so a server-side status has nothing to say about it.
                status = if (enabled) BookStatusFilter.All else status,
                grouping = if (enabled) PublicationGrouping.Books else grouping,
                selectedBook = null,
                selectedSeries = null, selectedAuthor = null,
            )
        }
    }

    fun retrySeriesBooks() {
        mutableState.value.selectedAuthor?.let { loadAuthorBooks(it); return }
        val name = mutableState.value.selectedSeries?.name ?: mutableState.value.expandedSeries ?: return
        val libraryId = mutableState.value.selected?.id ?: return
        loadSeriesBooks(SeriesSummary(libraryId, name, emptyList(), 0, null))
    }

    fun retry() {
        val current = mutableState.value
        if (current.offlineOnly) {
            refreshLibrarySummaries(force = true)
            return
        }
        when (libraryRetryTarget(current.selected != null, current.selectedSeries != null || current.selectedAuthor != null || current.expandedSeries != null)) {
            LibraryRetryTarget.Group -> retrySeriesBooks()
            LibraryRetryTarget.Shelf -> current.selected?.let(::loadFirstPage)
            LibraryRetryTarget.Discovery -> refresh()
        }
    }

    fun loadMore() {
        val current = mutableState.value
        val saved = credential ?: return
        val library = current.selected ?: return
        val cursor = current.nextCursor ?: return
        val generation = contentRequests.captureShelf()
        if (current.loadingMore || !library.isPublicationLibrary()) return
        update { copy(loadingMore = true, error = null) }
        viewModelScope.launch {
            if (current.grouping == PublicationGrouping.Series) {
                runCatching { catalogue.series(saved, library.id, cursor, current.query, current.bucket) }
                    .onSuccess { page -> if (contentRequests.isCurrentShelf(generation)) update { copy(series = series + page.items, nextCursor = page.nextCursor, loadingMore = false) } }
                    .onFailure { failure -> if (contentRequests.isCurrentShelf(generation)) update { copy(loadingMore = false, error = failure.userMessage()) } }
            } else if (current.grouping == PublicationGrouping.Authors) {
                runCatching { catalogue.authors(saved, library.id, cursor, current.query, current.bucket) }
                    .onSuccess { page -> if (contentRequests.isCurrentShelf(generation)) update { copy(authors = authors + page.items, nextCursor = page.nextCursor, loadingMore = false) } }
                    .onFailure { failure -> if (contentRequests.isCurrentShelf(generation)) update { copy(loadingMore = false, error = failure.userMessage()) } }
            } else {
                runCatching { catalogue.books(saved, library.id, cursor, current.query, current.bucket, current.bookSort(), current.status.wire) }
                    .onSuccess { page -> if (contentRequests.isCurrentShelf(generation)) update { copy(books = books + page.items, nextCursor = page.nextCursor, loadingMore = false) } }
                    .onFailure { failure -> if (contentRequests.isCurrentShelf(generation)) update { copy(loadingMore = false, error = failure.userMessage()) } }
            }
        }
    }

    fun loadMoreVisible() {
        if (mutableState.value.selectedSeries != null || mutableState.value.selectedAuthor != null || mutableState.value.expandedSeries != null) loadMoreGroup()
        else loadMore()
    }

    private fun loadMoreGroup() {
        val current = mutableState.value
        val saved = credential ?: return
        val cursor = current.groupNextCursor ?: return
        val generation = contentRequests.captureGroup()
        if (current.groupLoading || current.groupLoadingMore) return
        update { copy(groupLoadingMore = true, groupError = null) }
        viewModelScope.launch {
            runCatching {
                current.selectedAuthor?.let { catalogue.authorBooks(saved, it.libraryId, it.name, cursor) }
                    ?: (current.selectedSeries?.name ?: current.expandedSeries)?.let { name -> current.selected?.id?.let { id -> catalogue.seriesBooks(saved, id, name, cursor) } }
                    ?: error("Publication group is no longer open")
            }
                .onSuccess { page -> if (contentRequests.isCurrentGroup(generation)) update { copy(seriesBooks = seriesBooks + page.items, groupNextCursor = page.nextCursor, groupLoadingMore = false) } }
                .onFailure { failure -> if (contentRequests.isCurrentGroup(generation)) update { copy(groupLoadingMore = false, groupError = failure.userMessage()) } }
        }
    }


    private fun loadFirstPage(library: LibrarySummary) {
        val saved = credential ?: return
        val generation = contentRequests.advanceShelf()
        val grouping = mutableState.value.grouping
        val query = mutableState.value.query
        val bucket = mutableState.value.bucket
        // `loadingMore` too: a page still in flight is dropped by the generation check, and nothing else would clear its flag.
        update { copy(initialLoading = true, books = emptyList(), series = emptyList(), authors = emptyList(), collections = emptyList(), selectedCollection = null, nextCursor = null, error = null, loadingMore = false) }
        viewModelScope.launch {
            if (grouping == PublicationGrouping.Continue) {
                runCatching { catalogue.continuing(saved, library.id, query) }
                    .onSuccess { items -> if (contentRequests.isCurrentShelf(generation)) update { copy(books = items, nextCursor = null, initialLoading = false) } }
                    .onFailure { failure -> if (contentRequests.isCurrentShelf(generation)) update { copy(initialLoading = false, error = failure.userMessage()) } }
            } else if (grouping == PublicationGrouping.Series) {
                runCatching { catalogue.series(saved, library.id, null, query, bucket) }
                    .onSuccess { page -> if (contentRequests.isCurrentShelf(generation)) update { copy(series = page.items, nextCursor = page.nextCursor, initialLoading = false) } }
                    .onFailure { failure -> if (contentRequests.isCurrentShelf(generation)) update { copy(initialLoading = false, error = failure.userMessage()) } }
            } else if (grouping == PublicationGrouping.Authors) {
                runCatching { catalogue.authors(saved, library.id, null, query, bucket) }
                    .onSuccess { page -> if (contentRequests.isCurrentShelf(generation)) update { copy(authors = page.items, nextCursor = page.nextCursor, initialLoading = false) } }
                    .onFailure { failure -> if (contentRequests.isCurrentShelf(generation)) update { copy(initialLoading = false, error = failure.userMessage()) } }
            } else if (grouping == PublicationGrouping.Collections) {
                runCatching { catalogue.collections(saved, library.id) }
                    .onSuccess { items ->
                        if (!contentRequests.isCurrentShelf(generation)) return@onSuccess
                        update { copy(collections = items, nextCursor = null, initialLoading = false) }
                        pendingCollectionOpen?.let { id -> pendingCollectionOpen = null; items.firstOrNull { it.id == id }?.let { openCollection(it); update { copy(collectionFromAll = true) } } }
                    }
                    .onFailure { failure -> if (contentRequests.isCurrentShelf(generation)) update { copy(initialLoading = false, error = failure.userMessage()) } }
            } else {
                runCatching { catalogue.books(saved, library.id, null, query, bucket, mutableState.value.bookSort(), mutableState.value.status.wire) }
                    .onSuccess { page -> if (contentRequests.isCurrentShelf(generation)) update { copy(books = page.items, nextCursor = page.nextCursor, initialLoading = false) } }
                    .onFailure { failure -> if (contentRequests.isCurrentShelf(generation)) update { copy(initialLoading = false, error = failure.userMessage()) } }
            }
        }
    }

    private fun loadSeriesBooks(series: SeriesSummary) {
        val saved = credential ?: return
        val generation = contentRequests.advanceGroup()
        update { copy(groupLoading = true, groupLoadingMore = false, groupNextCursor = null, groupError = null, seriesBooks = emptyList()) }
        viewModelScope.launch {
            runCatching { catalogue.seriesBooks(saved, series.libraryId, series.name) }
                .onSuccess { page -> if (contentRequests.isCurrentGroup(generation)) update { copy(seriesBooks = page.items, groupNextCursor = page.nextCursor, groupLoading = false) } }
                .onFailure { failure -> if (contentRequests.isCurrentGroup(generation)) update { copy(groupLoading = false, groupError = failure.userMessage()) } }
        }
    }

    private fun loadAuthorBooks(author: AuthorSummary) {
        val saved = credential ?: return
        val generation = contentRequests.advanceGroup()
        update { copy(groupLoading = true, groupLoadingMore = false, groupNextCursor = null, groupError = null, seriesBooks = emptyList()) }
        viewModelScope.launch {
            runCatching { catalogue.authorBooks(saved, author.libraryId, author.name) }
                .onSuccess { page -> if (contentRequests.isCurrentGroup(generation)) update { copy(seriesBooks = page.items, groupNextCursor = page.nextCursor, groupLoading = false) } }
                .onFailure { failure -> if (contentRequests.isCurrentGroup(generation)) update { copy(groupLoading = false, groupError = failure.userMessage()) } }
        }
    }

    // Progress ticks from a download arrive on the IO thread while the screen updates state on the
    // main thread, so a plain read-then-write can drop one of them. `update` is a compare-and-set.
    fun seriesCoverUrl(series: SeriesSummary): String? = credential?.let { catalogue.seriesCoverUrl(it, series) }
    fun collectionCoverUrl(collection: PublicationCollection): String? = credential?.let { catalogue.collectionCoverUrl(it, collection) }
    fun authorCoverUrl(author: AuthorSummary): String? = credential?.let { catalogue.authorCoverUrl(it, author) }
    fun coverUrl(book: BookSummary): String? = credential?.let { catalogue.coverUrl(it, book) }
    fun authorizationHeader(): String? = credential?.let { "Bearer ${it.token}" }

    internal fun update(transform: LibraryState.() -> LibraryState) {
        mutableState.update { it.transform() }
    }



}
