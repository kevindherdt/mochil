package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class EpubLocatorPolicyTest {
    @Test fun offlineOrNewerExactPositionsRemainPendingUntilSamePositionIsAcknowledged() {
        assertTrue(positionNeedsSynchronization("page-24", null))
        assertTrue(positionNeedsSynchronization("page-24", "page-23"))
        assertFalse(positionNeedsSynchronization("page-24", "page-24"))
    }
}
