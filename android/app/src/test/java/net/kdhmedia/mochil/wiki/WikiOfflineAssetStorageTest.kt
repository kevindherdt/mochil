package net.kdhmedia.mochil.wiki

import java.io.File
import java.nio.file.Files
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import org.junit.Assert.*
import org.junit.Test

class WikiOfflineAssetStorageTest {
    private val first = WikiEntry("file", "first", "first.png", format = "png", revision = "1")
    private val second = WikiEntry("file", "second", "second.png", format = "png", revision = "1")

    @Test fun `missing optional downloads retain matching revision and pruning removes obsolete ones`() = withRoot { root ->
        val store = WikiOfflineAssetStore(root)
        assertEquals(2, store.mergeAndPrune("owner", "vault", listOf(first, second),
            mapOf(first to byteArrayOf(1, 2), second to byteArrayOf(3, 4))))
        assertEquals(2, WikiOfflineAssetStore(root).mergeAndPrune("owner", "vault", listOf(first, second), emptyMap()))
        assertEquals(1, store.mergeAndPrune("owner", "vault", listOf(second), emptyMap()))
        val directory = File(root, wikiOfflineStorageIdentity("owner", "vault"))
        assertFalse(File(directory, wikiOfflineAssetFileName(first)).exists())
        assertArrayEquals(byteArrayOf(3, 4), File(directory, wikiOfflineAssetFileName(second)).readBytes())
    }

    @Test fun `two overlapping instances preserve both desired assets without temporary leftovers`() = withRoot { root ->
        val start = CountDownLatch(1)
        val executor = Executors.newFixedThreadPool(2)
        try {
            val jobs = listOf(first, second).mapIndexed { index, entry -> executor.submit {
                val store = WikiOfflineAssetStore(root)
                check(start.await(2, TimeUnit.SECONDS))
                repeat(30) {
                    store.mergeAndPrune("owner", "vault", listOf(first, second),
                        mapOf(entry to ByteArray(16 * 1024) { (index + 1).toByte() }))
                }
            } }
            start.countDown()
            jobs.forEach { it.get(10, TimeUnit.SECONDS) }
            val directory = File(root, wikiOfflineStorageIdentity("owner", "vault"))
            assertEquals(setOf(wikiOfflineAssetFileName(first), wikiOfflineAssetFileName(second)),
                directory.listFiles()!!.map { it.name }.toSet())
            assertTrue(File(directory, wikiOfflineAssetFileName(first)).readBytes().all { it == 1.toByte() })
            assertTrue(File(directory, wikiOfflineAssetFileName(second)).readBytes().all { it == 2.toByte() })
        } finally {
            start.countDown()
            executor.shutdownNow()
            executor.awaitTermination(2, TimeUnit.SECONDS)
        }
    }

    @Test fun `pruning one vault never touches another vault or owner`() = withRoot { root ->
        val store = WikiOfflineAssetStore(root)
        for ((owner, vault) in listOf("one" to "a", "one" to "b", "two" to "a")) {
            store.mergeAndPrune(owner, vault, listOf(first), mapOf(first to byteArrayOf(5)))
        }
        store.mergeAndPrune("one", "a", emptyList(), emptyMap())
        for ((owner, vault) in listOf("one" to "b", "two" to "a")) {
            assertTrue(File(File(root, wikiOfflineStorageIdentity(owner, vault)), wikiOfflineAssetFileName(first)).isFile)
        }
    }

    private fun withRoot(test: (File) -> Unit) {
        val root = Files.createTempDirectory("wiki-assets-test-").toFile()
        try { test(root) } finally { root.deleteRecursively() }
    }
}
