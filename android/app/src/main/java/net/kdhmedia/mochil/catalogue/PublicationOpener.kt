package net.kdhmedia.mochil.catalogue

import android.content.Context
import android.content.Intent
import androidx.core.content.FileProvider
import java.io.File
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URI
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.reader.comic.ComicReaderActivity
import net.kdhmedia.mochil.reader.epub.EpubReaderActivity
import net.kdhmedia.mochil.reader.pdf.PdfReaderActivity
import net.kdhmedia.mochil.offline.DownloadSpacePolicy
import net.kdhmedia.mochil.offline.ResumableHttpDownload
import java.nio.file.StandardCopyOption

class PublicationOpener(
    private val context: Context,
    private val client: CatalogueClient = CatalogueClient(),
) {
    suspend fun prepare(
        credential: DeviceCredential,
        book: BookSummary,
        nextBook: BookSummary? = null,
        forceDownload: Boolean = false,
        onProgress: (done: Long, total: Long) -> Unit = { _, _ -> },
    ): Intent = withContext(Dispatchers.IO) {
        val streamURL = book.pageStreamURL
        if (!forceDownload && book.format.equals("cbz", true) && !streamURL.isNullOrBlank()) {
            // The server's page_stream_url only indicates that this book is streamable (it is the
            // same gate the web/OPDS client is bound by); it is never dereferenced directly, since
            // it points at the session/Basic-only /stream/ endpoint which rejects device tokens.
            // The device fetches pages through the Bearer-gated Android route instead.
            val route = client.publicationPageTemplate(book)
            return@withContext intentForStream(route, book.pageCount, book.title, "${book.libraryId}:${book.id}", book.libraryId, book.id).apply {
                if (nextBook != null) putExtra(ComicReaderActivity.EXTRA_NEXT_BOOK, nextBook)
            }
        }
        val directory = File(context.filesDir, "publications").apply { mkdirs() } // readers accept only this root
        val extension = PublicationFileNaming.safeExtension(book.format)
        val baseName = PublicationFileNaming.sanitizedTitle(book.title)
        val target = PublicationFileNaming.resolveTarget(
            PublicationDownloadStore(context), directory, PublicationFileNaming.Source.CATALOGUE, book.libraryId, book.id, book.title, extension,
        )
        // All opener instances share the same partial and resume files for this target.
        // Lock before checking the cache, so the waiter reuses the completed download.
        downloadLocks[(target.absolutePath.hashCode() and Int.MAX_VALUE) % downloadLocks.size].withLock {
            val newlyDownloaded = !target.isFile || target.length() == 0L
            if (newlyDownloaded) {
                val textOnly = book.format.equals("epub", true)
                download(credential, client.publicationUrl(credential, book, textOnly), target, if (textOnly) 0L else book.size, onProgress)
            }
            val downloadStore = PublicationDownloadStore(context)
            val previous = downloadStore.find(book.libraryId, book.id)
            val locationStore = DownloadLocationStore(context)
            val externalUri = if (newlyDownloaded || locationStore.shouldMirror(previous?.externalUri)) {
                locationStore.mirror(target, "$baseName.$extension", book.mediaType)
            } else previous?.externalUri
            if (!downloadStore.record(book, target, externalUri)) {
                // The bytes are on disk but the index would not take them, so the
                // download is not usable: say so instead of reporting success and
                // letting the book vanish from the offline shelf after a restart.
                throw IOException("The downloaded book could not be added to the offline library.")
            }
            if (externalUri != null && externalUri != previous?.externalUri) locationStore.delete(previous?.externalUri)
            intentFor(target, book.mediaType, book.title, "${book.libraryId}:${book.id}", book.format, book.libraryId, book.id).apply {
                if (nextBook != null && book.format.equals("cbz", true)) putExtra(ComicReaderActivity.EXTRA_NEXT_BOOK, nextBook)
            }
        }
    }

    fun prepare(download: DownloadedPublication): Intent =
        intentFor(File(download.path), download.mediaType, download.title, "${download.libraryId}:${download.bookId}", download.format, download.libraryId, download.bookId)

    fun prepareFile(file: File, mediaType: String, title: String, progressKey: String, format: String, externalTitleId: String? = null): Intent =
        intentFor(file, mediaType, title, progressKey, format).apply {
            if (!externalTitleId.isNullOrBlank()) putExtra(EpubReaderActivity.EXTRA_EXTERNAL_TITLE_ID, externalTitleId)
        }

    /** Builds a reader [Intent] that fetches comic pages on demand instead of opening a local file. */
    private fun intentForStream(
        pageStreamURL: String,
        pageCount: Int,
        title: String,
        progressKey: String,
        libraryId: String,
        bookId: String,
    ): Intent = Intent(context, ComicReaderActivity::class.java).apply {
        putExtra(ComicReaderActivity.EXTRA_STREAM_URL, pageStreamURL)
        putExtra(ComicReaderActivity.EXTRA_PAGE_COUNT, pageCount)
        putExtra(ComicReaderActivity.EXTRA_TITLE, title)
        putExtra(ComicReaderActivity.EXTRA_PROGRESS_KEY, progressKey)
        putExtra(ComicReaderActivity.EXTRA_LIBRARY_ID, libraryId)
        putExtra(ComicReaderActivity.EXTRA_BOOK_ID, bookId)
        addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    }

    private fun intentFor(
        file: File,
        mediaType: String,
        title: String,
        progressKey: String,
        format: String,
        libraryId: String = "",
        bookId: String = "",
    ): Intent {
        if (!file.isFile) error("This offline publication is no longer available.")
        if (format.equals("cbz", true)) {
            return Intent(context, ComicReaderActivity::class.java).apply {
                putExtra(ComicReaderActivity.EXTRA_PATH, file.absolutePath)
                putExtra(ComicReaderActivity.EXTRA_TITLE, title)
                putExtra(ComicReaderActivity.EXTRA_PROGRESS_KEY, progressKey)
                putExtra(ComicReaderActivity.EXTRA_LIBRARY_ID, libraryId)
                putExtra(ComicReaderActivity.EXTRA_BOOK_ID, bookId)
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
        }
        if (format.equals("epub", true)) {
            return Intent(context, EpubReaderActivity::class.java).apply {
                putExtra(EpubReaderActivity.EXTRA_PATH, file.absolutePath)
                putExtra(EpubReaderActivity.EXTRA_TITLE, title)
                putExtra(EpubReaderActivity.EXTRA_PROGRESS_KEY, progressKey)
                putExtra(EpubReaderActivity.EXTRA_LIBRARY_ID, libraryId)
                putExtra(EpubReaderActivity.EXTRA_BOOK_ID, bookId)
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
        }
        if (format.equals("pdf", true)) {
            return Intent(context, PdfReaderActivity::class.java).apply {
                putExtra(PdfReaderActivity.EXTRA_PATH, file.absolutePath)
                putExtra(PdfReaderActivity.EXTRA_TITLE, title)
                putExtra(PdfReaderActivity.EXTRA_PROGRESS_KEY, progressKey)
                putExtra(PdfReaderActivity.EXTRA_LIBRARY_ID, libraryId)
                putExtra(PdfReaderActivity.EXTRA_BOOK_ID, bookId)
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
        }
        val uri = FileProvider.getUriForFile(context, "${context.packageName}.files", file)
        return Intent(Intent.ACTION_VIEW).apply {
            setDataAndType(uri, mediaType.ifBlank { "application/octet-stream" })
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_ACTIVITY_NEW_TASK)
        }
    }

    private fun download(credential: DeviceCredential, initialUrl: String, target: File, catalogueSize: Long, onProgress: (Long, Long) -> Unit) {
        val origin = URI(credential.serverUrl)
        var current = URI(initialUrl)
        val temporary = File(target.parentFile, ".${target.name}.part")
        repeat(5) {
            val connection = current.toURL().openConnection() as HttpURLConnection
            connection.instanceFollowRedirects = false
            connection.connectTimeout = 15_000
            connection.readTimeout = 60_000
            if (sameOrigin(origin, current)) connection.setRequestProperty("Authorization", "Bearer ${credential.token}")
            val resume = ResumableHttpDownload.configure(connection, temporary)
            try {
                if (connection.responseCode in 300..399) {
                    val location = connection.getHeaderField("Location") ?: error("The server returned an invalid download redirect.")
                    current = current.resolve(location)
                    if (!current.scheme.equals("https", true)) error("The server returned an insecure download redirect.")
                    return@repeat
                }
                if (connection.responseCode == ResumableHttpDownload.HTTP_RANGE_NOT_SATISFIABLE && resume.offset > 0) {
                    ResumableHttpDownload.discard(temporary)
                    return@repeat
                }
                if (connection.responseCode !in 200..299) error("The server returned ${connection.responseCode} while opening this book.")
                val expected = connection.contentLengthLong.takeIf { it > 0 } ?: catalogueSize
                if (resume.offset == 0L) DownloadSpacePolicy.requireCapacity(requireNotNull(target.parentFile), expected, MAX_PUBLICATION_BYTES)
                ResumableHttpDownload.copyResponse(connection, temporary, requireNotNull(target.parentFile), MAX_PUBLICATION_BYTES, resume) { onProgress(it, expected) }
                runCatching { java.nio.file.Files.move(temporary.toPath(), target.toPath(), StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING) }
                    .recoverCatching { java.nio.file.Files.move(temporary.toPath(), target.toPath(), StandardCopyOption.REPLACE_EXISTING) }
                    .getOrElse { error("Could not finish caching the publication.") }
                ResumableHttpDownload.finish(temporary)
                return
            } finally {
                connection.disconnect()
            }
        }
        error("The server returned too many download redirects.")
    }

    private fun sameOrigin(left: URI, right: URI): Boolean =
        left.scheme.equals(right.scheme, true) && left.host.equals(right.host, true) && effectivePort(left) == effectivePort(right)

    private fun effectivePort(uri: URI): Int = when {
        uri.port >= 0 -> uri.port
        uri.scheme.equals("https", true) -> 443
        else -> 80
    }

    private companion object {
        const val MAX_PUBLICATION_BYTES = 8L * 1024 * 1024 * 1024
        val downloadLocks = Array(64) { Mutex() }
    }
}
