package net.kdhmedia.mochil.catalogue

import java.io.File
import java.nio.ByteBuffer
import java.security.MessageDigest

/**
 * The one place that decides which private file backs an offline publication. Used by both
 * [PublicationOpener] (catalogue books) and the OPDS view model.
 */
object PublicationFileNaming {
    enum class Source { CATALOGUE, OPDS }

    fun sanitizedTitle(title: String): String =
        title.replace(Regex("[^A-Za-z0-9._ -]"), "_").trim().take(80).ifBlank { "publication" }

    fun safeExtension(format: String): String =
        format.lowercase().takeIf { it.matches(Regex("[a-z0-9]{1,8}")) } ?: "bin"

    /** Collision-proof index key for items that have no id of their own: `u:` + full SHA-256 hex of [url]. */
    fun urlKey(url: String): String =
        "u:" + MessageDigest.getInstance("SHA-256").digest(url.toByteArray(Charsets.UTF_8)).joinToString("") { "%02x".format(it) }

    /**
     * `<readable title>-<128 bits of SHA-256>.<ext>`. The digest covers an unambiguous (length-prefixed)
     * encoding of the source kind, the library (for OPDS this includes the connection id) and the full
     * book/item [identity], so equal titles, equal ids in different libraries, and ids with equal
     * `String.hashCode()` all land on different files. [identity] must be the full id (the acquisition
     * URL for OPDS items without an id), never a hash of it.
     */
    fun fileName(source: Source, libraryId: String, identity: String, title: String, extension: String): String {
        val digest = MessageDigest.getInstance("SHA-256")
        listOf("mochil-publication-v1", source.name, libraryId, identity).forEach { part ->
            val bytes = part.toByteArray(Charsets.UTF_8)
            digest.update(ByteBuffer.allocate(4).putInt(bytes.size).array())
            digest.update(bytes)
        }
        val hash = digest.digest().take(16).joinToString("") { "%02x".format(it) }
        return "${sanitizedTitle(title)}-$hash.${safeExtension(extension)}"
    }

    /**
     * Where [libraryId]/[bookId] lives on disk. An existing, non-empty file that the index records for
     * exactly this key always wins, so downloads made by older builds keep their stored path (nothing is
     * renamed, deleted or fetched again). Otherwise the file gets the collision-resistant name; a file
     * that merely sits at some legacy-style name but is not indexed for this key is never adopted.
     * [bookId] is the index key; [identity] is what the file name is derived from.
     */
    fun resolveTarget(
        store: PublicationDownloadStore,
        directory: File,
        source: Source,
        libraryId: String,
        bookId: String,
        title: String,
        extension: String,
        identity: String = bookId,
    ): File {
        store.find(libraryId, bookId)?.let { indexed ->
            val file = File(indexed.path)
            if (file.isFile && file.length() > 0) return file
        }
        return File(directory, fileName(source, libraryId, identity, title, extension))
    }
}
