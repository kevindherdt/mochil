package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Test

class LibraryRetryPolicyTest {
    @Test fun retriesTheSmallestFailedPublicationScope() {
        assertEquals(LibraryRetryTarget.Discovery, libraryRetryTarget(hasSelectedLibrary = false, hasOpenGroup = false))
        assertEquals(LibraryRetryTarget.Shelf, libraryRetryTarget(hasSelectedLibrary = true, hasOpenGroup = false))
        assertEquals(LibraryRetryTarget.Group, libraryRetryTarget(hasSelectedLibrary = true, hasOpenGroup = true))
    }
}
