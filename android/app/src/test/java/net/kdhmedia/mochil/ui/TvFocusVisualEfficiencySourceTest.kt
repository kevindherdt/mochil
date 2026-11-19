package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TvFocusVisualEfficiencySourceTest {
    @Test
    fun sharedFocusVisualsDoNotAttachIdleGraphicsOrBorders() {
        val controls = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/TvControls.kt"))
        val helper = controls.substringAfter("fun Modifier.tvFocusVisual").substringBefore("internal class TvSelectHoldState")
        assertTrue(helper.contains("if (scale != 1f)"))
        assertTrue(helper.contains("else -> Modifier"))

        listOf("TvHomeScreen.kt", "TvAlphabetGrid.kt", "TvSettingsCategories.kt").forEach { name ->
            val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/$name"))
            assertTrue("$name should use the shared conditional focus visual", source.contains(".tvFocusVisual("))
            assertFalse("$name should not allocate an idle scale modifier", source.contains("scale(if (focused)"))
            assertFalse("$name should not allocate an idle zero-width border", source.contains("border(if (focused)"))
        }
    }
}
