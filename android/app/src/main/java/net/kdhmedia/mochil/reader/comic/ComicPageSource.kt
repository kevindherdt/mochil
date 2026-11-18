package net.kdhmedia.mochil.reader.comic

import java.io.Closeable
import java.io.File

/**
 * A source of decoded comic page bytes. [ComicArchive] reads these from a fully downloaded local
 * CBZ; [StreamedComicPageSource] fetches each page on demand over HTTP (OPDS-PSE page streaming)
 * so the whole file never needs to be downloaded first. [ComicReaderScreen] only depends on this
 * interface, so paging, direction, tap zones and reading-position sync behave identically for
 * both: see [openComicPageSource].
 */
interface ComicPageSource : Closeable {
    val pageCount: Int
    fun page(index: Int): ByteArray
}

/** Chooses between a fully downloaded local archive and on-demand page streaming. */
sealed interface ComicSource {
    data class Local(val file: File) : ComicSource

    /** [pageStreamURL] is the server's `{pageNumber}` template, resolved against [serverUrl]. */
    data class Streamed(
        val serverUrl: String,
        val pageStreamURL: String,
        val pageCount: Int,
        val token: String?,
    ) : ComicSource
}

internal fun openComicPageSource(source: ComicSource): ComicPageSource = when (source) {
    is ComicSource.Local -> ComicArchive(source.file)
    is ComicSource.Streamed -> StreamedComicPageSource(source.serverUrl, source.pageStreamURL, source.pageCount, source.token)
}
