@file:android.annotation.SuppressLint("ApplySharedPref") // Persisted URI grants and their index change as one durable operation.
package net.kdhmedia.mochil.catalogue

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.provider.DocumentsContract
import java.io.File
import java.io.ByteArrayOutputStream

data class DownloadLocation(
    val treeUri: String? = null,
    val label: String = "App storage",
    val message: String? = null,
    val messageIsError: Boolean = false,
)

data class ExternalAudioFile(
    val uri: String,
    val displayName: String,
    val mediaType: String,
    val sizeBytes: Long,
)

class DownloadLocationStore(private val context: Context) {
    private val preferences = context.getSharedPreferences("download_location", Context.MODE_PRIVATE)

    fun current(): DownloadLocation {
        val value = preferences.getString(TREE_URI, null)?.takeIf(String::isNotBlank)
        return DownloadLocation(
            value,
            value?.let(::folderLabel) ?: "App storage",
            preferences.getString(MESSAGE, null),
            preferences.getBoolean(MESSAGE_IS_ERROR, false),
        )
    }

    fun select(uri: Uri): DownloadLocation {
        require(uri.scheme == "content") { "Choose a folder from Android's folder picker." }
        context.contentResolver.takePersistableUriPermission(uri, Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_GRANT_WRITE_URI_PERMISSION)
        preferences.edit().putString(TREE_URI, uri.toString()).remove(MESSAGE).remove(MESSAGE_IS_ERROR).commit()
        return current()
    }

    fun useAppStorage(): DownloadLocation {
        preferences.edit().remove(TREE_URI).remove(MESSAGE).remove(MESSAGE_IS_ERROR).commit()
        return current()
    }

    fun shouldMirror(existingUri: String?): Boolean {
        val selected = current().treeUri?.let(Uri::parse) ?: return false
        if (existingUri.isNullOrBlank()) return true
        return runCatching {
            DocumentsContract.getTreeDocumentId(selected) != DocumentsContract.getTreeDocumentId(Uri.parse(existingUri))
        }.getOrDefault(true)
    }

    fun mirror(source: File, displayName: String, mediaType: String): String? {
        val tree = current().treeUri?.let(Uri::parse) ?: return null
        return runCatching {
            val resolver = context.contentResolver
            val parent = DocumentsContract.buildDocumentUriUsingTree(tree, DocumentsContract.getTreeDocumentId(tree))
            val target = writeNewDownloadDocument(
                create = {
                    DocumentsContract.createDocument(resolver, parent, mediaType.ifBlank { "application/octet-stream" }, safeFileName(displayName))
                        ?: error("The selected folder refused the download.")
                },
                write = { document ->
                    resolver.openOutputStream(document, "w")?.use { output -> source.inputStream().use { it.copyTo(output) } }
                        ?: error("The selected folder could not be written.")
                },
                discard = { document -> DocumentsContract.deleteDocument(resolver, document) },
            )
            preferences.edit().remove(MESSAGE).remove(MESSAGE_IS_ERROR).apply()
            target.toString()
        }.onFailure {
            preferences.edit()
                .putString(MESSAGE, "Could not copy the latest download to the selected folder. The offline app copy is still available.")
                .putBoolean(MESSAGE_IS_ERROR, true)
                .apply()
        }.getOrNull()
    }

    fun delete(uri: String?): Boolean {
        if (uri.isNullOrBlank()) return true
        return runCatching {
            val candidate = Uri.parse(uri)
            context.contentResolver.delete(candidate, null, null) > 0 ||
                context.contentResolver.openFileDescriptor(candidate, "r")?.use { false } ?: true
        }.getOrDefault(false)
    }

    /** Persist a bounded, non-secret index beside user-visible downloads. */
    fun writeAudioIndex(value: String): Boolean {
        if (value.toByteArray().size > MAX_INDEX_BYTES) return false
        val tree = current().treeUri?.let(Uri::parse) ?: return false
        return runCatching {
            val resolver = context.contentResolver
            val parent = DocumentsContract.buildDocumentUriUsingTree(tree, DocumentsContract.getTreeDocumentId(tree))
            val target = findChild(tree, AUDIO_INDEX_NAME)
                ?: DocumentsContract.createDocument(resolver, parent, "application/json", AUDIO_INDEX_NAME)
                ?: error("The selected folder refused the download index.")
            resolver.openOutputStream(target, "wt")?.use { it.write(value.toByteArray()) }
                ?: error("The selected folder could not write the download index.")
            true
        }.getOrDefault(false)
    }

    fun readAudioIndex(): String? {
        val tree = current().treeUri?.let(Uri::parse) ?: return null
        val document = findChild(tree, AUDIO_INDEX_NAME) ?: return null
        return runCatching {
            context.contentResolver.openInputStream(document)?.use { input ->
                val output = ByteArrayOutputStream()
                val buffer = ByteArray(8192)
                var total = 0
                while (true) {
                    val read = input.read(buffer)
                    if (read < 0) break
                    total += read
                    require(total <= MAX_INDEX_BYTES) { "Download index is too large." }
                    output.write(buffer, 0, read)
                }
                output.toString(Charsets.UTF_8.name())
            }
        }.getOrNull()
    }

    /** Finds user-visible audio files when an uninstall removed Mochil' private index. */
    fun audioFiles(): List<ExternalAudioFile> {
        val tree = current().treeUri?.let(Uri::parse) ?: return emptyList()
        val resolver = context.contentResolver
        val children = DocumentsContract.buildChildDocumentsUriUsingTree(tree, DocumentsContract.getTreeDocumentId(tree))
        return runCatching {
            resolver.query(
                children,
                arrayOf(
                    DocumentsContract.Document.COLUMN_DOCUMENT_ID,
                    DocumentsContract.Document.COLUMN_DISPLAY_NAME,
                    DocumentsContract.Document.COLUMN_MIME_TYPE,
                    DocumentsContract.Document.COLUMN_SIZE,
                ),
                null,
                null,
                null,
            )?.use { cursor ->
                val id = cursor.getColumnIndexOrThrow(DocumentsContract.Document.COLUMN_DOCUMENT_ID)
                val name = cursor.getColumnIndexOrThrow(DocumentsContract.Document.COLUMN_DISPLAY_NAME)
                val type = cursor.getColumnIndexOrThrow(DocumentsContract.Document.COLUMN_MIME_TYPE)
                val size = cursor.getColumnIndexOrThrow(DocumentsContract.Document.COLUMN_SIZE)
                buildList {
                    while (cursor.moveToNext() && this.size < MAX_AUDIO_FILES) {
                        val displayName = cursor.getString(name).orEmpty()
                        val mediaType = cursor.getString(type).orEmpty()
                        if (displayName != AUDIO_INDEX_NAME && isAudioFile(displayName, mediaType)) add(
                            ExternalAudioFile(
                                DocumentsContract.buildDocumentUriUsingTree(tree, cursor.getString(id)).toString(),
                                displayName,
                                mediaType.ifBlank { mediaTypeForName(displayName) },
                                cursor.getLong(size).coerceAtLeast(0),
                            ),
                        )
                    }
                }
            }.orEmpty()
        }.getOrDefault(emptyList())
    }

    fun canRead(uri: String?): Boolean {
        if (!owns(uri)) return false
        return runCatching { context.contentResolver.openFileDescriptor(Uri.parse(uri), "r")?.use { true } ?: false }.getOrDefault(false)
    }

    fun owns(uri: String?): Boolean {
        if (uri.isNullOrBlank()) return false
        val tree = current().treeUri?.let(Uri::parse) ?: return false
        return runCatching {
            val candidate = Uri.parse(uri)
            candidate.scheme == tree.scheme && candidate.authority == tree.authority &&
                DocumentsContract.getDocumentId(candidate).substringBeforeLast('/') == DocumentsContract.getTreeDocumentId(tree).trimEnd('/')
        }.getOrDefault(false)
    }

    private fun findChild(tree: Uri, displayName: String): Uri? {
        val resolver = context.contentResolver
        val children = DocumentsContract.buildChildDocumentsUriUsingTree(tree, DocumentsContract.getTreeDocumentId(tree))
        return resolver.query(children, arrayOf(DocumentsContract.Document.COLUMN_DOCUMENT_ID, DocumentsContract.Document.COLUMN_DISPLAY_NAME), null, null, null)?.use { cursor ->
            val idColumn = cursor.getColumnIndexOrThrow(DocumentsContract.Document.COLUMN_DOCUMENT_ID)
            val nameColumn = cursor.getColumnIndexOrThrow(DocumentsContract.Document.COLUMN_DISPLAY_NAME)
            var result: Uri? = null
            while (cursor.moveToNext()) if (cursor.getString(nameColumn) == displayName) {
                result = DocumentsContract.buildDocumentUriUsingTree(tree, cursor.getString(idColumn))
                break
            }
            result
        }
    }

    private fun folderLabel(value: String): String = runCatching {
        DocumentsContract.getTreeDocumentId(Uri.parse(value)).substringAfter(':').ifBlank { "Selected folder" }
    }.getOrDefault("Selected folder")

    private fun safeFileName(value: String): String = value.replace(Regex("[\\p{Cntrl}/\\\\]"), "_").trim().take(120).ifBlank { "Mochil download" }

    private companion object {
        const val TREE_URI = "tree_uri"
        const val MESSAGE = "message"
        const val MESSAGE_IS_ERROR = "message_is_error"
        const val AUDIO_INDEX_NAME = ".mochil-audio-downloads.json"
        const val MAX_INDEX_BYTES = 2 * 1024 * 1024
        const val MAX_AUDIO_FILES = 10_000
        val AUDIO_EXTENSIONS = setOf("mp3", "m4a", "aac", "flac", "ogg", "opus", "wav", "webm")

        fun isAudioFile(name: String, mediaType: String): Boolean =
            mediaType.startsWith("audio/") || name.substringAfterLast('.', "").lowercase() in AUDIO_EXTENSIONS

        fun mediaTypeForName(name: String): String = when (name.substringAfterLast('.', "").lowercase()) {
            "m4a" -> "audio/mp4"
            "aac" -> "audio/aac"
            "flac" -> "audio/flac"
            "ogg", "opus" -> "audio/ogg"
            "wav" -> "audio/wav"
            "webm" -> "audio/webm"
            else -> "audio/mpeg"
        }
    }
}
