package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Test

class TvGridPositionStoreTest {
    @Test
    fun `route keys isolate filters layouts and ambiguous text`() {
        val base = TvGridRouteKey("music", "Albums", "Covers", "Artist:Album", "Newest", "All", "", "")
        assertNotEquals(base.stableKey(), base.copy(layout = "List").stableKey())
        assertNotEquals(base.stableKey(), base.copy(query = "baki").stableKey())
        assertNotEquals(base.stableKey(), base.copy(bucket = "B").stableKey())
        assertNotEquals(
            base.copy(libraryId = "a:b", hierarchy = "c").stableKey(),
            base.copy(libraryId = "a", hierarchy = "b:c").stableKey(),
        )
    }

    @Test
    fun `position store is access ordered and bounded`() {
        val store = TvGridPositionStore(maximumEntries = 2)
        store.put("albums", 12, 4)
        store.put("artists", 3, 2)
        assertEquals(TvGridPosition(12, 4), store.get("albums"))
        store.put("tracks", 7, 1)
        assertNull(store.get("artists"))
        assertEquals(TvGridPosition(12, 4), store.get("albums"))
        assertEquals(TvGridPosition(7, 1), store.get("tracks"))
    }

    @Test
    fun `position store clamps invalid scroll coordinates`() {
        val store = TvGridPositionStore(maximumEntries = 1)
        store.put("library", -8, -20)
        assertEquals(TvGridPosition(0, 0), store.get("library"))
    }

    @Test
    fun `focused item survives scroll updates and remains route scoped`() {
        val store = TvGridPositionStore(maximumEntries = 2)
        store.focus("albums", "album-12")
        store.put("albums", 9, 18)

        assertEquals(TvGridPosition(9, 18, "album-12"), store.get("albums"))
        assertEquals(true, store.hasFocus("albums"))
        assertEquals("album-12", store.focusedItem("albums"))
        assertEquals(false, store.hasFocus("artists"))
        assertNull(store.focusedItem("artists"))
    }
}
