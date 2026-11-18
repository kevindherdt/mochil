package net.kdhmedia.mochil.wiki

import java.io.File
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URLEncoder
import java.security.MessageDigest
import java.nio.file.Files
import java.nio.file.StandardCopyOption
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.offline.DownloadSpacePolicy

/** Dedicated disposable PDF cache, never the Markdown/image offline store. */
internal class WikiPdfDownloads(
    private val directory: File,
    private val maximumBytes: Long = 64L * 1024 * 1024,
    private val maximumFiles: Int = 4,
    private val open: (DeviceCredential, String) -> HttpURLConnection = { credential, path ->
        openAndroidApiGet(credential, path, readTimeoutMillis = 30_000)
    },
) {
    init { require(maximumBytes > 0 && maximumFiles > 0) }

    suspend fun prepare(credential: DeviceCredential, libraryId: String, entry: WikiEntry): File = withContext(Dispatchers.IO) {
        writes.withLock {
            require(libraryId.isNotBlank() && entry.id.isNotBlank() && !entry.isFolder && entry.format.equals("pdf", true))
            if (!directory.isDirectory && !directory.mkdirs()) throw IOException("Could not prepare attachment storage.")
            // The process-wide writer lock means no other attachment instance
            // can own one of these partials. Surviving files are crash leftovers.
            directory.listFiles().orEmpty().filter { it.isFile && PART_FILE.matches(it.name) }.forEach {
                if (!it.delete()) throw IOException("Could not clean temporary attachment storage.")
            }
            val target = File(directory, "${wikiPdfIdentity(credential, libraryId, entry)}.pdf")
            if (entry.revision.isNotBlank() && validPdf(target) && target.length() <= maximumBytes) {
                target.setLastModified(System.currentTimeMillis())
                prune(target)
                return@withLock target
            }
            DownloadSpacePolicy.requireCapacity(directory, maximumBytes, maximumBytes)
            val temporary = File.createTempFile("wiki-pdf-", ".part", directory)
            try {
                val path = "/api/android/v1/document-assets/${segment(libraryId)}/${segment(entry.id)}/${segment(entry.path.substringAfterLast('/'))}"
                val connection = open(credential, path)
                try {
                    if (connection.responseCode !in 200..299) throw IOException("Could not download this PDF (${connection.responseCode}).")
                    if (!connection.contentType.orEmpty().substringBefore(';').trim().equals("application/pdf", true)) {
                        throw IOException("The server did not return a PDF attachment.")
                    }
                    val announced = connection.contentLengthLong
                    if (announced > maximumBytes) throw IOException("This PDF exceeds the attachment download limit.")
                    var total = 0L
                    connection.inputStream.use { input -> temporary.outputStream().use { output ->
                        val buffer = ByteArray(32 * 1024)
                        while (true) {
                            currentCoroutineContext().ensureActive()
                            val count = input.read(buffer)
                            if (count < 0) break
                            total += count
                            if (total > maximumBytes) throw IOException("This PDF exceeds the attachment download limit.")
                            output.write(buffer, 0, count)
                        }
                    } }
                    if (announced >= 0 && total != announced) throw IOException("The PDF download was incomplete.")
                    if (!validPdf(temporary)) throw IOException("This attachment does not contain a PDF header.")
                    currentCoroutineContext().ensureActive()
                    Files.move(temporary.toPath(), target.toPath(), StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING)
                    prune(target)
                    target
                } finally { connection.disconnect() }
            } finally { temporary.delete() }
        }
    }

    /** Evict only after a replacement exists; failed downloads must keep usable copies. */
    private fun prune(target: File) {
        directory.listFiles().orEmpty().filter { it.isFile && CACHE_FILE.matches(it.name) && it != target }
            .sortedByDescending(File::lastModified).drop(maximumFiles - 1).forEach {
                if (!it.delete()) throw IOException("Could not free attachment cache space.")
            }
    }

    private fun validPdf(file: File): Boolean = file.isFile && file.inputStream().use { input ->
        val prefix = ByteArray(5)
        input.read(prefix) == 5 && prefix.contentEquals("%PDF-".toByteArray(Charsets.US_ASCII))
    }

    private fun segment(value: String) = URLEncoder.encode(value, "UTF-8").replace("+", "%20")
    private companion object {
        val writes = Mutex()
        val CACHE_FILE = Regex("[a-f0-9]{64}\\.pdf")
        val PART_FILE = Regex("wiki-pdf--?[0-9]+\\.part")
    }
}

internal fun wikiPdfIdentity(credential: DeviceCredential, libraryId: String, entry: WikiEntry): String {
    val identity = listOf(credential.serverUrl, credential.deviceId, libraryId, entry.id, entry.revision)
        .joinToString("") { "${it.length}:$it" }
    return MessageDigest.getInstance("SHA-256").digest(identity.toByteArray(Charsets.UTF_8))
        .joinToString("") { "%02x".format(it.toInt() and 255) }
}
