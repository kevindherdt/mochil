package net.kdhmedia.mochil.wiki

import java.io.File
import java.nio.file.Files
import kotlinx.coroutines.runBlocking
import org.junit.Assert.*
import org.junit.Test

class WikiOfflineAssetReuseTest {
    private val image = WikiEntry("file", "image", "image.png", format = "png", revision = "one")

    @Test fun `unchanged repeat mirror makes no image requests`() = withStore { _, store ->
        val images = (1..32).map { image.copy(id = "image-$it", path = "$it.png") }
        store.mergeAndPrune("owner", "vault", images, images.associateWith { byteArrayOf(1, 2) })
        var requests = 0
        val targets = store.downloadTargets("owner", "vault", images)
        val loaded = runBlocking { loadBoundedWikiAssets(targets, 4) { batch ->
            requests += batch.size
            batch.map { it to byteArrayOf(1, 2) }
        } }
        assertEquals(0, requests)
        assertTrue(loaded.isEmpty())
        assertEquals(32, store.mergeAndPrune("owner", "vault", images, loaded))
    }

    @Test fun `changed identity missing revision and other owners require download`() = withStore { _, store ->
        store.mergeAndPrune("owner", "vault", listOf(image), mapOf(image to byteArrayOf(1)))
        for (changed in listOf(image.copy(revision = "two"), image.copy(revision = ""), image.copy(path = "renamed.png"))) {
            assertEquals(listOf(changed), store.downloadTargets("owner", "vault", listOf(changed)))
        }
        assertEquals(listOf(image), store.downloadTargets("other", "vault", listOf(image)))
        assertEquals(listOf(image), store.downloadTargets("owner", "other", listOf(image)))
    }

    @Test fun `empty and oversized cached files are not reusable`() = withStore { root, store ->
        val directory = File(root, wikiOfflineStorageIdentity("owner", "vault")).apply { mkdirs() }
        val file = File(directory, wikiOfflineAssetFileName(image))
        file.writeBytes(byteArrayOf())
        assertEquals(listOf(image), store.downloadTargets("owner", "vault", listOf(image)))
        java.io.RandomAccessFile(file, "rw").use { it.setLength(MAX_OFFLINE_ASSET_BYTES.toLong() + 1) }
        assertEquals(listOf(image), store.downloadTargets("owner", "vault", listOf(image)))
    }

    private fun withStore(test: (File, WikiOfflineAssetStore) -> Unit) {
        val root = Files.createTempDirectory("wiki-reuse-test-").toFile()
        try { test(root, WikiOfflineAssetStore(root)) } finally { root.deleteRecursively() }
    }
}
