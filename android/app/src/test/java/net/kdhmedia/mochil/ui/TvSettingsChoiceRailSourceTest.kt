package net.kdhmedia.mochil.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

class TvSettingsChoiceRailSourceTest {
    @Test
    fun `remote settings choices retain a full tv focus target`() {
        val source = sequenceOf(
            File("src/main/java/net/kdhmedia/mochil/ui/TvSettingsChoiceRail.kt"),
            File("app/src/main/java/net/kdhmedia/mochil/ui/TvSettingsChoiceRail.kt"),
            File("android/app/src/main/java/net/kdhmedia/mochil/ui/TvSettingsChoiceRail.kt"),
        ).first(File::isFile).readText()

        assertTrue(source.contains("Modifier.heightIn(min = 48.dp).tvFocusBorder()"))
    }
}
