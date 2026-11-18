package net.kdhmedia.mochil.ui

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/**
 * Where the main app should go next, asked for by a screen that lives in its own activity (the chat). The shell reads it
 * when it is shown again, moves there and clears it. Process-wide on purpose: it only lives until the shell consumes it.
 */
object PendingShellOpen {
    const val HOME = "home"
    const val SETTINGS = "settings"
    private const val LIBRARY = "library:"

    var target by mutableStateOf<String?>(null)

    fun library(id: String) = LIBRARY + id
    fun libraryId(target: String): String? = target.takeIf { it.startsWith(LIBRARY) }?.removePrefix(LIBRARY)
}
