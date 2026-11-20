package net.kdhmedia.mochil.local

import android.app.Application
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.provider.DocumentsContract
import android.media.MediaMetadataRetriever
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.video.VideoPlayerActivity

enum class LocalMediaKind { Publications, Audio, Video }
data class LocalMediaItem(val sourceUri: String, val documentUri: String, val name: String, val extension: String, val mimeType: String, val size: Long, val kind: LocalMediaKind,
    val lastModified: Long = 0, val embeddedTitle: String = "", val artist: String = "", val album: String = "", val durationSeconds: Double = 0.0) {
    val title get() = embeddedTitle.ifBlank { name.substringBeforeLast('.').ifBlank { name } }
    val stableId get() = documentUri.hashCode().toUInt().toString(16)
}
data class LocalFolderSource(val uri: String, val name: String, val mediaCount: Int = 0, val scanning: Boolean = false, val error: String? = null)
data class LocalSourceState(val folders: List<LocalFolderSource> = emptyList(), val items: List<LocalMediaItem> = emptyList())

class LocalSourceViewModel(application: Application) : AndroidViewModel(application) {
    private val folderStore = LocalFolderStore(application)
    private val indexStore = LocalMediaIndexStore(application)
    private val sourceUris = folderStore.list()
    private val mutableState = MutableStateFlow(LocalSourceState(sourceUris.map { uri -> LocalFolderSource(uri, folderName(Uri.parse(uri)), scanning = true) }))
    private val publicationOpener = LocalPublicationOpener(application)
    val state = mutableState.asStateFlow()

    init {
        viewModelScope.launch {
            val cachedItems = withContext(Dispatchers.IO) { indexStore.all(sourceUris) }
            mutableState.value = LocalSourceState(
                sourceUris.map { uri -> LocalFolderSource(uri, folderName(Uri.parse(uri)), cachedItems.count { it.sourceUri == uri }, scanning = true) },
                cachedItems,
            )
            sourceUris.forEach(::scan)
        }
    }

    fun add(uri: Uri?) {
        if (uri == null) return
        runCatching { getApplication<Application>().contentResolver.takePersistableUriPermission(uri, Intent.FLAG_GRANT_READ_URI_PERMISSION) }
        folderStore.add(uri.toString())
        val source = LocalFolderSource(uri.toString(), folderName(uri), scanning = true)
        mutableState.value = mutableState.value.copy(folders = (mutableState.value.folders.filterNot { it.uri == source.uri } + source).sortedBy { it.name.lowercase() })
        scan(source.uri)
    }

    fun remove(source: LocalFolderSource) {
        folderStore.remove(source.uri)
        viewModelScope.launch(Dispatchers.IO) { indexStore.remove(source.uri) }
        runCatching { getApplication<Application>().contentResolver.releasePersistableUriPermission(Uri.parse(source.uri), Intent.FLAG_GRANT_READ_URI_PERMISSION) }
        mutableState.value = mutableState.value.copy(folders = mutableState.value.folders.filterNot { it.uri == source.uri }, items = mutableState.value.items.filterNot { it.sourceUri == source.uri })
    }

    fun refresh(source: LocalFolderSource) = scan(source.uri)

    fun openPublication(item: LocalMediaItem) {
        viewModelScope.launch {
            runCatching { publicationOpener.prepare(item) }
                .onSuccess { getApplication<Application>().startActivity(it) }
                .onFailure { failure -> updateFolderError(item.sourceUri, failure.message ?: "Could not open this publication.") }
        }
    }

    fun openVideo(item: LocalMediaItem) {
        getApplication<Application>().startActivity(Intent(getApplication(), VideoPlayerActivity::class.java).apply {
            putExtra(VideoPlayerActivity.EXTRA_LIBRARY_ID, "local")
            putExtra(VideoPlayerActivity.EXTRA_ITEM_ID, item.stableId)
            putExtra(VideoPlayerActivity.EXTRA_TITLE, item.title)
            putExtra(VideoPlayerActivity.EXTRA_URL, item.documentUri)
            putExtra(VideoPlayerActivity.EXTRA_LOCAL, true)
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_GRANT_READ_URI_PERMISSION)
        })
    }

    private fun scan(encodedUri: String) {
        mutableState.value = mutableState.value.copy(folders = mutableState.value.folders.map { if (it.uri == encodedUri) it.copy(scanning = true, error = null) else it })
        viewModelScope.launch {
            val previous = mutableState.value.items.filter { it.sourceUri == encodedUri }
            val result = runCatching {
                withContext(Dispatchers.IO) {
                    discoverSupportedMedia(getApplication(), Uri.parse(encodedUri), previous).also { indexStore.replace(encodedUri, it) }
                }
            }
            val current = mutableState.value
            mutableState.value = current.copy(folders = current.folders.map { source ->
                if (source.uri != encodedUri) source
                else result.fold({ source.copy(mediaCount = it.size, scanning = false) }, { source.copy(scanning = false, error = "Could not read this folder.") })
            }, items = result.fold({ found -> (current.items.filterNot { it.sourceUri == encodedUri } + found).sortedBy { it.title.lowercase() } }, { current.items }))
        }
    }

    private fun updateFolderError(sourceUri: String, message: String) {
        mutableState.value = mutableState.value.copy(folders = mutableState.value.folders.map { if (it.uri == sourceUri) it.copy(error = message) else it })
    }
}

internal fun discoverSupportedMedia(context: Context, treeUri: Uri, previous: List<LocalMediaItem> = emptyList()): List<LocalMediaItem> {
    val rootId = DocumentsContract.getTreeDocumentId(treeUri)
    val pending = ArrayDeque<String>().apply { add(rootId) }
    val found = mutableListOf<LocalMediaItem>()
    var visited = 0
    val previousByUri = previous.associateBy(LocalMediaItem::documentUri)
    while (pending.isNotEmpty() && visited < MAX_DOCUMENTS) {
        val parentId = pending.removeFirst()
        val children = DocumentsContract.buildChildDocumentsUriUsingTree(treeUri, parentId)
        context.contentResolver.query(children, PROJECTION, null, null, null)?.use { cursor ->
            val idIndex = cursor.getColumnIndexOrThrow(DocumentsContract.Document.COLUMN_DOCUMENT_ID)
            val nameIndex = cursor.getColumnIndexOrThrow(DocumentsContract.Document.COLUMN_DISPLAY_NAME)
            val mimeIndex = cursor.getColumnIndexOrThrow(DocumentsContract.Document.COLUMN_MIME_TYPE)
            val sizeIndex = cursor.getColumnIndexOrThrow(DocumentsContract.Document.COLUMN_SIZE)
            val modifiedIndex = cursor.getColumnIndex(DocumentsContract.Document.COLUMN_LAST_MODIFIED)
            while (cursor.moveToNext() && visited < MAX_DOCUMENTS) {
                visited++
                val id = cursor.getString(idIndex)
                val mime = cursor.getString(mimeIndex).orEmpty()
                if (mime == DocumentsContract.Document.MIME_TYPE_DIR) pending.add(id)
                else {
                    val name = cursor.getString(nameIndex).orEmpty()
                    val extension = name.substringAfterLast('.', "").lowercase()
                    localMediaKind(extension)?.let { kind ->
                        val documentUri = DocumentsContract.buildDocumentUriUsingTree(treeUri, id).toString()
                        val size = cursor.getLong(sizeIndex).coerceAtLeast(0)
                        val modified = if (modifiedIndex >= 0) cursor.getLong(modifiedIndex).coerceAtLeast(0) else 0
                        val cached = previousByUri[documentUri]
                        found += if (cached != null && cached.size == size && cached.lastModified == modified) cached
                        else extractLocalMetadata(context, LocalMediaItem(treeUri.toString(), documentUri, name, extension, mime, size, kind, modified))
                    }
                }
            }
        }
    }
    return found
}

private class LocalFolderStore(context: Context) {
    private val preferences = context.getSharedPreferences("local_media_sources", Context.MODE_PRIVATE)
    fun list(): Set<String> = preferences.getStringSet("folders", emptySet()).orEmpty()
    fun add(uri: String) = preferences.edit().putStringSet("folders", list() + uri).apply()
    fun remove(uri: String) = preferences.edit().putStringSet("folders", list() - uri).apply()
}

private fun folderName(uri: Uri): String = runCatching {
    DocumentsContract.getTreeDocumentId(uri).substringAfterLast(':').substringAfterLast('/').ifBlank { "Media folder" }
}.getOrDefault("Media folder")

private val PROJECTION = arrayOf(
    DocumentsContract.Document.COLUMN_DOCUMENT_ID,
    DocumentsContract.Document.COLUMN_DISPLAY_NAME,
    DocumentsContract.Document.COLUMN_MIME_TYPE,
    DocumentsContract.Document.COLUMN_SIZE,
    DocumentsContract.Document.COLUMN_LAST_MODIFIED,
)
private const val MAX_DOCUMENTS = 50_000
internal fun localMediaKind(extension: String): LocalMediaKind? = when (extension.lowercase()) {
    "epub", "pdf", "cbz", "cbr" -> LocalMediaKind.Publications
    "mp3", "m4a", "m4b", "aac", "ogg", "oga", "opus", "flac", "wav", "wave", "weba" -> LocalMediaKind.Audio
    "mp4", "m4v", "mkv", "mov", "webm", "ogv" -> LocalMediaKind.Video
    else -> null
}

private fun extractLocalMetadata(context: Context, item: LocalMediaItem): LocalMediaItem {
    if (item.kind == LocalMediaKind.Publications) return item
    return runCatching {
        MediaMetadataRetriever().use { retriever ->
            retriever.setDataSource(context, Uri.parse(item.documentUri))
            item.copy(
                embeddedTitle = retriever.extractMetadata(MediaMetadataRetriever.METADATA_KEY_TITLE).orEmpty().trim().take(1_000),
                artist = retriever.extractMetadata(MediaMetadataRetriever.METADATA_KEY_ARTIST).orEmpty().trim().take(1_000),
                album = retriever.extractMetadata(MediaMetadataRetriever.METADATA_KEY_ALBUM).orEmpty().trim().take(1_000),
                durationSeconds = retriever.extractMetadata(MediaMetadataRetriever.METADATA_KEY_DURATION)?.toDoubleOrNull()?.div(1_000)?.takeIf { it.isFinite() && it >= 0 } ?: 0.0,
            )
        }
    }.getOrDefault(item)
}
