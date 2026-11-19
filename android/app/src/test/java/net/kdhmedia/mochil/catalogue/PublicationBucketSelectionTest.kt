package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Test

class PublicationBucketSelectionTest {
    @Test fun selectingTheActiveLetterClearsTheFilter() {
        assertEquals("", toggledPublicationBucket("M", "M"))
        assertEquals("", toggledPublicationBucket("#", "#"))
    }

    @Test fun selectingAnotherLetterReplacesTheFilter() {
        assertEquals("M", toggledPublicationBucket("A", "M"))
        assertEquals("#", toggledPublicationBucket("", "#"))
    }
}
