package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class TvResumeRefreshTest {
    @Test fun `TV resume refreshes only state visible to the active route`() {
        assertEquals(setOf(TvResumeRefresh.Libraries, TvResumeRefresh.Dashboard), tvResumeRefreshes("Home", "music"))
        assertEquals(setOf(TvResumeRefresh.Libraries, TvResumeRefresh.VideoProgress), tvResumeRefreshes("Library", "anime"))
        assertEquals(setOf(TvResumeRefresh.Libraries), tvResumeRefreshes("Library", "music"))
        assertTrue(tvResumeRefreshes("NowPlaying", "music").isEmpty())
    }
}
