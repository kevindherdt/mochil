package net.kdhmedia.mochil.ui.kit

import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.VectorGroup
import androidx.compose.ui.graphics.vector.VectorNode
import androidx.compose.ui.graphics.vector.VectorPath
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class LucideIconsTest {
    private val dock = mapOf(
        "BookOpen" to LucideIcons.BookOpen,
        "Compass" to LucideIcons.Compass,
        "Sparkles" to LucideIcons.Sparkles,
        "NotebookText" to LucideIcons.NotebookText,
        "Settings" to LucideIcons.Settings,
        "Server" to LucideIcons.Server,
        "Palette" to LucideIcons.Palette,
        "LibraryBig" to LucideIcons.LibraryBig,
        "Cloud" to LucideIcons.Cloud,
        "Download" to LucideIcons.Download,
        "BookOpenText" to LucideIcons.BookOpenText,
        "MonitorPlay" to LucideIcons.MonitorPlay,
        "ShieldCheck" to LucideIcons.ShieldCheck,
        "ChevronRight" to LucideIcons.ChevronRight,
        "ArrowLeft" to LucideIcons.ArrowLeft,
    )

    private fun paths(node: VectorNode): List<VectorPath> = when (node) {
        is VectorPath -> listOf(node)
        is VectorGroup -> node.flatMap { paths(it) }
        else -> emptyList()
    }

    @Test fun `every glyph is a 24 unit stroke drawing`() {
        dock.forEach { (name, icon) ->
            assertEquals("$name viewport", 24f, icon.viewportWidth, 0f)
            assertEquals("$name viewport", 24f, icon.viewportHeight, 0f)
            val drawn = paths(icon.root)
            assertTrue("$name has paths", drawn.isNotEmpty())
            drawn.forEach { path ->
                assertNull("$name path is not filled", path.fill)
                assertNotNull("$name path is stroked", path.stroke)
                assertEquals("$name stroke width", 2f, path.strokeLineWidth, 0f)
            }
        }
    }

    @Test fun `glyphs are distinct drawings`() {
        val signatures = dock.values.map { icon: ImageVector -> paths(icon.root).joinToString("|") { it.pathData.toString() } }
        assertEquals(signatures.size, signatures.toSet().size)
    }
}
