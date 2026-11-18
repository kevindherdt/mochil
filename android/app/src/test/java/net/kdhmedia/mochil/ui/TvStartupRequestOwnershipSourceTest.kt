package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

class TvStartupRequestOwnershipSourceTest {
    @Test fun `startup library refresh rejects overlapping requests`() {
        val library = File("src/main/java/net/kdhmedia/mochil/catalogue/LibraryViewModel.kt").readText()
        assertTrue(library.contains("if (summaryRefreshInFlight)"))
        assertTrue(library.contains("if (!force && !current.offlineOnly && current.initialLoading) return"))
        assertTrue(library.contains("if (force) summaryRetryPending = true"))
    }
}
