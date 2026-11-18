package net.kdhmedia.mochil.webdav

import android.app.Application
import android.content.Intent
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import java.io.File
import java.net.HttpURLConnection
import java.net.URI
import java.nio.file.StandardCopyOption
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.catalogue.PublicationOpener
import net.kdhmedia.mochil.offline.DownloadSpacePolicy
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.video.VideoPlayerActivity
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics

data class WebDavState(
    val connections: List<WebDavConnection> = emptyList(),
    val selectedId: String? = null,
    val pageUrl: String? = null,
    val entries: List<WebDavEntry> = emptyList(),
    val backStack: List<String> = emptyList(),
    val loading: Boolean = false,
    val openingHref: String? = null,
    val readyAudio: AudioTrack? = null,
    val indexedEntries: List<WebDavEntry> = emptyList(),
    val indexing: Boolean = false,
    val indexedFolders: Int = 0,
    val indexedMedia: Int = 0,
    val indexComplete: Boolean = true,
    val message: String? = null,
    val messageIsError: Boolean = false,
)

class WebDavViewModel(application: Application) : AndroidViewModel(application) {
    private val store = WebDavConnectionStore(application)
    private val opener = PublicationOpener(application)
    private val pageCache = WebDavPageCache(application)
    private val indexStore = WebDavIndexStore(application)
    private val streamReferences = WebDavStreamReferenceStore(application)
    private val mutableState = MutableStateFlow(WebDavState(connections = store.list()))
    private var loadJob: Job? = null
    private var indexJob: Job? = null
    val state = mutableState.asStateFlow()

    init { mutableState.value.connections.firstOrNull()?.let { select(it.id) } }

    fun add(name: String, url: String, username: String, password: String) {
        runCatching { store.put(name, url, username, password) }
            .onSuccess { value -> mutableState.value = mutableState.value.copy(connections = store.list(), message = null); select(value.id) }
            .onFailure { mutableState.value = mutableState.value.copy(message = it.message ?: "Could not save this WebDAV connection.", messageIsError = true) }
    }

    fun update(connection: WebDavConnection, name: String, url: String, username: String, password: String) {
        runCatching { store.update(connection.id, name, url, username, password) }
            .onSuccess { updated ->
                loadJob?.cancel(); indexJob?.cancel()
                viewModelScope.launch {
                    if (updated.rootUrl != connection.rootUrl || updated.username != connection.username || updated.password != connection.password) {
                        withContext(Dispatchers.IO) {
                            pageCache.removeConnection(connection.id)
                            indexStore.remove(connection.id)
                            streamReferences.removeConnection(connection.id)
                        }
                    }
                    mutableState.value = mutableState.value.copy(connections = store.list(), message = "${updated.name} updated.", messageIsError = false)
                    select(updated.id)
                }
            }
            .onFailure { mutableState.value = mutableState.value.copy(message = it.message ?: "Could not update this WebDAV connection.", messageIsError = true) }
    }

    fun remove(connection: WebDavConnection) {
        if (mutableState.value.selectedId == connection.id) { loadJob?.cancel(); indexJob?.cancel() }
        viewModelScope.launch {
            val removed = withContext(Dispatchers.IO) { runCatching { store.remove(connection.id) } }
            if (removed.isFailure) {
                mutableState.value = mutableState.value.copy(message = removed.exceptionOrNull()?.message ?: "Could not remove this WebDAV connection.", messageIsError = true)
                return@launch
            }
            val connections = withContext(Dispatchers.IO) {
                pageCache.removeConnection(connection.id); indexStore.remove(connection.id); streamReferences.removeConnection(connection.id); store.list()
            }
            mutableState.value = WebDavState(connections = connections, message = "${connection.name} removed. Downloaded media is unchanged.")
            connections.firstOrNull()?.let { select(it.id) }
        }
    }

    fun select(id: String) {
        val connection = store.list().firstOrNull { it.id == id } ?: return
        indexJob?.cancel()
        mutableState.value = mutableState.value.copy(indexedEntries = emptyList(), indexComplete = true, indexing = false, indexedFolders = 0, indexedMedia = 0)
        indexJob = viewModelScope.launch {
            val index = withContext(Dispatchers.IO) { indexStore.get(connection) }
            if (mutableState.value.selectedId == id) mutableState.value = mutableState.value.copy(indexedEntries = index?.entries.orEmpty(), indexComplete = index?.complete ?: true)
        }
        load(connection, connection.rootUrl, emptyList())
    }

    fun indexSelected() {
        val connection = selected() ?: return
        if (mutableState.value.indexing) return
        mutableState.value = mutableState.value.copy(indexing = true, indexedFolders = 0, indexedMedia = 0, message = null, messageIsError = false)
        indexJob = viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) {
                PerformanceDiagnostics.measure("network", "index_webdav_library") {
                    WebDavIndexer(pageCache).scan(connection) { folders, media ->
                        mutableState.value = mutableState.value.copy(indexedFolders = folders, indexedMedia = media)
                    }.also(indexStore::put)
                }
            } }.onSuccess { index -> if (mutableState.value.selectedId == connection.id) {
                mutableState.value = mutableState.value.copy(indexing = false, indexedEntries = index.entries, indexComplete = index.complete,
                    message = if (index.complete) "Indexed ${index.entries.size} media files." else "Saved the first ${index.entries.size} media files. This library reached the device index limit.", messageIsError = false)
            } }.onFailure { failure -> if (failure !is kotlinx.coroutines.CancellationException && mutableState.value.selectedId == connection.id) {
                PerformanceDiagnostics.record("network", webDavDiagnosticName(failure), outcome = "error")
                mutableState.value = mutableState.value.copy(indexing = false, message = failure.message ?: "Could not index this WebDAV library.", messageIsError = true)
            } }
        }
    }

    fun open(entry: WebDavEntry) {
        val connection = selected() ?: return
        if (entry.collection) return load(connection, entry.href, mutableState.value.backStack + (mutableState.value.pageUrl ?: connection.rootUrl))
        if (mutableState.value.openingHref != null) return
        mutableState.value = mutableState.value.copy(openingHref = entry.href, message = null)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { prepareMedia(connection, entry) } }
                .onSuccess { prepared ->
                    when (entry.extension) {
                        in PUBLICATION_EXTENSIONS -> {
                            val file = requireNotNull(prepared.file)
                            val type = entry.contentType.ifBlank { mediaType(entry.extension) }
                            getApplication<Application>().startActivity(
                                opener.prepareFile(file, type, entry.name.substringBeforeLast('.'), "webdav:${connection.id}:${webDavResourceId(entry.href, entry.etag)}", entry.extension)
                                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                            )
                            mutableState.value = mutableState.value.copy(openingHref = null)
                        }
                        in VIDEO_EXTENSIONS -> {
                            getApplication<Application>().startActivity(Intent(getApplication(), VideoPlayerActivity::class.java).apply {
                                putExtra(VideoPlayerActivity.EXTRA_LIBRARY_ID, "webdav:${connection.id}")
                                putExtra(VideoPlayerActivity.EXTRA_ITEM_ID, webDavResourceId(entry.href, entry.etag))
                                putExtra(VideoPlayerActivity.EXTRA_TITLE, entry.name.substringBeforeLast('.'))
                                putExtra(VideoPlayerActivity.EXTRA_URL, prepared.uri)
                                putExtra(VideoPlayerActivity.EXTRA_LOCAL, true)
                                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                            })
                            mutableState.value = mutableState.value.copy(openingHref = null)
                        }
                        else -> mutableState.value = mutableState.value.copy(
                            openingHref = null,
                            readyAudio = AudioTrack(prepared.uri, "webdav:${connection.id}", entry.name.substringBeforeLast('.'), "", connection.name, 0.0, entry.contentType.ifBlank { mediaType(entry.extension) }, libraryKind = "music"),
                        )
                    }
                }
                .onFailure { failure -> PerformanceDiagnostics.record("network", webDavDiagnosticName(failure), outcome = "error"); mutableState.value = mutableState.value.copy(openingHref = null, message = failure.message ?: "Could not download this publication.", messageIsError = true) }
        }
    }

    fun back(): Boolean {
        val current = mutableState.value
        if (current.backStack.isEmpty()) return false
        load(selected() ?: return false, current.backStack.last(), current.backStack.dropLast(1))
        return true
    }

    fun consumeReadyAudio() { mutableState.value = mutableState.value.copy(readyAudio = null) }

    private fun prepareMedia(connection: WebDavConnection, entry: WebDavEntry): PreparedWebDavMedia {
        if (entry.extension in PUBLICATION_EXTENSIONS) {
            val file = PerformanceDiagnostics.measure("network", "download_webdav_media") { download(connection, entry) }
            return PreparedWebDavMedia(file.toURI().toString(), file)
        }
        val rangeReady = PerformanceDiagnostics.measure("network", "stream_webdav_media") { WebDavClient(connection).supportsRanges(entry.href) }
        if (rangeReady) return PreparedWebDavMedia(streamReferences.put(connection, entry.href).toString())
        val file = PerformanceDiagnostics.measure("network", "download_webdav_media") { download(connection, entry) }
        return PreparedWebDavMedia(file.toURI().toString(), file)
    }

    private fun load(connection: WebDavConnection, url: String, stack: List<String>) {
        loadJob?.cancel()
        mutableState.value = mutableState.value.copy(selectedId = connection.id, pageUrl = url, entries = emptyList(), backStack = stack, loading = true, message = null)
        loadJob = viewModelScope.launch {
            val cached = withContext(Dispatchers.IO) { pageCache.get(connection, url) }
            if (mutableState.value.selectedId != connection.id) return@launch
            if (cached != null) mutableState.value = mutableState.value.copy(entries = cached.entries)
            try {
                val fetched = withContext(Dispatchers.IO) { PerformanceDiagnostics.measure("network", "load_webdav_folder") { WebDavClient(connection).list(url, cached?.validator.orEmpty()) } }
                val page = resolveWebDavPage(cached, fetched)
                if (!fetched.notModified) withContext(Dispatchers.IO) { pageCache.put(connection, page) }
                if (mutableState.value.selectedId == connection.id) mutableState.value = mutableState.value.copy(pageUrl = page.url, entries = page.entries, backStack = stack, loading = false)
            } catch (failure: Throwable) {
                if (failure is kotlinx.coroutines.CancellationException) throw failure
                PerformanceDiagnostics.record("network", webDavDiagnosticName(failure), outcome = "error")
                if (mutableState.value.selectedId == connection.id) mutableState.value = mutableState.value.copy(loading = false, message = if (cached != null) "Showing the last saved folder. ${failure.message ?: "WebDAV is unavailable."}" else failure.message ?: "Could not browse this WebDAV folder.", messageIsError = true)
            }
        }
    }

    private fun download(connection: WebDavConnection, entry: WebDavEntry): File {
        val root = validatedWebDavUrl(connection.rootUrl)
        var current = validatedWebDavUrl(entry.href)
        // Inside publications/: the readers accept only that root, so a WebDAV
        // download cached elsewhere opened a reader that closed on the spot.
        val directory = File(getApplication<Application>().filesDir, "publications/webdav/${connection.id}").apply { mkdirs() }
        val identity = webDavResourceId(entry.href, entry.etag)
        val target = File(directory, "$identity.${entry.extension}")
        if (target.isFile && target.length() > 0) return target
        repeat(4) { redirect ->
            if (!sameOrigin(root, current)) throw WebDavRequestException("redirect", "The download left the configured WebDAV server.")
            val request = current.toURL().openConnection() as HttpURLConnection
            request.instanceFollowRedirects = false; request.connectTimeout = 15_000; request.readTimeout = 60_000
            if (connection.username.isNotBlank()) request.setRequestProperty("Authorization", basicAuthorization(connection.username, connection.password))
            try {
                if (request.responseCode in 300..399) {
                    if (redirect >= 3) throw WebDavRequestException("redirect", "The WebDAV download returned too many redirects.")
                    current = validatedWebDavUrl(current.resolve(request.getHeaderField("Location") ?: error("The WebDAV server returned an invalid redirect.")).toString())
                    return@repeat
                }
                if (request.responseCode !in 200..299) throw webDavStatusFailure(request.responseCode)
                val temporary = File(directory, ".$identity.part")
                try {
                    val expected = request.contentLengthLong.takeIf { it > 0 } ?: entry.contentLength
                    DownloadSpacePolicy.requireCapacity(directory, expected, MAX_DOWNLOAD_BYTES)
                    request.inputStream.use { input -> temporary.outputStream().buffered().use { output -> DownloadSpacePolicy.copyBounded(input, output, directory, MAX_DOWNLOAD_BYTES) } }
                    require(temporary.length() > 0) { "The downloaded file is empty." }
                    runCatching { java.nio.file.Files.move(temporary.toPath(), target.toPath(), StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING) }
                        .recoverCatching { java.nio.file.Files.move(temporary.toPath(), target.toPath(), StandardCopyOption.REPLACE_EXISTING) }.getOrThrow()
                } finally { temporary.delete() }
                return target
            } finally { request.disconnect() }
        }
        error("The WebDAV download could not be completed.")
    }

    private fun selected() = mutableState.value.selectedId?.let { id -> store.list().firstOrNull { it.id == id } }

    private companion object {
        val PUBLICATION_EXTENSIONS = setOf("epub", "pdf", "cbz", "cbr")
        val VIDEO_EXTENSIONS = setOf("mp4", "m4v", "mkv", "webm", "mov")
        const val MAX_DOWNLOAD_BYTES = 8L * 1024 * 1024 * 1024
    }
}

private fun sameOrigin(left: URI, right: URI) = left.scheme.equals(right.scheme, true) && left.host.equals(right.host, true) && (if (left.port >= 0) left.port else 443) == (if (right.port >= 0) right.port else 443)
private fun mediaType(extension: String) = when (extension) {
    "epub" -> "application/epub+zip"; "pdf" -> "application/pdf"; "cbz" -> "application/vnd.comicbook+zip"; "cbr" -> "application/vnd.comicbook-rar"
    "mp3" -> "audio/mpeg"; "m4a", "m4b", "aac" -> "audio/mp4"; "flac" -> "audio/flac"; "ogg", "opus" -> "audio/ogg"; "wav" -> "audio/wav"
    "mp4", "m4v" -> "video/mp4"; "mkv" -> "video/x-matroska"; "webm" -> "video/webm"; "mov" -> "video/quicktime"
    else -> "application/octet-stream"
}

internal fun resolveWebDavPage(cached: WebDavPage?, fetched: WebDavPage): WebDavPage =
    if (fetched.notModified) cached ?: fetched.copy(notModified = false) else fetched

private data class PreparedWebDavMedia(val uri: String, val file: File? = null)
