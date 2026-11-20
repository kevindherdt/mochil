package net.kdhmedia.mochil.wiki

import java.io.File
import java.io.RandomAccessFile
import java.nio.file.Files
import org.junit.Assert.*
import org.junit.Test

class WikiOfflineAssetReadTest {
    private val image = WikiEntry("file", "image", "image.png", format = "png", revision = "one")

    @Test fun `reopening excludes empty oversized and stale revision files without deleting them`() = withRoot { root ->
        val store = WikiOfflineAssetStore(root)
        val entries = (0..3).map { image.copy(id = "$it", path = "$it.png") }
        val sizes = listOf(0L, MAX_OFFLINE_ASSET_BYTES.toLong() + 1, 8L, 16L)
        entries.zip(sizes).forEach { (entry, size) -> cacheFile(root, entry, size) }
        val requested = entries.take(3) + entries[3].copy(revision = "two")
        assertEquals(setOf("2"), store.cachedFiles("owner", "vault", requested).keys)
        assertEquals(4, File(root, wikiOfflineStorageIdentity("owner", "vault")).listFiles()!!.size)
        assertTrue(store.cachedFiles("other", "vault", entries).isEmpty())
        assertTrue(store.cachedFiles("owner", "other", entries).isEmpty())
    }

    @Test fun `reopening enforces aggregate and count limits even without a prior merge`() = withRoot { root ->
        val store = WikiOfflineAssetStore(root)
        val entries = (1..40).map { image.copy(id = "$it", path = "$it.png") }
        entries.forEach { cacheFile(root, it, MAX_OFFLINE_ASSET_BYTES.toLong()) }
        assertEquals(8, store.cachedFiles("owner", "vault", entries).size)
        entries.forEach { cacheFile(root, it, 1) }
        assertEquals(MAX_OFFLINE_ASSETS, store.cachedFiles("owner", "vault", entries).size)
        assertTrue(store.cachedFiles("owner", "vault", entries.map { it.copy(format = "pdf") }).isEmpty())
    }

    private fun cacheFile(root: File, entry: WikiEntry, size: Long) {
        val directory = File(root, wikiOfflineStorageIdentity("owner", "vault")).apply { mkdirs() }
        RandomAccessFile(File(directory, wikiOfflineAssetFileName(entry)), "rw").use { it.setLength(size) }
    }

    private fun withRoot(test: (File) -> Unit) {
        val root = Files.createTempDirectory("wiki-read-bounds-").toFile()
        try { test(root) } finally { root.deleteRecursively() }
    }
}
