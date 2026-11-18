package net.kdhmedia.mochil.catalogue

import android.app.UiModeManager
import android.content.Context
import android.content.pm.PackageManager
import android.content.res.Configuration
import android.os.Build

enum class ClientDeviceClass(val label: String) {
    Handheld("This device"),
    Television("Android TV"),
    EInk("E-ink device"),
}

class LibraryVisibilityStore(context: Context) {
    private val preferences = context.getSharedPreferences("library_visibility", Context.MODE_PRIVATE)
    val deviceClass: ClientDeviceClass = detectClientDeviceClass(context)

    fun isVisible(library: LibrarySummary): Boolean {
        val shown = preferences.getStringSet(SHOWN, emptySet()).orEmpty()
        val hidden = preferences.getStringSet(HIDDEN, emptySet()).orEmpty()
        return resolveLibraryVisibility(library.id, defaultLibraryVisible(deviceClass, library.kind), shown, hidden)
    }

    fun setVisible(library: LibrarySummary, visible: Boolean) {
        val shown = preferences.getStringSet(SHOWN, emptySet()).orEmpty().toMutableSet()
        val hidden = preferences.getStringSet(HIDDEN, emptySet()).orEmpty().toMutableSet()
        shown.remove(library.id)
        hidden.remove(library.id)
        if (visible != defaultLibraryVisible(deviceClass, library.kind)) {
            (if (visible) shown else hidden).add(library.id)
        }
        preferences.edit().putStringSet(SHOWN, shown).putStringSet(HIDDEN, hidden).apply()
    }

    fun filter(libraries: List<LibrarySummary>) = libraries.filter(::isVisible)

    private companion object {
        const val SHOWN = "shown"
        const val HIDDEN = "hidden"
    }
}

internal fun defaultLibraryVisible(deviceClass: ClientDeviceClass, kind: String): Boolean {
    val normalized = kind.lowercase().replace("-", "").replace("_", "").replace(" ", "")
    return when (deviceClass) {
        ClientDeviceClass.Handheld -> true
        ClientDeviceClass.Television -> normalized !in setOf("book", "books", "lightnovel", "lightnovels", "novel", "novels", "comic", "comics", "manga", "document", "documents", "calibre")
        ClientDeviceClass.EInk -> normalized !in setOf("movie", "movies", "tv", "television", "tvshow", "tvshows", "anime", "video", "videos")
    }
}

internal fun resolveLibraryVisibility(id: String, defaultVisible: Boolean, shown: Set<String>, hidden: Set<String>): Boolean =
    when (id) {
        in shown -> true
        in hidden -> false
        else -> defaultVisible
    }

internal fun detectClientDeviceClass(context: Context): ClientDeviceClass {
    val uiMode = (context.getSystemService(Context.UI_MODE_SERVICE) as? UiModeManager)?.currentModeType
        ?: (context.resources.configuration.uiMode and Configuration.UI_MODE_TYPE_MASK)
    if (uiMode == Configuration.UI_MODE_TYPE_TELEVISION || context.packageManager.hasSystemFeature(PackageManager.FEATURE_LEANBACK)) {
        return ClientDeviceClass.Television
    }
    val features = context.packageManager.systemAvailableFeatures.mapNotNull { it.name }.toSet()
    return if (isEInkDevice(features, Build.MANUFACTURER, Build.BRAND, Build.MODEL)) ClientDeviceClass.EInk else ClientDeviceClass.Handheld
}

internal fun isEInkDevice(features: Set<String>, manufacturer: String, brand: String, model: String): Boolean {
    if (features.any { feature -> feature.lowercase().replace("_", "").contains("eink") }) return true
    val identity = "$manufacturer $brand $model".lowercase()
    val knownEInk = listOf("onyx", "boox", "kobo", "remarkable", "pocketbook", "boyue", "meebook", "inkbook", "tolino")
    return knownEInk.any(identity::contains) || model.lowercase().contains("kindle")
}
