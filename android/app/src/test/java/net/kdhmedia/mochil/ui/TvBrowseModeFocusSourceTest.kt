package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

class TvBrowseModeFocusSourceTest {
    @Test fun `browse menu restores focus to its active mode`() {
        val source = File("src/main/java/net/kdhmedia/mochil/ui/TvLibraryScreen.kt").readText()
        val picker = source.substringAfter("private fun TvBrowseModePicker").substringBefore("@Composable\nprivate fun TvEmptyLibrary")
        assertTrue(picker.contains("if (expanded) selectedFocus.requestFocus()"))
        assertTrue(picker.contains("mode.id == selected.id"))
    }
}
