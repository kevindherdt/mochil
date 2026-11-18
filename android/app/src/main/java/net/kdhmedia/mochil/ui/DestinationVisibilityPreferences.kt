package net.kdhmedia.mochil.ui

import android.content.Context

enum class OptionalDestination(val storageKey: String) {
    Collections("collections"),
    Sources("sources"),
}

class DestinationVisibilityPreferences(context: Context) {
    private val preferences = context.getSharedPreferences("destination_visibility", Context.MODE_PRIVATE)

    fun isVisible(destination: OptionalDestination): Boolean =
        destination.storageKey !in preferences.getStringSet(HIDDEN, emptySet()).orEmpty()

    fun setVisible(destination: OptionalDestination, visible: Boolean) {
        val hidden = preferences.getStringSet(HIDDEN, emptySet()).orEmpty().toMutableSet()
        if (visible) hidden.remove(destination.storageKey) else hidden.add(destination.storageKey)
        preferences.edit().putStringSet(HIDDEN, hidden).apply()
    }

    private companion object { const val HIDDEN = "hidden" }
}
