package net.kdhmedia.mochil.reader.epub

/**
 * Decides which in-flight EPUB search is still allowed to publish its result.
 *
 * A publication-wide search scans up to [net.kdhmedia.mochil.reader.epub.EpubSearch]'s result cap over the whole
 * book, so it takes seconds. The reader can dismiss the sheet or type another query inside that window; without a token
 * the finished scan overwrote the dismissed state and the sheet popped back open over the page being read.
 */
internal class EpubSearchGeneration {
    private var current = 0

    /** Claims the right to publish; every later call supersedes the previous one. */
    fun start(): Int = ++current

    fun isCurrent(token: Int): Boolean = token == current
}
