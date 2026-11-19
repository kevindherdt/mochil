package net.kdhmedia.mochil.reader.comic

import java.io.File
import java.io.ByteArrayOutputStream
import java.util.concurrent.ConcurrentHashMap
import java.util.zip.ZipEntry
import java.util.zip.ZipFile

/**
 * Decodes CBZ-style zip comic pages.
 *
 * Decompressed page bytes are kept in a small bounded per-archive LRU so repeated reads of the
 * same page (recomposition re-renders, scrolling back, pager prefetch revisiting a page) do not
 * re-read and re-inflate the same zip entry. Decoding is guarded by a lock *per page index*
 * rather than one lock for the whole archive, so a slow decode of one page can never stall the
 * pager's prefetch of a neighboring page; concurrent requests for the *same* page coalesce onto
 * a single decode instead of doing the work twice.
 */
class ComicArchive(
    file: File,
    private val maxCachedPages: Int = defaultMaxCachedPages,
    private val maxCacheBytes: Long = defaultMaxCacheBytes,
) : ComicPageSource {
    private val archive = ZipFile(file)
    private val pages: List<ZipEntry> = archive.entries().asSequence()
        .filter { !it.isDirectory && extension(it.name) in supportedExtensions && it.size in 1..maximumPageBytes }
        .sortedWith(compareBy(::naturalKey))
        .take(maximumPages)
        .toList()

    // Guards only the small bookkeeping below (map get/put/trim), never the zip decode itself,
    // so it can never serialize decoding of two different pages.
    private val cacheLock = Any()
    private val cache = LinkedHashMap<Int, ByteArray>(16, 0.75f, true)
    private var cacheBytes = 0L

    // One lock per page index: different pages decode fully concurrently; repeated requests for
    // the same page while it is already decoding wait for that single decode instead of redoing it.
    private val pageLocks = ConcurrentHashMap<Int, Any>()

    @Volatile
    var decodeCount: Int = 0
        private set

    override val pageCount: Int get() = pages.size

    override fun page(index: Int): ByteArray {
        cached(index)?.let { return it }
        val lock = pageLocks.getOrPut(index) { Any() }
        synchronized(lock) {
            cached(index)?.let { return it }
            val decoded = decode(index)
            store(index, decoded)
            return decoded
        }
    }

    override fun close() {
        archive.close()
        synchronized(cacheLock) {
            cache.clear()
            cacheBytes = 0L
        }
    }

    /** Test-only seam: lets a test hold a page's decode lock to prove other pages are not blocked by it. */
    internal fun lockForTesting(index: Int): Any = pageLocks.getOrPut(index) { Any() }

    private fun cached(index: Int): ByteArray? = synchronized(cacheLock) { cache[index] }

    private fun store(index: Int, data: ByteArray) {
        synchronized(cacheLock) {
            cache[index] = data
            cacheBytes += data.size.toLong()
            while (cache.size > 1 && (cache.size > maxCachedPages || cacheBytes > maxCacheBytes)) {
                val eldestKey = cache.keys.iterator().next()
                val removed = cache.remove(eldestKey)
                if (removed != null) cacheBytes -= removed.size.toLong()
            }
        }
    }

    private fun decode(index: Int): ByteArray {
        val entry = pages.getOrNull(index) ?: error("Comic page is unavailable.")
        if (entry.size !in 1..maximumPageBytes) error("Comic page is too large.")
        val bytes = archive.getInputStream(entry).use { input ->
            val output = ByteArrayOutputStream(entry.size.toInt())
            val buffer = ByteArray(32 * 1024)
            var total = 0L
            while (true) {
                val read = input.read(buffer)
                if (read < 0) break
                total += read
                if (total > maximumPageBytes) error("Comic page is too large.")
                output.write(buffer, 0, read)
            }
            output.toByteArray()
        }
        decodeCount++
        return bytes
    }

    private companion object {
        const val maximumPages = 20_000
        const val maximumPageBytes = 64L shl 20
        val supportedExtensions = setOf("jpg", "jpeg", "png", "webp", "gif", "avif")
        const val defaultMaxCachedPages = 6

        // Mirrors the standard Android bitmap-cache sizing convention (roughly 1/8th of the
        // app's heap), bounded so a single comic cannot blow a low-memory device's budget.
        val defaultMaxCacheBytes: Long = (Runtime.getRuntime().maxMemory() / 8).coerceIn(16L shl 20, 64L shl 20)

        fun extension(name: String) = name.substringAfterLast('.', "").lowercase()

        fun naturalKey(entry: ZipEntry): String = Regex("(\\d+|\\D+)").findAll(entry.name.lowercase()).joinToString("\u0000") { match ->
            match.value.toLongOrNull()?.toString()?.padStart(20, '0') ?: match.value
        }
    }
}
