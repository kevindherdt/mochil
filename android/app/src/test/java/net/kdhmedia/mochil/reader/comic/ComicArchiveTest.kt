package net.kdhmedia.mochil.reader.comic

import java.io.File
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.zip.ZipEntry
import java.util.zip.ZipOutputStream
import kotlin.concurrent.thread
import kotlin.system.measureNanoTime
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ComicArchiveTest {
    @Test
    fun pagesAreNaturallyOrderedAndNonImagesAreIgnored() {
        val file = File.createTempFile("comic", ".cbz")
        try {
            ZipOutputStream(file.outputStream()).use { archive ->
                listOf("page10.jpg" to byteArrayOf(10), "notes.txt" to byteArrayOf(99), "page2.jpg" to byteArrayOf(2)).forEach { (name, data) ->
                    archive.putNextEntry(ZipEntry(name))
                    archive.write(data)
                    archive.closeEntry()
                }
            }
            ComicArchive(file).use { comic ->
                assertEquals(2, comic.pageCount)
                assertArrayEquals(byteArrayOf(2), comic.page(0))
                assertArrayEquals(byteArrayOf(10), comic.page(1))
            }
        } finally {
            file.delete()
        }
    }

    @Test
    fun repeatedReadsOfTheSamePageDoNotReDecode() {
        withArchive(pageCount = 2) { archive ->
            repeat(5) { archive.page(0) }
            assertEquals("page 0 should decode only once across repeated reads", 1, archive.decodeCount)
            repeat(5) { archive.page(1) }
            assertEquals("page 1 should decode only once across repeated reads", 2, archive.decodeCount)
            // Composition re-renders revisit already-decoded pages; this must not re-decode.
            archive.page(0)
            assertEquals(2, archive.decodeCount)
        }
    }

    @Test
    fun cachedPageBytesAreStableAcrossReads() {
        withArchive(pageCount = 1) { archive ->
            val first = archive.page(0)
            val second = archive.page(0)
            assertArrayEquals(first, second)
        }
    }

    @Test
    fun cacheEvictsLeastRecentlyUsedPageWhenBoundedByCount() {
        val file = buildComic(pageCount = 3)
        try {
            ComicArchive(file, maxCachedPages = 2, maxCacheBytes = 1L shl 30).use { archive ->
                archive.page(0)
                archive.page(1)
                archive.page(2) // bounded to 2 entries: evicts page 0 (least recently used)
                assertEquals(3, archive.decodeCount)

                archive.page(0) // was evicted, must re-decode
                assertEquals(4, archive.decodeCount)

                archive.page(2) // was not evicted by the page-0 reload, must stay cached
                assertEquals(4, archive.decodeCount)
            }
        } finally {
            file.delete()
        }
    }

    @Test
    fun decodingOnePageDoesNotBlockDecodingAnotherPage() {
        val file = buildComic(pageCount = 2)
        try {
            ComicArchive(file).use { archive ->
                val holding = CountDownLatch(1)
                val release = CountDownLatch(1)
                val worker = thread {
                    synchronized(archive.lockForTesting(0)) {
                        holding.countDown()
                        release.await(2, TimeUnit.SECONDS)
                    }
                }
                assertTrue(holding.await(2, TimeUnit.SECONDS))
                val elapsedNanos = measureNanoTime { archive.page(1) }
                release.countDown()
                worker.join(2_000)
                assertTrue(
                    "decoding page 1 should not wait on page 0's lock, took ${elapsedNanos / 1_000_000}ms",
                    elapsedNanos < TimeUnit.MILLISECONDS.toNanos(300),
                )
            }
        } finally {
            file.delete()
        }
    }

    private fun buildComic(pageCount: Int): File {
        val file = File.createTempFile("comic", ".cbz")
        ZipOutputStream(file.outputStream()).use { archive ->
            repeat(pageCount) { index ->
                archive.putNextEntry(ZipEntry("page${index.toString().padStart(3, '0')}.jpg"))
                archive.write(byteArrayOf(index.toByte()))
                archive.closeEntry()
            }
        }
        return file
    }

    private fun withArchive(pageCount: Int, block: (ComicArchive) -> Unit) {
        val file = buildComic(pageCount)
        try {
            ComicArchive(file).use(block)
        } finally {
            file.delete()
        }
    }
}
