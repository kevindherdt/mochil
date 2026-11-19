package net.kdhmedia.mochil.narration

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Media3 refuses a second media session in one app with the same id, and the app dies when the
 * service that builds it starts. Only one session may use the default id; the rest name themselves.
 */
class MediaSessionIdsTest {
    private val sources = sequenceOf(File("src/main/java"), File("app/src/main/java")).firstOrNull(File::isDirectory)
        ?: error("Android sources were not found")

    private fun builders(): List<Pair<String, String>> = sources.walkTopDown().filter { it.extension == "kt" }.flatMap { file ->
        val text = file.readText()
        Regex("""MediaSession\.Builder\([^)]*\)([^\n]*(\n\s*\.[^\n]*)*)""").findAll(text).map { file.name to it.value }
    }.toList()

    @Test fun atMostOneMediaSessionUsesTheDefaultId() {
        val found = builders()
        assertTrue("expected the music and narration sessions", found.size >= 2)
        val unnamed = found.filter { !it.second.contains(".setId(") }.map { it.first }
        assertTrue("sessions without their own id: $unnamed", unnamed.size <= 1)
    }

    @Test fun theNarrationSessionHasAnIdOfItsOwn() {
        val narration = builders().single { it.first == "NarrationService.kt" }
        assertTrue(narration.second.contains(".setId("))
        assertEquals(1, Regex("""SESSION_ID = "mochil-narration"""").findAll(File(sources, "net/kdhmedia/mochil/narration/NarrationService.kt").readText()).count())
    }
}
