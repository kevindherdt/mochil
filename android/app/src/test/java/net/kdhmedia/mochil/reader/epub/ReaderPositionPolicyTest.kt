package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Test

class ReaderPositionPolicyTest {
    @Test fun exactPageIsNeverReplacedByPreviousStructuralElementForLocalRestore() {
        val positions = readerPositionPair(exactPage = "visible-page-24", structural = "paragraph-start-page-23")

        assertEquals("visible-page-24", positions.local)
        assertEquals("paragraph-start-page-23", positions.synchronized)
    }
}
