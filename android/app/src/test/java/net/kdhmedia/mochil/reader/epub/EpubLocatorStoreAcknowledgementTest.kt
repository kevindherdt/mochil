package net.kdhmedia.mochil.reader.epub

import androidx.test.core.app.ApplicationProvider
import android.content.Context
import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.readium.r2.shared.publication.Locator
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class EpubLocatorStoreAcknowledgementTest {
    @Test fun olderAcknowledgementMustNotReplaceNewerLocalPosition() {
        val store = EpubLocatorStore(ApplicationProvider.getApplicationContext<Context>())
        val key = "stale-ack-test"
        store.clear(key)
        val old = locator(0.2)
        val newer = locator(0.4)
        store.save(key, old)
        store.save(key, newer) // Reader moved while the old request was in flight.

        store.markSynchronized(key, old, 3, 1000)

        assertEquals(newer.toJSON().toString(), store.load(key)?.toJSON()?.toString())
        assertTrue(store.pending(key))
        assertEquals(3L, store.revision(key))
    }

    @Test fun returningToPreviouslySyncedPositionDuringFlightRemainsPending() {
        val store = EpubLocatorStore(ApplicationProvider.getApplicationContext<Context>())
        val key = "return-during-flight"
        store.clear(key)
        val original = locator(0.2)
        val sent = locator(0.4)
        store.save(key, original)
        store.markSynchronized(key, original, 1, 100)
        store.save(key, sent)
        store.beginSync(key)
        store.save(key, original)
        assertTrue(store.pending(key))
        store.markSynchronized(key, sent, 2, 200)
        store.endSync(key)
        assertEquals(original.toJSON().toString(), store.load(key)?.toJSON()?.toString())
        assertTrue(store.pending(key))
        assertEquals(2L, store.revision(key))
    }

    @Test fun matchingAcknowledgementClearsPending() {
        val store = EpubLocatorStore(ApplicationProvider.getApplicationContext<Context>())
        val key = "matching-ack"
        store.clear(key)
        val position = locator(0.4)
        store.saveNow(key, position)
        store.markSynchronized(key, position, 2, 200)
        org.junit.Assert.assertFalse(store.pending(key))
        assertEquals(200L, store.lastSynchronizedAt(key))
    }

    @Test fun remoteAdoptionOnlyReplacesExpectedLocalPosition() {
        val context = ApplicationProvider.getApplicationContext<Context>()
        val store = EpubLocatorStore(context)
        val other = EpubLocatorStore(context)
        val key = "remote-compare"
        store.clear(key)
        val original = locator(0.2)
        val remote = locator(0.7)
        val newer = locator(0.4)
        store.save(key, original)
        other.save(key, newer)
        org.junit.Assert.assertFalse(store.adoptRemote(key, original, remote, 2))
        assertEquals(newer.toJSON().toString(), store.load(key)?.toJSON()?.toString())
        assertTrue(store.pending(key))
        assertTrue(store.adoptRemote(key, newer, remote, 2))
        assertEquals(remote.toJSON().toString(), store.load(key)?.toJSON()?.toString())
        org.junit.Assert.assertFalse(store.pending(key))
    }

    @Test fun lowerRevisionAcknowledgementCannotUndoNewerAcknowledgement() {
        val store = EpubLocatorStore(ApplicationProvider.getApplicationContext<Context>())
        val key = "out-of-order-ack"
        store.clear(key)
        val newer = locator(0.7)
        store.save(key, newer)
        store.markSynchronized(key, newer, 4, 400)
        store.markSynchronized(key, locator(0.2), 3, 300)
        assertEquals(4L, store.revision(key))
        assertEquals(400L, store.lastSynchronizedAt(key))
        org.junit.Assert.assertFalse(store.pending(key))
    }

    @Test fun overlappingAcknowledgementsClearPendingAfterBothFinish() {
        val store = EpubLocatorStore(ApplicationProvider.getApplicationContext<Context>())
        val key = "overlapping-acks"
        store.clear(key)
        val position = locator(0.4)
        store.save(key, position)
        store.beginSync(key)
        store.beginSync(key)
        store.markSynchronized(key, position, 2, 200)
        store.markSynchronized(key, position, 3, 300)
        store.endSync(key, true)
        store.endSync(key, true)
        org.junit.Assert.assertFalse(store.pending(key))
    }

    @Test fun failedOverlappingRequestKeepsPendingEvenAfterAnotherAcknowledgement() {
        val store = EpubLocatorStore(ApplicationProvider.getApplicationContext<Context>())
        val key = "overlapping-failure"
        store.clear(key)
        val position = locator(0.4)
        store.save(key, position)
        store.beginSync(key)
        store.beginSync(key)
        store.endSync(key, false)
        store.markSynchronized(key, position, 2, 200)
        store.endSync(key, true)
        assertTrue(store.pending(key))
    }

    @Test fun failedFlightThenReturningToOldAcknowledgementStaysPendingUntilConfirmed() {
        val store = EpubLocatorStore(ApplicationProvider.getApplicationContext<Context>())
        val key = "uncertain-return"
        store.clear(key)
        val original = locator(0.2)
        val sent = locator(0.4)
        store.save(key, original)
        store.markSynchronized(key, original, 1, 100)
        store.save(key, sent)
        store.beginSync(key)
        store.endSync(key, false)
        store.saveNow(key, original)
        assertTrue(store.pending(key))
        store.beginSync(key)
        store.markSynchronized(key, original, 2, 200)
        store.endSync(key, true)
        org.junit.Assert.assertFalse(store.pending(key))
    }

    private fun locator(fraction: Double): Locator = requireNotNull(Locator.fromJSON(JSONObject().apply {
        put("href", "chapter.xhtml")
        put("type", "application/xhtml+xml")
        put("locations", JSONObject().put("totalProgression", fraction))
    }))
}
