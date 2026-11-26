package net.kdhmedia.mochil.catalogue

import android.content.Context
import android.content.SharedPreferences
import org.json.JSONArray
import org.json.JSONObject

/** Owns durable publication navigation and the last-known library directory. */
internal class LibraryNavigationStore(context: Context) {
    private val preferences = context.getSharedPreferences("library_navigation", Context.MODE_PRIVATE)

    /** The saved view, or the device's own when none is saved: a phone opens on the plain list of books (where the status chips are), the television on series. */
    fun grouping(device: ClientDeviceClass): PublicationGrouping =
        preferences.enumValue("grouping", device.defaultGrouping()).startingView(device.defaultGrouping())
    fun sort(): String = preferences.getString("sort", "")?.takeIf { it in setOf("", "author", "series", "recent") } ?: ""
    fun setSort(sort: String) { preferences.edit().putString("sort", sort).apply() }
    private val layouts = BrowseLayoutStore(context)
    fun seriesLayout(): PublicationLayout = layout()
    fun booksLayout(): PublicationLayout = layout()
    private fun layout() = if (layouts.isList()) PublicationLayout.List else PublicationLayout.Covers
    fun listenLayout(onChange: (PublicationLayout) -> Unit) = layouts.listen { onChange(if (it) PublicationLayout.List else PublicationLayout.Covers) }

    fun selected(libraries: List<LibrarySummary>): LibrarySummary? =
        libraries.firstOrNull { it.id == preferences.getString("library", null) }
            ?: libraries.firstOrNull { it.isPublicationLibrary() }
            ?: libraries.firstOrNull()

    fun select(libraryId: String) {
        preferences.edit().putString("library", libraryId).apply()
    }

    fun setGrouping(grouping: PublicationGrouping) {
        preferences.edit().putString("grouping", grouping.name).apply()
    }

    fun setLayout(grouping: PublicationGrouping, layout: PublicationLayout) {
        layouts.setList(layout == PublicationLayout.List)
    }

    fun saveLibraries(libraries: List<LibrarySummary>) {
        preferences.edit().putString("libraries", LibraryDirectoryCodec.encode(libraries)).apply()
    }

    fun cachedLibraries(): List<LibrarySummary> = LibraryDirectoryCodec.decode(preferences.getString("libraries", "[]"))
}

internal object LibraryDirectoryCodec {
    fun encode(libraries: List<LibrarySummary>): String = JSONArray().apply {
        libraries.forEach { library ->
            put(JSONObject().put("id", library.id).put("name", library.name).put("kind", library.kind).put("count", library.count))
        }
    }.toString()

    fun decode(raw: String?): List<LibrarySummary> = runCatching {
        val values = JSONArray(raw ?: "[]")
        List(values.length()) { index ->
            values.getJSONObject(index).run { LibrarySummary(getString("id"), getString("name"), optString("kind"), optInt("count")) }
        }.filter { it.id.isNotBlank() && it.name.isNotBlank() }
    }.getOrDefault(emptyList())
}

private inline fun <reified T : Enum<T>> SharedPreferences.enumValue(key: String, fallback: T): T =
    runCatching { enumValueOf<T>(getString(key, fallback.name) ?: fallback.name) }.getOrDefault(fallback)

/**
 * The view a library opens on. "In progress" and "Collections" are short trips, not places to live: remembering them made a library
 * open on a handful of books until another view was picked, so they fall back to the series view.
 */
internal fun PublicationGrouping.startingView(fallback: PublicationGrouping = PublicationGrouping.Series): PublicationGrouping = when (this) {
    PublicationGrouping.Continue, PublicationGrouping.Collections -> fallback
    else -> this
}

internal fun ClientDeviceClass.defaultGrouping(): PublicationGrouping =
    if (this == ClientDeviceClass.Television) PublicationGrouping.Series else PublicationGrouping.Books
