package net.kdhmedia.mochil.ui.kit

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/** The kit reads sizes and colours from the generated tokens. A literal in here is a number the web can no longer move. */
class KitTokenLawTest {
    private val kit = File("src/main/java/net/kdhmedia/mochil/ui/kit")
    private val literal = Regex("""(?<![\w.])(?:[1-9]\d*(?:\.\d+)?|0\.\d+)\.(?:dp|sp)\b|Color\(0x""")

    @Test fun `the kit directory is found`() {
        assertTrue("run from the app module: ${kit.absolutePath}", kit.isDirectory)
    }

    @Test fun `no kit file carries a raw size or colour`() {
        val offenders = kit.walkTopDown().filter { it.extension == "kt" }
            .filterNot { it.useLines { lines -> lines.first().startsWith("// GENERATED") } }
            .flatMap { file ->
                file.readLines().mapIndexedNotNull { i, line ->
                    val code = line.substringBefore("//")
                    if (literal.containsMatchIn(code)) "${file.name}:${i + 1}: ${line.trim()}" else null
                }
            }.toList()
        assertTrue("literals in ui/kit (use MochilSpace / MochilFontSize / MochilChrome / the colour scheme):\n${offenders.joinToString("\n")}", offenders.isEmpty())
    }
}
