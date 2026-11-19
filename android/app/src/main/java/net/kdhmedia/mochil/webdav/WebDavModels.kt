package net.kdhmedia.mochil.webdav

data class WebDavConnection(
    val id: String,
    val name: String,
    val rootUrl: String,
    val username: String,
    val password: String,
    val cacheGeneration: String = id,
)

/** Opaque cache generation; editing an address or credential invalidates every persisted derivative. */
internal fun webDavCredentialKey(connection: WebDavConnection): String = connection.cacheGeneration

data class WebDavEntry(
    val href: String,
    val name: String,
    val collection: Boolean,
    val contentLength: Long,
    val contentType: String,
    val etag: String,
    val modified: String,
) {
    val supported: Boolean get() = collection || extension in SUPPORTED_EXTENSIONS
    val extension: String get() = name.substringAfterLast('.', "").lowercase()
}

data class WebDavPage(val url: String, val entries: List<WebDavEntry>, val validator: String = "", val notModified: Boolean = false)

internal val SUPPORTED_EXTENSIONS = setOf(
    "epub", "pdf", "cbz", "cbr",
    "mp3", "m4a", "m4b", "aac", "flac", "ogg", "opus", "wav",
    "mp4", "m4v", "mkv", "webm", "mov",
)
