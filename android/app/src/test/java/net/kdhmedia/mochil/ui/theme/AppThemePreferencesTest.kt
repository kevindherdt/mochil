package net.kdhmedia.mochil.ui.theme

import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class AppThemePreferencesTest {
    private fun stored(palette: String?) = RuntimeEnvironment.getApplication().let { context ->
        context.getSharedPreferences("appearance", 0).edit().clear().apply { palette?.let { putString("palette", it) }; putString("mode", "Dark"); putBoolean("reduce_motion", true) }.commit()
        AppThemePreferences(context).load()
    }

    @Test fun thereAreFivePalettes() =
        assertEquals(listOf("Ember", "Tide", "Paper", "MagentaCream", "EInk"), AppPalette.entries.map { it.name })

    @Test fun aSavedNocturneBecomesTideAndKeepsTheOtherSettings() {
        val loaded = stored("Nocturne")
        assertEquals(AppPalette.Tide, loaded.palette)
        assertEquals(AppThemeMode.Dark, loaded.mode)
        assertEquals(true, loaded.reduceMotion)
    }

    @Test fun otherPalettesAndMissingValuesAreUnchanged() {
        assertEquals(AppPalette.Paper, stored("Paper").palette)
        assertEquals(AppPalette.EInk, stored("EInk").palette)
        assertEquals(AppPalette.MagentaCream, stored("MagentaCream").palette)
        assertEquals(AppPalette.Ember, stored(null).palette)
        assertEquals(AppPalette.Ember, stored("Garbage").palette)
    }

    @Test fun namesSavedBeforeTheRenameStillResolve() {
        // Espresso and Kiln were the enum names before the two families were reworked.
        assertEquals(AppPalette.Ember, stored("Espresso").palette)
        assertEquals(AppPalette.Tide, stored("Kiln").palette)
    }
}
