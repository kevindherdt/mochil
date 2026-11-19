package net.kdhmedia.mochil.ui

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.staticCompositionLocalOf
import net.kdhmedia.mochil.catalogue.LibrarySummary

/** The order of libraries in the TV rail, chosen in Settings; libraries it has never seen keep the server's order after the chosen ones. */
internal class TvLibraryOrder(context: Context) {
    private val preferences = context.applicationContext.getSharedPreferences("tv_library_order", Context.MODE_PRIVATE)
    private val ids = mutableStateOf(preferences.getString(KEY, "").orEmpty().split(',').filter(String::isNotBlank))

    fun sorted(libraries: List<LibrarySummary>): List<LibrarySummary> {
        val rank = ids.value.withIndex().associate { it.value to it.index }
        return libraries.sortedBy { rank[it.id] ?: Int.MAX_VALUE }
    }

    /** Moves [id] one place up (-1) or down (+1) among [libraries] as they are shown now. */
    fun move(id: String, delta: Int, libraries: List<LibrarySummary>) {
        val shown = sorted(libraries).map(LibrarySummary::id).toMutableList()
        val from = shown.indexOf(id)
        val to = from + delta
        if (from < 0 || to !in shown.indices) return
        shown.add(to, shown.removeAt(from))
        ids.value = shown
        preferences.edit().putString(KEY, shown.joinToString(",")).apply()
    }

    private companion object { const val KEY = "ids" }
}

internal val LocalTvLibraryOrder = staticCompositionLocalOf<TvLibraryOrder?> { null }

/** The libraries in the order this TV shows them. */
@Composable
internal fun List<LibrarySummary>.inTvOrder(): List<LibrarySummary> = LocalTvLibraryOrder.current?.sorted(this) ?: this
