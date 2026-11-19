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

    @Test fun thereAreFourPalettes() =
        assertEquals(listOf("Espresso", "Kiln", "Paper", "EInk"), AppPalette.entries.map { it.name })

    @Test fun aSavedNocturneBecomesKilnAndKeepsTheOtherSettings() {
        val loaded = stored("Nocturne")
        assertEquals(AppPalette.Kiln, loaded.palette)
        assertEquals(AppThemeMode.Dark, loaded.mode)
        assertEquals(true, loaded.reduceMotion)
    }

    @Test fun otherPalettesAndMissingValuesAreUnchanged() {
        assertEquals(AppPalette.Paper, stored("Paper").palette)
        assertEquals(AppPalette.EInk, stored("EInk").palette)
        assertEquals(AppPalette.Espresso, stored(null).palette)
        assertEquals(AppPalette.Espresso, stored("Garbage").palette)
    }
}
