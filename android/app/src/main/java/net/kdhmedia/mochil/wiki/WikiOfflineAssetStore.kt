package net.kdhmedia.mochil.wiki

import android.content.Context
import android.net.Uri
import java.io.File
import java.security.MessageDigest

/** Bounded app-private image cache for explicitly mirrored vault pages. */
internal class WikiOfflineAssetStore(private val root: File) {
    constructor(context: Context) : this(File(context.applicationContext.noBackupFilesDir, "wiki_offline_assets"))

    fun mergeAndPrune(ownerId: String, libraryId: String, desired: List<WikiEntry>, assets: Map<WikiEntry, ByteArray>): Int = synchronized(storageLock) {
        clearUnowned(libraryId)
        val directory = directory(ownerId, libraryId)
        directory.mkdirs()
        val keep = boundedWikiAssetNames(desired.map { entry ->
            val name = fileName(entry)
            val incoming = assets[entry]?.takeIf { it.isNotEmpty() && it.size <= MAX_OFFLINE_ASSET_BYTES }
            val cached = File(directory, name)
            name to (incoming?.size?.toLong() ?: if (cached.isFile) cached.length() else 0L)
        })
        assets.forEach { (entry, bytes) ->
            if (fileName(entry) !in keep || bytes.isEmpty() || bytes.size > MAX_OFFLINE_ASSET_BYTES) return@forEach
            val target = File(directory, fileName(entry))
            val temporary = File.createTempFile("asset-", ".tmp", directory)
            try {
                temporary.outputStream().use { it.write(bytes) }
                try {
                    java.nio.file.Files.move(
                        temporary.toPath(),
                        target.toPath(),
                        java.nio.file.StandardCopyOption.ATOMIC_MOVE,
                        java.nio.file.StandardCopyOption.REPLACE_EXISTING,
                    )
                } catch (_: java.nio.file.AtomicMoveNotSupportedException) {
                    java.nio.file.Files.move(temporary.toPath(), target.toPath(), java.nio.file.StandardCopyOption.REPLACE_EXISTING)
                }
            } finally { temporary.delete() }
        }
        directory.listFiles().orEmpty().filter { it.isFile && it.name !in keep }.forEach(File::delete)
        directory.listFiles().orEmpty().count { it.isFile && it.name in keep }
    }

    fun cachedUrls(ownerId: String, libraryId: String, entries: List<WikiEntry>): Map<String, String> =
        cachedFiles(ownerId, libraryId, entries).mapValues { (_, file) -> Uri.fromFile(file).toString() }

    /** Revalidate on read too: interrupted or externally altered cache files are not trusted. */
    internal fun cachedFiles(ownerId: String, libraryId: String, entries: List<WikiEntry>): Map<String, File> = synchronized(storageLock) {
        clearUnowned(libraryId)
        val directory = directory(ownerId, libraryId)
        val available = directory.listFiles().orEmpty().filter(File::isFile).associateBy(File::getName)
        if (available.isEmpty()) return@synchronized emptyMap()
        val images = entries.filter { !it.isFolder && it.format.lowercase() in WIKI_IMAGE_FORMATS }
        val keep = boundedWikiAssetNames(images.map { fileName(it).let { name -> name to (available[name]?.length() ?: 0L) } })
        images.mapNotNull { entry ->
            val name = fileName(entry)
            available[name]?.takeIf { name in keep }?.let { entry.id to it }
        }.toMap()
    }

    /** Reuse only revision-addressed files; unknown revisions must be refreshed. */
    fun downloadTargets(ownerId: String, libraryId: String, entries: List<WikiEntry>): List<WikiEntry> = synchronized(storageLock) {
        val directory = directory(ownerId, libraryId)
        entries.filter { entry ->
            val cached = File(directory, fileName(entry))
            entry.revision.isBlank() || !cached.isFile || cached.length() !in 1L..MAX_OFFLINE_ASSET_BYTES.toLong()
        }
    }

    private fun directory(ownerId: String, libraryId: String) = File(root, wikiOfflineStorageIdentity(ownerId, libraryId))
    private fun clearUnowned(libraryId: String) { File(root, wikiLegacyStorageIdentity(libraryId)).deleteRecursively() }

    private companion object {
        // Activity recreation can overlap instances pointing at the same directory.
        // Pruning must never remove another instance's in-flight temporary file.
        val storageLock = Any()
    }
}

internal fun wikiOfflineAssetFileName(entry: WikiEntry): String = sha256("${entry.id}\u0000${entry.revision}\u0000${entry.path}") + ".bin"

private fun fileName(entry: WikiEntry) = wikiOfflineAssetFileName(entry)
private fun sha256(value: String): String {
    val digest = MessageDigest.getInstance("SHA-256").digest(value.encodeToByteArray())
    return buildString(digest.size * 2) {
        for (byte in digest) {
            val unsigned = byte.toInt() and 0xff
            append("0123456789abcdef"[unsigned ushr 4])
            append("0123456789abcdef"[unsigned and 15])
        }
    }
}

internal const val MAX_OFFLINE_ASSETS = 32
internal const val MAX_OFFLINE_ASSET_BYTES = 3 * 1024 * 1024
internal const val MAX_OFFLINE_ASSET_TOTAL_BYTES = 24 * 1024 * 1024
