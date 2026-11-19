package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TvMediaCardEfficiencySourceTest {
    private val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/TvMediaGrid.kt"))

    @Test fun `unfocused cards do not allocate focus graphics modifiers`() {
        // TvWideMediaCard shares tvFocusVisual() with every other card (grow-and-settle +
        // border, animated, reduce-motion aware) instead of a bare unconditional scale/border;
        // tvFocusVisual's own `.then(if (scale != 1f) ... else Modifier)` and
        // `when { focused -> border; selected -> border; else -> Modifier }` are what keep the
        // no-allocation-when-unfocused property, so this only needs to confirm the shared helper
        // is actually used here rather than reintroducing a hand-rolled, unanimated modifier chain.
        assertTrue(source.contains(".tvFocusVisual(focused, shape)"))
        assertFalse(source.contains("scale(if (focused)"))
        assertFalse(source.contains("border(if (focused)"))
        assertFalse(source.contains("Modifier.scale(TV_CARD_FOCUS_SCALE)"))
    }
}
