package net.kdhmedia.mochil.opds

import android.app.Application
import android.content.Intent
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import java.io.File
import java.net.HttpURLConnection
import java.net.URI
import java.nio.charset.StandardCharsets
import java.nio.file.StandardCopyOption
import java.util.Base64
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.isActive
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.catalogue.PublicationOpener
import net.kdhmedia.mochil.catalogue.PublicationFileNaming
import net.kdhmedia.mochil.catalogue.PublicationDownloadStore
import net.kdhmedia.mochil.catalogue.DownloadLocationStore
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.offline.DownloadSpacePolicy
import net.kdhmedia.mochil.source.SourceItem

data class OpdsState(
    val connections: List<OpdsConnection> = emptyList(),
    val selectedId: String? = null,
    val pageTitle: String = "",
    val currentUrl: String? = null,
    val items: List<SourceItem> = emptyList(),
    val backStack: List<String> = emptyList(),
    val nextUrl: String? = null,
    val searchAvailable: Boolean = false,
    val loading: Boolean = false,
    val openingIds: Set<String> = emptySet(),
    val artwork: Map<String, String> = emptyMap(),
    val message: String? = null,
)

class OpdsViewModel(
    application: Application,
    // The page fetch is a seam so a test can hold one page open and prove the newest navigation still wins.
    private val pageLoader: suspend (OpdsConnection, String?, String?) -> OpdsPage = { connection, url, query ->
        OpdsCatalogueSource(connection).let { source -> if (query == null) source.page(url) else source.searchPage(query, url) }
    },
    // Read-side seam for the same reason: navigating needs the connection list, and a test can supply one without the
    // device keystore the real store is encrypted with.
    private val connections: () -> List<OpdsConnection> = { OpdsConnectionStore(application).list() },
) : AndroidViewModel(application) {
    private val store by lazy { OpdsConnectionStore(application) }
    private val opener = PublicationOpener(application)
    private val artworkRepository = OpdsArtworkRepository(application)
    private val loadingArtwork = mutableSetOf<String>()
    private var loadJob: Job? = null
    private val mutableState = MutableStateFlow(OpdsState(connections = connections()))
    val state = mutableState.asStateFlow()

    init {
        connections().firstOrNull()?.let { select(it.id) }
    }

    fun add(name: String, url: String, username: String, password: String) {
        runCatching { store.put(name, url, username, password) }
            .onSuccess { connection ->
                mutableState.value = mutableState.value.copy(connections = store.list(), message = null)
                select(connection.id)
            }
            .onFailure { mutableState.value = mutableState.value.copy(message = it.message ?: "Could not save this OPDS catalogue.") }
    }

    fun update(connection: OpdsConnection, name: String, url: String, username: String, password: String) {
        runCatching { store.update(connection.id, name, url, username, password) }
            .onSuccess { updated ->
                mutableState.value = mutableState.value.copy(connections = store.list(), message = "${updated.name} updated.")
                select(updated.id)
            }
            .onFailure { mutableState.value = mutableState.value.copy(message = it.message ?: "Could not update this OPDS catalogue.") }
    }

    fun remove(connection: OpdsConnection) {
        runCatching { store.remove(connection.id) }
            .onSuccess { mutableState.value = OpdsState(connections = store.list(), message = "${connection.name} removed. Downloaded books are unchanged.") }
            .onFailure { mutableState.value = mutableState.value.copy(message = it.message ?: "Could not remove this OPDS catalogue.") }
    }

    fun select(id: String) = load(id, null, emptyList(), append = false)

    fun openFolder(item: SourceItem) {
        val target = item.navigationUrl ?: return
        val current = mutableState.value
        val currentUrl = current.currentUrl ?: selected()?.catalogueUrl ?: return
        load(current.selectedId ?: return, target, current.backStack + currentUrl, append = false)
    }

    fun back(): Boolean {
        val current = mutableState.value
        if (current.backStack.isEmpty()) return false
        load(current.selectedId ?: return false, current.backStack.last(), current.backStack.dropLast(1), append = false)
        return true
    }

    fun loadMore() {
        val current = mutableState.value
        load(current.selectedId ?: return, current.nextUrl ?: return, current.backStack, append = true)
    }

    fun search(query: String) {
        val connection = selected() ?: return
        if (query.isBlank()) return select(connection.id)
        load(connection.id, null, emptyList(), append = false, query = query.trim().take(200))
    }

    fun open(item: SourceItem) {
        val connection = selected() ?: return
        open(connection, item)
    }

    private fun open(connection: OpdsConnection, item: SourceItem) {
        val url = item.acquisitionUrl ?: return
        if (item.id in mutableState.value.openingIds) return
        mutableState.value = mutableState.value.copy(openingIds = mutableState.value.openingIds + item.id, message = null)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { download(connection, item, url) } }
                .onSuccess { file ->
                    val format = formatFor(item.mediaType, url)
                    val remoteID = item.id.ifBlank { item.acquisitionUrl.orEmpty() }
                    val intent = opener.prepareFile(file, item.mediaType, item.title, "opds:${connection.id}:$remoteID", format).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                    getApplication<Application>().startActivity(intent)
                    mutableState.value = mutableState.value.copy(openingIds = mutableState.value.openingIds - item.id)
                }
                .onFailure { failure -> mutableState.value = mutableState.value.copy(openingIds = mutableState.value.openingIds - item.id, message = failure.message ?: "Could not open this publication.") }
        }
    }


    fun requestArtwork(item: SourceItem) {
        val url = item.artworkUrl?.takeIf(String::isNotBlank) ?: return
        val connection = selected() ?: return
        if (url in mutableState.value.artwork || !loadingArtwork.add(url)) return
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { artworkRepository.load(connection, url) } }
                .onSuccess { file -> mutableState.value = mutableState.value.copy(artwork = mutableState.value.artwork + (url to file.toURI().toString())) }
            loadingArtwork.remove(url)
        }
    }

    private fun load(id: String, url: String?, stack: List<String>, append: Boolean, query: String? = null) {
        val connection = connections().firstOrNull { it.id == id } ?: return
        // The newest navigation wins. A page still arriving used to swallow the next tap outright, so on a slow feed a
        // folder row, the search button and the Back arrow all looked dead until the request finished (Back even
        // reported that it had handled the press). Cancel the older load and let the newer intent own the state.
        loadJob?.cancel()
        // Record where this load is going as it starts, not only when it lands: Back must be able to leave a folder
        // that is still loading instead of finding an empty stack and closing the whole OPDS screen.
        mutableState.value = mutableState.value.copy(selectedId = id, loading = true, message = null, backStack = stack)
        loadJob = viewModelScope.launch {
            val result = runCatching { withContext(Dispatchers.IO) { pageLoader(connection, url, query) } }
            // A cancelled load is not a failure: the state already belongs to whatever replaced it.
            if (!isActive) return@launch
            result.onSuccess { page -> mutableState.value = mutableState.value.copy(
                pageTitle = page.title, items = if (append) (mutableState.value.items + page.items).distinctBy(SourceItem::id) else page.items,
                currentUrl = if (append) mutableState.value.currentUrl else url ?: connection.catalogueUrl,
                backStack = stack, nextUrl = page.nextUrl, searchAvailable = page.searchAvailable, loading = false,
            ) }
                .onFailure { failure -> mutableState.value = mutableState.value.copy(loading = false, message = failure.message ?: "Could not load this OPDS catalogue.") }
        }
    }

    private fun selected() = mutableState.value.selectedId?.let { id -> connections().firstOrNull { it.id == id } }

    private fun download(connection: OpdsConnection, item: SourceItem, initialUrl: String): File {
        val origin = validatedHttps(connection.catalogueUrl)
        var current = validatedHttps(initialUrl)
        val cachedFormat = formatFor(item.mediaType, initialUrl)
        // Under publications/, because every reader (EPUB, PDF, comic) refuses a
        // file outside that root: cached here under files/opds, an OPDS download
        // opened its reader, which immediately finished without showing anything.
        val cachedDirectory = File(getApplication<Application>().filesDir, "publications/opds").apply { mkdirs() }
        val cachedTarget = resolveTarget(connection, item, cachedDirectory, cachedFormat)
        if (cachedTarget.isFile && cachedTarget.length() > 0) return recordDownload(connection, item, cachedTarget, cachedFormat)
        repeat(4) { redirect ->
            val request = current.toURL().openConnection() as HttpURLConnection
            request.instanceFollowRedirects = false; request.connectTimeout = 15_000; request.readTimeout = 60_000
            if (sameOpdsOrigin(origin, current) && connection.username.isNotBlank()) {
                val token = Base64.getEncoder().encodeToString("${connection.username}:${connection.password}".toByteArray(StandardCharsets.UTF_8))
                request.setRequestProperty("Authorization", "Basic $token")
            }
            try {
                if (request.responseCode in 300..399) {
                    require(redirect < 3) { "The publication returned too many redirects." }
                    current = validatedHttps(current.resolve(request.getHeaderField("Location") ?: error("The publication returned an invalid redirect.")).toString())
                    return@repeat
                }
                require(request.responseCode in 200..299) { "The publication server returned ${request.responseCode}." }
                val format = formatFor(item.mediaType, current.toString())
                val directory = cachedDirectory
                val target = resolveTarget(connection, item, directory, format)
                val temporary = File(directory, ".${target.name}.part")
                try {
                    DownloadSpacePolicy.requireCapacity(directory, request.contentLengthLong, MAX_PUBLICATION_BYTES)
                    request.inputStream.use { input -> temporary.outputStream().buffered().use { output -> DownloadSpacePolicy.copyBounded(input, output, directory, MAX_PUBLICATION_BYTES) } }
                    require(temporary.length() > 0) { "The downloaded publication is empty." }
                    runCatching { java.nio.file.Files.move(temporary.toPath(), target.toPath(), StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING) }
                        .recoverCatching { java.nio.file.Files.move(temporary.toPath(), target.toPath(), StandardCopyOption.REPLACE_EXISTING) }.getOrThrow()
                } finally { temporary.delete() }
                return recordDownload(connection, item, target, format)
            } finally { request.disconnect() }
        }
        error("The publication could not be downloaded.")
    }

    private fun resolveTarget(connection: OpdsConnection, item: SourceItem, directory: File, format: String): File =
        opdsTarget(PublicationDownloadStore(getApplication()), connection.id, item, directory, format)

    private fun recordDownload(connection: OpdsConnection, item: SourceItem, file: File, format: String): File =
        recordOpdsDownload(getApplication(), connection.id, item, file, format)
}

/** Where [item] lives (or will live) on disk for [connectionId]; the index key and the file identity both come from [item]. */
internal fun opdsTarget(store: PublicationDownloadStore, connectionId: String, item: SourceItem, directory: File, format: String): File =
    PublicationFileNaming.resolveTarget(
        store, directory, PublicationFileNaming.Source.OPDS, "opds:$connectionId",
        opdsBookId(item), item.title, format, identity = item.id.ifBlank { item.acquisitionUrl.orEmpty() },
    )

internal fun recordOpdsDownload(context: android.content.Context, connectionId: String, item: SourceItem, file: File, format: String): File {
    val libraryId = "opds:$connectionId"
    val bookId = opdsBookId(item)
    val book = BookSummary(bookId, libraryId, item.title, item.authors, "", null, format, item.mediaType, "", "", file.length())
    val downloads = PublicationDownloadStore(context)
    val previous = downloads.find(libraryId, bookId)
    val location = DownloadLocationStore(context)
    val externalUri = if (location.shouldMirror(previous?.externalUri)) location.mirror(file, "${item.title}.$format", item.mediaType) else previous?.externalUri
    if (externalUri != null && externalUri != previous?.externalUri) location.delete(previous?.externalUri)
    downloads.record(book, file, externalUri)
    return file
}

/**
 * The index key for [item]. Non-blank ids are used as-is, so existing entries still match.
 *
 * Blank ids used to be keyed by `acquisitionUrl.hashCode()`, which collides (e.g. "...Aa" and "...BB"), so two
 * different books on one connection shared an entry and the second was served the first's file. They now get
 * `u:` + the full SHA-256 of the URL ([PublicationFileNaming.urlKey]). There is deliberately NO fallback lookup
 * of the old hash key: that lookup is exactly the collision. Legacy blank-id entries therefore simply stop
 * matching and are downloaded once more under the new key; their old files and index rows are left untouched
 * on disk (never deleted or renamed here).
 */
internal fun opdsBookId(item: SourceItem): String = item.id.ifBlank { PublicationFileNaming.urlKey(item.acquisitionUrl.orEmpty()) }

internal fun formatFor(mediaType: String, url: String): String = when (mediaType.substringBefore(';').lowercase()) {
    "application/epub+zip" -> "epub"
    "application/pdf" -> "pdf"
    "application/vnd.comicbook+zip", "application/x-cbz" -> "cbz"
    else -> URI(url).path.substringAfterLast('.', "bin").lowercase().takeIf { it.matches(Regex("[a-z0-9]{1,8}")) } ?: "bin"
}
private fun sameOpdsOrigin(left: URI, right: URI) = left.scheme.equals(right.scheme, true) && left.host.equals(right.host, true) && (if (left.port >= 0) left.port else 443) == (if (right.port >= 0) right.port else 443)
private const val MAX_PUBLICATION_BYTES = 8L * 1024 * 1024 * 1024
