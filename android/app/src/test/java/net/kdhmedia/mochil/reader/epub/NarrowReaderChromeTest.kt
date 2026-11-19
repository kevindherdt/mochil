package net.kdhmedia.mochil.reader.epub

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

class NarrowReaderChromeTest {
    private val chrome = File("src/main/java/net/kdhmedia/mochil/reader/epub/EpubReaderChrome.kt").readText()

    @Test fun settingsControlsAreGroupedIntoTabsRatherThanOneCrowdedList() {
        val settings = chrome.substringAfter("AnimatedVisibility(settingsExpanded)").substringBefore("if (fontsExpanded)")
        assertTrue("Settings need grouped tab sections", settings.contains("SettingsSection"))
        assertTrue("Settings need a tab row to switch sections", settings.contains("PrimaryScrollableTabRow("))
    }

    @Test fun navigationTabsCanScrollInsteadOfBreakingWords() {
        assertTrue("Navigation needs scrollable tabs", chrome.contains("PrimaryScrollableTabRow("))
    }
}
