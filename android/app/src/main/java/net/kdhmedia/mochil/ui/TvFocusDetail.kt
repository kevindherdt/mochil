package net.kdhmedia.mochil.ui

import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

internal data class TvFocusDetail(
    val title: String,
    val metadata: String,
    val description: String = "",
    val navigationBucket: String = tvNavigationBucket(title),
)

@Stable
internal class TvFocusDetailState {
    var value by mutableStateOf<TvFocusDetail?>(null)
        private set

    fun show(detail: TvFocusDetail) { value = detail }
    fun clear() { value = null }
}

internal fun tvFocusDescription(value: String, maximumLength: Int = 280): String = value
    .replace(Regex("\\s+"), " ")
    .trim()
    .take(maximumLength)

/** Matches the catalogue's compact A-Z navigation without locale-sensitive surprises. */
internal fun tvNavigationBucket(value: String): String {
    val normalized = value.trim().lowercase(java.util.Locale.ROOT)
    val article = listOf("the ", "an ", "a ").firstOrNull(normalized::startsWith)
    val sorted = if (article == null) normalized else normalized.removePrefix(article).trim()
    return sorted.firstOrNull()?.uppercaseChar()?.takeIf { it in 'A'..'Z' }?.toString() ?: "#"
}
