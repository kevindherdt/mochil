package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LibraryRequestGenerationTest {
    @Test fun shelfChangesInvalidateOlderShelfWork() {
        val requests = LibraryRequestGeneration()
        val old = requests.captureShelf()
        val current = requests.advanceShelf()
        assertFalse(requests.isCurrentShelf(old))
        assertTrue(requests.isCurrentShelf(current))
    }

    @Test fun groupAndShelfPagingHaveIndependentLifetimes() {
        val requests = LibraryRequestGeneration()
        val shelf = requests.advanceShelf()
        val group = requests.advanceGroup()
        requests.advanceGroup()
        assertTrue(requests.isCurrentShelf(shelf))
        assertFalse(requests.isCurrentGroup(group))
    }
}
