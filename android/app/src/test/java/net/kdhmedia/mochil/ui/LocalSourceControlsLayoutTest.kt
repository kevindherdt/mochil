package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

class LocalSourceControlsLayoutTest {
    @Test fun sourceActionsStartCollapsedSoLibraryRetainsSpace() {
        val source = File("src/main/java/net/kdhmedia/mochil/ui/LocalFirstApp.kt").readText()
        assertTrue("Source actions should start collapsed", source.contains("var sourceActionsExpanded by rememberSaveable { mutableStateOf(false) }"))
        assertTrue("Source actions should only be displayed when expanded", source.contains("if (sourceActionsExpanded)"))
    }
}
