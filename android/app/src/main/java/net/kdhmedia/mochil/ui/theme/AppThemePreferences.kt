package net.kdhmedia.mochil.ui.theme

import android.content.Context
import net.kdhmedia.mochil.catalogue.ClientDeviceClass
import net.kdhmedia.mochil.catalogue.detectClientDeviceClass

enum class AppPalette(val label: String) { Espresso("Espresso"), Kiln("Kiln"), Paper("Paper"), MagentaCream("Magenta Cream"), EInk("E-ink") }
enum class AppThemeMode(val label: String) { System("Device"), Light("Light"), Dark("Dark") }
data class AppThemeSelection(val palette: AppPalette = AppPalette.Espresso, val mode: AppThemeMode = AppThemeMode.System, val reduceMotion: Boolean = false)

class AppThemePreferences(private val context: Context) {
    private val preferences = context.getSharedPreferences("appearance", Context.MODE_PRIVATE)
    /** Animations are off by default on an e-ink screen, where they only smear; elsewhere they stay on until switched off. */
    fun load() = AppThemeSelection(
        enumValueOrDefault(preferences.getString("palette", null)?.let { if (it == "Nocturne") "Kiln" else it }, AppPalette.Espresso), // Nocturne was retired; its dark cool look is closest to Kiln
        enumValueOrDefault(preferences.getString("mode", null), AppThemeMode.System),
        if (preferences.contains("reduce_motion")) preferences.getBoolean("reduce_motion", false) else detectClientDeviceClass(context) == ClientDeviceClass.EInk,
    )
    fun save(selection: AppThemeSelection) { preferences.edit().putString("palette", selection.palette.name).putString("mode", selection.mode.name).putBoolean("reduce_motion", selection.reduceMotion).apply() }
    private inline fun <reified T : Enum<T>> enumValueOrDefault(value: String?, fallback: T): T = enumValues<T>().firstOrNull { it.name == value } ?: fallback
}
