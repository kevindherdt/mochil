package net.kdhmedia.mochil.wiki

import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.cancel
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withContext
import org.junit.Assert.*
import org.junit.Test

class WikiOfflineAssetLoadTest {
    private fun images(count: Int) = (0 until count).map { WikiEntry("file", "$it", "$it.png", format = "png") }

    @Test fun `full byte budget stops further network batches`() = runBlocking {
        var requests = 0
        val image = ByteArray(MAX_OFFLINE_ASSET_BYTES)
        val result = loadBoundedWikiAssets(images(32), 4) { batch ->
            requests += batch.size
            batch.map { it to image }
        }
        assertEquals(8, requests)
        assertEquals(8, result.size)
        assertEquals(MAX_OFFLINE_ASSET_TOTAL_BYTES, result.values.sumOf { it.size })
    }

    @Test fun `missing images do not prevent later batches and item count stays bounded`() = runBlocking {
        var requests = 0
        val result = loadBoundedWikiAssets(images(100), 4) { batch ->
            requests += batch.size
            batch.map { it to if (it.id == "0") null else byteArrayOf(1) }
        }
        assertEquals(MAX_OFFLINE_ASSETS, requests)
        assertEquals(MAX_OFFLINE_ASSETS - 1, result.size)
    }

    @Test fun `cancelled batch never schedules the next batch`() = runBlocking {
        var requests = 0
        try {
            loadBoundedWikiAssets(images(12), 4) {
                requests++
                throw CancellationException("left vault")
            }
            fail("cancellation swallowed")
        } catch (_: CancellationException) { }
        assertEquals(1, requests)
    }

    @Test fun `missing image remains optional`() = runBlocking {
        assertNull(loadOptionalWikiAsset { throw IOException("missing") })
    }

    @Test fun `successful image is retained`() = runBlocking {
        val bytes = byteArrayOf(1, 2)
        assertSame(bytes, loadOptionalWikiAsset { bytes })
    }

    @Test fun `explicit cancellation is never treated as a missing image`() = runBlocking {
        val cancelled = CancellationException("left vault")
        try {
            loadOptionalWikiAsset { throw cancelled }
            fail("cancellation swallowed")
        } catch (actual: CancellationException) { assertSame(cancelled, actual) }
    }

    @Test fun `blocking fetch result is discarded after cancellation`() = runBlocking {
        try {
            withContext(Job()) {
                val context = currentCoroutineContext()
                loadOptionalWikiAsset {
                    context.cancel()
                    byteArrayOf(1)
                }
                fail("cancelled response published")
            }
            fail("cancelled work completed")
        } catch (_: CancellationException) { }
    }
}
