package net.kdhmedia.mochil.reader.epub

import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.readium.r2.shared.publication.Locator

class EpubProgressCoordinatorTest {
    @Test fun pendingOfflinePositionStaysDurableAndDeviceOnly() {
        val store = FakeProgressStore(pending = true)
        val coordinator = EpubProgressCoordinator(store, "book", null, "", "", "", NoOpProgressGateway)
        assertEquals(ReaderSyncPhase.DeviceOnly, coordinator.initialStatus().phase)
        assertTrue(coordinator.shouldResumePending())
        assertFalse(coordinator.canSynchronize)
    }

    @Test fun connectedCleanPositionReportsLastServerAcknowledgement() {
        val store = FakeProgressStore(pending = false, synchronizedAt = 1234)
        val coordinator = EpubProgressCoordinator(store, "book", DeviceCredential("https://example.test", "device", "token"), "library", "book", "", NoOpProgressGateway)
        assertEquals(ReaderSyncStatus(ReaderSyncPhase.Synced, 1234), coordinator.initialStatus())
    }

    @Test fun passiveConflictsRequireRealMovementButEquivalentPagesDoNotConflict() {
        assertTrue(readingProgressEquivalent(0.5, 0.5015))
        assertFalse(readingProgressEquivalent(0.5, 0.51))
        assertFalse(readingProgressMoved(0.5, 0.500001))
        assertTrue(readingProgressMoved(0.5, 0.5001))
    }

    @Test fun pendingOfflineReadingWinsOverAnOlderSameDeviceServerPosition() {
        assertEquals(
            EpubInitialPositionDecision.KeepLocal,
            decideEpubInitialPosition(localPending = true, sameDevice = true, equivalent = false),
        )
    }

    @Test fun cleanSameDeviceProgressUsesRemoteWhileEquivalentProgressKeepsTheExactLocalPage() {
        assertEquals(
            EpubInitialPositionDecision.UseRemote,
            decideEpubInitialPosition(localPending = false, sameDevice = true, equivalent = false),
        )
        assertEquals(
            EpubInitialPositionDecision.KeepLocalSynchronized,
            decideEpubInitialPosition(localPending = true, sameDevice = false, equivalent = true),
        )
        assertEquals(
            EpubInitialPositionDecision.KeepLocalSynchronized,
            decideEpubInitialPosition(localPending = false, sameDevice = true, equivalent = true),
        )
    }

    @Test fun cleanPassiveClientUsesNewerProgressFromAnotherDeviceWithoutPrompting() {
        assertEquals(
            EpubInitialPositionDecision.UseRemote,
            decideEpubInitialPosition(localPending = false, sameDevice = false, equivalent = false),
        )
    }

    @Test fun divergentProgressFromAnotherDeviceStillRequiresReview() {
        assertEquals(
            EpubInitialPositionDecision.Ask,
            decideEpubInitialPosition(localPending = true, sameDevice = false, equivalent = false),
        )
    }

    @Test fun staleReaderCannotClaimOwnershipUntilThePageActuallyMoves() {
        val ownership = ReaderProgressOwnership()
        ownership.waitForMovement(0.5)

        assertFalse(ownership.canWrite(0.5))
        assertFalse(ownership.canWrite(0.500001))
        assertTrue(ownership.canWrite(0.51))
        assertTrue(ownership.canWrite(0.51))
    }

    @Test fun divergentProgressPromptIdentifiesTheOtherDeviceAndExplainsLocalOwnership() {
        val position = ServerReadingPosition(4, 0, 0.487, emptyMap(), "", null, "remote", "Samsung Tab")

        assertEquals(
            "Samsung Tab saved newer progress at 48%. This device also has unsynced reading progress. Go to the newer position?",
            readingPositionPromptMessage(position),
        )
    }
}

private class FakeProgressStore(
    private var pending: Boolean = false,
    private val synchronizedAt: Long = 0,
) : EpubProgressStateStore {
    private var revision = 0L
    override fun beginSync(publicationKey: String) { pending = true }
    override fun endSync(publicationKey: String, acknowledged: Boolean) = Unit
    override fun pending(publicationKey: String) = pending
    override fun markPending(publicationKey: String) { pending = true }
    override fun markSynchronized(publicationKey: String, locator: Locator, revision: Long, nowMillis: Long): Long { this.revision = revision; pending = false; return nowMillis }
    override fun adoptRemote(publicationKey: String, expected: Locator?, locator: Locator, revision: Long): Boolean { this.revision = revision; pending = false; return true }
    override fun current(publicationKey: String): Locator? = null
    override fun revision(publicationKey: String) = revision
    override fun saveRevision(publicationKey: String, revision: Long) { this.revision = revision }
    override fun lastSynchronizedAt(publicationKey: String) = synchronizedAt
}

private object NoOpProgressGateway : ReadingProgressGateway {
    override suspend fun get(credential: DeviceCredential, libraryId: String, bookId: String) = null
    override suspend fun put(credential: DeviceCredential, libraryId: String, bookId: String, locator: Locator, fraction: Double, baseRevision: Long, spineIndex: Int, koReaderLocator: String?) = error("not used")
    override suspend fun getExternal(credential: DeviceCredential, titleId: String) = null
    override suspend fun putExternal(credential: DeviceCredential, titleId: String, locator: Locator, fraction: Double, baseRevision: Long) = error("not used")
}
