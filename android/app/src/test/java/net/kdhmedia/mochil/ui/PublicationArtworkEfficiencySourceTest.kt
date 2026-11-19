package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Test

class PublicationArtworkEfficiencySourceTest {
    @Test fun `lazy publication covers do not animate every viewport decode`() {
        val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/PublicationItems.kt"))
        assertFalse(source.contains("crossfade(true)"))
    }
}
