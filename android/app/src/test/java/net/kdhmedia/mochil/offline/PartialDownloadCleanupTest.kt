package net.kdhmedia.mochil.offline

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PartialDownloadCleanupTest {
    @Test fun `process startup delegates cleanup away from the main thread`() {
        val source = java.io.File("src/main/java/net/kdhmedia/mochil/offline/PartialDownloadCleanup.kt").readText()
        assertTrue(source.contains("Dispatchers.IO"))
        assertTrue(source.contains("scope.launch { clean(filesRoot) }"))
    }

    @Test fun `startup removes only partial media files`() {
        val root = kotlin.io.path.createTempDirectory("mochil-partials").toFile()
        try {
            val music = root.resolve("music").apply { mkdirs() }
            val publications = root.resolve("publications").apply { mkdirs() }
            val audioPart = music.resolve("track.mp3.part").apply { writeText("partial") }
            val bookPart = publications.resolve("book.epub.part").apply { writeText("partial") }
            val complete = music.resolve("track.mp3").apply { writeText("complete") }
            assertEquals(2, PartialDownloadCleanup.clean(root))
            assertFalse(audioPart.exists())
            assertFalse(bookPart.exists())
            assertTrue(complete.exists())
        } finally { root.deleteRecursively() }
    }
}
