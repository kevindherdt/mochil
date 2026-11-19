package net.kdhmedia.mochil

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/** Media3's track picker is an AppCompat dialog: with the plain app theme it throws when shown and takes the player down. */
class TrackDialogThemeTest {
    private fun source(path: String) = sequenceOf(File("src/main/$path"), File("app/src/main/$path")).firstOrNull(File::isFile)?.readText() ?: error("$path was not found")

    @Test fun trackPickerHasAnAppCompatTheme() {
        assertTrue(Regex("""name="Theme\.Mochil\.TrackDialog"\s+parent="Theme\.AppCompat""").containsMatchIn(source("res/values/styles.xml")))
    }

    @Test fun playerUsesThatThemeForTheTrackPicker() {
        assertTrue(source("java/net/kdhmedia/mochil/video/VideoPlayerActivity.kt").contains("setTheme(net.kdhmedia.mochil.R.style.Theme_Mochil_TrackDialog)"))
    }
}
