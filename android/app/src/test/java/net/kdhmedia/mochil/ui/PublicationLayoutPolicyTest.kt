package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.catalogue.LibraryState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PublicationLayoutPolicyTest {
    @Test fun `tablet detail uses two panes`() = assertTrue(usesPublicationTwoPane(840, true))
    @Test fun `phone detail stays focused`() = assertFalse(usesPublicationTwoPane(839, true))
    @Test fun `tablet browse stays one pane until an item opens`() = assertFalse(usesPublicationTwoPane(1200, false))
    @Test fun `media detail follows the same adaptive breakpoint`() = assertTrue(usesMediaTwoPane(900, true))

    @Test fun `group paging never reuses the root catalogue cursor`() {
        val paging = LibraryState(nextCursor = "root", groupNextCursor = "group", groupLoadingMore = true, groupError = "retry").groupPagingState()
        assertEquals("group", paging.nextCursor)
        assertEquals(true, paging.loadingMore)
        assertEquals("retry", paging.error)
    }
}
