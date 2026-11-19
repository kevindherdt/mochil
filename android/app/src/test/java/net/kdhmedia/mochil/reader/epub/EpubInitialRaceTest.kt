package net.kdhmedia.mochil.reader.epub

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.async
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import org.readium.r2.shared.publication.Locator
import org.readium.r2.shared.publication.Manifest
import org.readium.r2.shared.publication.Publication
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class EpubInitialRaceTest {
    private fun locator(fraction: Double) = requireNotNull(Locator.fromJSON(JSONObject("""{"href":"chapter.xhtml","type":"application/xhtml+xml","locations":{"totalProgression":$fraction}}""")))
    private fun publication(): Publication = Publication(requireNotNull(Manifest.fromJSON(JSONObject("""{"metadata":{"title":"Test"},"readingOrder":[{"href":"chapter.xhtml","type":"application/xhtml+xml"}]}"""))))
    private val credential = DeviceCredential("https://example.test", "local", "token")
    private fun gateway(remote: Locator) = object : ReadingProgressGateway {
        override suspend fun get(credential: DeviceCredential, libraryId: String, bookId: String) = ServerReadingPosition(5, 0, 0.8, mapOf("readium" to remote.toJSON().toString()), "", null, "other", "Other")
        override suspend fun getExternal(credential: DeviceCredential, titleId: String) = error("unused")
        override suspend fun put(credential: DeviceCredential, libraryId: String, bookId: String, locator: Locator, fraction: Double, baseRevision: Long, spineIndex: Int, koReaderLocator: String?) = error("unused")
        override suspend fun putExternal(credential: DeviceCredential, titleId: String, locator: Locator, fraction: Double, baseRevision: Long) = error("unused")
    }

    @Test fun delayedGetNeverReturnsCapturedLocalAfterMovement() = runBlocking {
        for (outcome in listOf("null", "stale", "error", "unresolvable")) {
            val store = EpubLocatorStore(ApplicationProvider.getApplicationContext<Context>())
            val key = "delayed-$outcome"
            store.clear(key)
            val original = locator(0.2)
            val moved = locator(0.4)
            store.save(key, original)
            store.saveRevision(key, 3)
            val started = kotlinx.coroutines.CompletableDeferred<Unit>()
            val release = kotlinx.coroutines.CompletableDeferred<Unit>()
            val gateway = object : ReadingProgressGateway {
                override suspend fun get(credential: DeviceCredential, libraryId: String, bookId: String): ServerReadingPosition? {
                    started.complete(Unit)
                    release.await()
                    return when (outcome) {
                        "null" -> null
                        "stale" -> ServerReadingPosition(2, 0, 0.8, emptyMap(), "", null, "other", "Other")
                        "error" -> throw java.io.IOException("offline")
                        else -> ServerReadingPosition(5, 0, 0.8, emptyMap(), "", null, "other", "Other")
                    }
                }
                override suspend fun getExternal(credential: DeviceCredential, titleId: String) = error("unused")
                override suspend fun put(credential: DeviceCredential, libraryId: String, bookId: String, locator: Locator, fraction: Double, baseRevision: Long, spineIndex: Int, koReaderLocator: String?) = error("unused")
                override suspend fun putExternal(credential: DeviceCredential, titleId: String, locator: Locator, fraction: Double, baseRevision: Long) = error("unused")
            }
            val coordinator = EpubProgressCoordinator(store, key, credential, "library", "book", "", gateway)
            val result = async { coordinator.resolveInitial(publication(), original) { false } }
            started.await()
            store.save(key, moved)
            release.complete(Unit)
            assertEquals(outcome, moved.toJSON().toString(), result.await().locator?.toJSON()?.toString())
        }
    }

    @Test fun decliningRemoteReturnsLatestLocalPosition() = runBlocking {
        val store = EpubLocatorStore(ApplicationProvider.getApplicationContext<Context>())
        val key = "decline-race"
        store.clear(key)
        val local = locator(0.2)
        val moved = locator(0.4)
        store.save(key, local)
        val coordinator = EpubProgressCoordinator(store, key, credential, "library", "book", "", gateway(locator(0.8)))
        val update = coordinator.resolveInitial(publication(), local) { store.save(key, moved); false }
        assertEquals(moved.toJSON().toString(), update.locator?.toJSON()?.toString())
        assertTrue(store.pending(key))
    }

    @Test fun failedAdoptionDoesNotConsumeRemoteRevision() = runBlocking {
        val store = EpubLocatorStore(ApplicationProvider.getApplicationContext<Context>())
        val key = "failed-adoption-race"
        store.clear(key)
        val local = locator(0.2)
        val moved = locator(0.4)
        store.save(key, local)
        val coordinator = EpubProgressCoordinator(store, key, credential, "library", "book", "", gateway(locator(0.8)))
        val update = coordinator.resolveInitial(publication(), local) { store.save(key, moved); true }
        assertEquals(moved.toJSON().toString(), update.locator?.toJSON()?.toString())
        assertEquals(0L, store.revision(key))
        var prompted = false
        coordinator.resolveInitial(publication(), moved) { prompted = true; false }
        assertTrue(prompted)
    }
}
