package net.kdhmedia.mochil.catalogue

import android.content.Context
import android.content.SharedPreferences

/**
 * The one saved Covers/List choice shared by books, music, podcasts and video. Every kind used to keep its own copy; none had a
 * reason to differ (all defaulted to Covers), so a choice made in one place never carried over to the others.
 *
 * Choices saved by older versions are folded in once, on first read: a List anywhere wins, because Covers is the default and so
 * List is the only choice someone actually made. The old keys are left alone and never read again.
 */
internal class BrowseLayoutStore(context: Context) {
    private val app = context.applicationContext
    private val preferences = app.getSharedPreferences(FILE, Context.MODE_PRIVATE)

    fun isList(): Boolean {
        preferences.getString(KEY, null)?.let { return it == LIST }
        val migrated = legacyChoices().any { it == LIST }
        preferences.edit().putString(KEY, if (migrated) LIST else COVERS).apply()
        return migrated
    }

    fun setList(list: Boolean) {
        preferences.edit().putString(KEY, if (list) LIST else COVERS).apply()
    }

    /** Calls [onChange] on the main thread whenever any screen changes the choice. Keep the returned handle alive as long as you listen. */
    fun listen(onChange: (list: Boolean) -> Unit): SharedPreferences.OnSharedPreferenceChangeListener {
        val listener = SharedPreferences.OnSharedPreferenceChangeListener { _, key -> if (key == KEY) onChange(isList()) }
        preferences.registerOnSharedPreferenceChangeListener(listener)
        return listener
    }

    private fun legacyChoices(): List<String?> {
        fun old(file: String) = app.getSharedPreferences(file, Context.MODE_PRIVATE)
        val video = old("video_browse")
        return listOf(
            old("library_navigation").getString("series_layout", null),
            old("library_navigation").getString("books_layout", null),
            old("music_view").getString("layout", null),
            old("podcast_view").getString("layout", null),
        ) + video.all.keys.filter { it.startsWith("layout_") }.sorted().map { video.getString(it, null) }
    }

    private companion object {
        const val FILE = "browse_layout"
        const val KEY = "layout"
        const val LIST = "List"
        const val COVERS = "Covers"
    }
}
