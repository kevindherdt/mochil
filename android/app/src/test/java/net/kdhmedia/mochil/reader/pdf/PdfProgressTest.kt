package net.kdhmedia.mochil.reader.pdf

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class PdfProgressTest {
    @Test fun `fixed layout local sync identities cannot collide around colons`() {
        assertNotEquals(fixedLayoutSyncKey("a:b", "c"), fixedLayoutSyncKey("a", "b:c"))
        assertEquals(fixedLayoutSyncKey("library", "book"), fixedLayoutSyncKey("library", "book"))
    }
    @Test fun fixedLayoutFormatsUseDistinctWireLocators() {
        assertEquals("pdf", FixedLayoutLocatorKind.Pdf.wireName)
        assertEquals("comic", FixedLayoutLocatorKind.Comic.wireName)
    }

    @Test fun structuralPageWinsOverFraction() {
        assertEquals(4, PdfServerPosition(1, 4, 0.9).pageFor(10))
    }

    @Test fun fractionRestoresLegacyPositionAndBoundsIt() {
        assertEquals(5, PdfServerPosition(1, null, 0.5).pageFor(11))
        assertEquals(0, PdfServerPosition(1, null, -5.0).pageFor(10))
        assertEquals(0, PdfServerPosition(1, 99, 0.0).pageFor(1))
    }

    @Test fun unchangedLocalPageFollowsNewerRemoteWithoutPrompt() {
        assertEquals(PdfOpenDecision.USE_REMOTE, decidePdfOpenPosition(4, 8, 3, PdfLocalSyncState(2, 4)))
    }

    @Test fun offlineLocalMovementIsNeverSilentlyDiscarded() {
        assertEquals(PdfOpenDecision.ASK, decidePdfOpenPosition(6, 8, 3, PdfLocalSyncState(2, 4)))
    }

    @Test fun sameOrAlreadyKnownRemoteDoesNotPrompt() {
        assertEquals(PdfOpenDecision.KEEP_LOCAL, decidePdfOpenPosition(8, 8, 3, PdfLocalSyncState(2, 4)))
        assertEquals(PdfOpenDecision.KEEP_LOCAL, decidePdfOpenPosition(6, 8, 2, PdfLocalSyncState(2, 4)))
    }

    @Test fun onlyAnActualPageChangeClaimsWriteOwnership() {
        assertEquals(false, hasPdfPageMovement(previous = 23, observed = 23))
        assertEquals(true, hasPdfPageMovement(previous = 23, observed = 24))
        assertEquals(true, hasPdfPageMovement(previous = 23, observed = 22))
    }

    @Test fun pendingPageIsPartOfDurableFixedLayoutState() {
        val state = PdfLocalSyncState(revision = 7, page = 12, pendingPage = 15)

        assertEquals(7, state.revision)
        assertEquals(12, state.page)
        assertEquals(15, state.pendingPage)
    }

    @Test fun failedOlderWriteCannotReplaceNewerQueuedPage() {
        assertEquals(18, pageToRetainAfterFailedWrite(attemptedPage = 17, newerQueuedPage = 18))
        assertEquals(17, pageToRetainAfterFailedWrite(attemptedPage = 17, newerQueuedPage = null))
    }

    @Test fun successfulWriteKeepsOnlyAPositionQueuedWhileItWasInFlight() {
        assertEquals(18, pendingPageAfterSuccessfulWrite(newerQueuedPage = 18))
        assertEquals(null, pendingPageAfterSuccessfulWrite(newerQueuedPage = null))
    }

    @Test fun successfulResponseMustAcknowledgeTheExactFixedLayoutPage() {
        assertEquals(true, fixedLayoutSaveAcknowledgesPage(submittedPage = 500, authoritativePage = 500))
        assertEquals(false, fixedLayoutSaveAcknowledgesPage(submittedPage = 500, authoritativePage = 501))
    }
}
