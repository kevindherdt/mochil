package net.kdhmedia.mochil.ui

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TvPagingTest {
    @Test fun `loads before remote focus reaches the final card`() {
        assertTrue(shouldLoadTvPage(totalItems = 40, lastVisibleIndex = 34, hasNextPage = true, loading = false))
    }

    @Test fun `does not duplicate an active request`() {
        assertFalse(shouldLoadTvPage(totalItems = 40, lastVisibleIndex = 39, hasNextPage = true, loading = true))
    }

    @Test fun `does not load without a cursor or visible content`() {
        assertFalse(shouldLoadTvPage(totalItems = 40, lastVisibleIndex = 39, hasNextPage = false, loading = false))
        assertFalse(shouldLoadTvPage(totalItems = 0, lastVisibleIndex = -1, hasNextPage = true, loading = false))
    }
}
