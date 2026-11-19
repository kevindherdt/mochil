package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The search panel on the book and music shelves is a place the viewer opened; a rotation used to close it, because
 * the flag was a plain `remember` and the activity has no configChanges. Video's panel already lives in the ViewModel.
 * Compose's saved-instance state is what survives recreation and cannot be observed without instrumentation, so the
 * property is pinned in the source.
 */
class SearchPanelRotationSourceTest {
    private fun source(path: String) = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/$path"))

    @Test fun `the book shelf remembers its open search panel`() {
        val source = source("PublicationScreen.kt")
        assertTrue(source.contains("var searchExpanded by rememberSaveable { mutableStateOf(false) }"))
    }

    @Test fun `the music shelf remembers its open search panel`() {
        val source = source("MusicScreen.kt")
        assertTrue(source.contains("var searchExpanded by rememberSaveable { mutableStateOf(false) }"))
    }
}
