package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LocalMediaCountTest {
    @Test fun singleSupportedFileUsesSingularNoun() {
        val source = File("src/main/java/net/kdhmedia/mochil/ui/LocalMediaLibrary.kt").readText()
        assertTrue(source.contains("if (folder.mediaCount == 1) \"file\" else \"files\""))
        assertFalse(source.contains("\${folder.mediaCount} supported files"))
    }
}
