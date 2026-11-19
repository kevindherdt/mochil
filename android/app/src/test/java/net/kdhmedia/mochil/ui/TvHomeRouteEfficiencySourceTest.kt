package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TvHomeRouteEfficiencySourceTest {
    private val source = File("src/main/java/net/kdhmedia/mochil/ui/TvHomeRoute.kt").readText()

    @Test
    fun `tv home never observes the catalogue, the rail owns the library list`() {
        assertFalse(source.contains("libraryViewModel.state"))
        assertTrue(File("src/main/java/net/kdhmedia/mochil/ui/TvNavRail.kt").readText().contains(".map { it.shellProjection() }"))
        assertFalse(source.contains("val library by libraryViewModel.state.collectAsStateWithLifecycle()"))
    }
}
