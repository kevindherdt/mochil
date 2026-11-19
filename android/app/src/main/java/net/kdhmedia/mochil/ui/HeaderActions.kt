package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp

/** One icon button for the row beside the library switcher. Plain data, so the header always draws the newest copy. */
internal data class HeaderAction(
    val icon: ImageVector,
    val description: String,
    val onClick: () -> Unit,
    val tint: Color = Color.Unspecified,
    val enabled: Boolean = true,
)

/** Where a library screen hands its search and layout icons up to the row that holds the library switcher; null when there is no such row. */
internal val LocalHeaderActions = staticCompositionLocalOf<((List<HeaderAction>?) -> Unit)?> { null }

@Composable
internal fun RowScope.HeaderActionButtons(actions: List<HeaderAction>) {
    actions.forEach { action ->
        IconButton(onClick = action.onClick, enabled = action.enabled) { Icon(action.icon, action.description, tint = action.tint) }
    }
}

/**
 * Icons that belong at the right of the switcher row, so they cost no row of their own and the covers start higher.
 * Where the switcher row is not shown (wide layouts) they stay in a thin row of their own.
 *
 * The actions travel as data rather than as a composable lambda: a lambda handed to another part of the screen kept drawing
 * an old copy, so a layout button showed the wrong icon and its tap repeated the old choice.
 */
@Composable
internal fun HeaderActions(actions: List<HeaderAction>) {
    val slot = LocalHeaderActions.current
    if (slot == null) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp), horizontalArrangement = Arrangement.End, verticalAlignment = Alignment.CenterVertically) { HeaderActionButtons(actions) }
        return
    }
    SideEffect { slot(actions) }
    DisposableEffect(slot) { onDispose { slot(null) } }
}
