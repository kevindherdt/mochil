package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

/**
 * A library opens on the plain list of books, as on the web, where the reading-status chips live. The television keeps the
 * series view it has always opened on, and a view somebody chose is never replaced by the default.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class LibraryNavigationDefaultTest {
    private fun store() = LibraryNavigationStore(RuntimeEnvironment.getApplication())

    @Test fun `a phone with no saved choice opens on all books`() {
        assertEquals(PublicationGrouping.Books, store().grouping(ClientDeviceClass.Handheld))
        assertEquals(PublicationGrouping.Books, store().grouping(ClientDeviceClass.EInk))
    }

    @Test fun `the television keeps series`() {
        assertEquals(PublicationGrouping.Series, store().grouping(ClientDeviceClass.Television))
    }

    @Test fun `a saved choice wins on every device`() {
        val saved = store().apply { setGrouping(PublicationGrouping.Series) }
        assertEquals(PublicationGrouping.Series, saved.grouping(ClientDeviceClass.Handheld))
        store().setGrouping(PublicationGrouping.Authors)
        assertEquals(PublicationGrouping.Authors, store().grouping(ClientDeviceClass.Handheld))
        assertEquals(PublicationGrouping.Authors, store().grouping(ClientDeviceClass.Television))
    }

    @Test fun `a saved short trip still falls back to the device default`() {
        store().setGrouping(PublicationGrouping.Continue)
        assertEquals(PublicationGrouping.Books, store().grouping(ClientDeviceClass.Handheld))
        assertEquals(PublicationGrouping.Series, store().grouping(ClientDeviceClass.Television))
    }
}
