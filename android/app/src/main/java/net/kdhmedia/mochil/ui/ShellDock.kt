package net.kdhmedia.mochil.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.hermes.HermesChatEntry
import net.kdhmedia.mochil.ui.kit.Dock

/**
 * The dock as the phone shell uses it: the Dock itself plus the one thing it does not know, the navigation bar. The surface
 * colour runs under the bar so gesture navigation and three-button navigation both sit on it rather than on a seam.
 * [chat] is null for an account that cannot reach the chat, which leaves a four place dock.
 */
@Composable
internal fun ShellDock(
    destination: MobileShellDestination,
    selectedLibrary: LibrarySummary?,
    chat: HermesChatEntry?,
    onDestination: (MobileShellDestination) -> Unit,
) {
    Box(Modifier.background(MaterialTheme.colorScheme.surface).windowInsetsPadding(WindowInsets.navigationBars)) {
        Dock(dockPlaces(chatAvailable = chat != null), dockActiveId(destination, selectedLibrary), onSelect = { place ->
            if (place.id == DockIds.AI) chat?.open?.invoke() else dockDestination(place.id)?.let(onDestination)
        })
    }
}
