package net.kdhmedia.mochil.ui.kit

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Density
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilFontSize
import net.kdhmedia.mochil.ui.theme.MochilSpace

/**
 * Five labels share the width, so the label stops growing at this system font scale. Past it "Settings" no longer fits
 * its fifth of a phone and was cut to "Sett…"; the glyph, the touch target and TalkBack still carry the place.
 */
private const val MAX_LABEL_FONT_SCALE = 1.3f

/**
 * One place in the dock. [id] is what the shell routes on; [label] is what is shown and what TalkBack reads. A [launcher]
 * opens a screen of its own (the chat) instead of becoming the current place, so it is announced as a button, not a tab.
 */
internal data class DockPlace(val id: String, val label: String, val icon: ImageVector, val launcher: Boolean = false)

/**
 * The phone's navigation: icon over label, flat, on the surface colour with no top border (web/src/shell/Dock.tsx).
 * The active place is two-toned like the web's: accent glyph, full-strength label. Everything else is muted.
 *
 * The bar is a minimum height, never a fixed one, so a large system font grows it instead of cutting the labels off. It
 * does not know about the navigation bar or the keyboard; the shell that hosts it handles both.
 */
@Composable
internal fun Dock(places: List<DockPlace>, activeId: String?, onSelect: (DockPlace) -> Unit, modifier: Modifier = Modifier) {
    Row(
        modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.surface)
            .height(IntrinsicSize.Min)
            .heightIn(min = MochilChrome.dockHeightPhone)
            .selectableGroup(),
    ) {
        places.forEach { place ->
            DockItem(place, selected = place.id == activeId, onClick = { onSelect(place) }, modifier = Modifier.weight(1f))
        }
    }
}

@Composable
private fun DockItem(place: DockPlace, selected: Boolean, onClick: () -> Unit, modifier: Modifier) {
    val scheme = MaterialTheme.colorScheme
    Column(
        modifier
            .fillMaxHeight()
            .heightIn(min = MochilChrome.touchMin)
            .then(
                if (place.launcher) Modifier.clickable(role = Role.Button, onClick = onClick)
                else Modifier.selectable(selected = selected, role = Role.Tab, onClick = onClick),
            ),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(MochilSpace.s4, Alignment.CenterVertically),
    ) {
        Icon(place.icon, contentDescription = null, Modifier.size(MochilChrome.dockGlyphPhone), tint = if (selected) scheme.primary else scheme.onSurfaceVariant)
        val density = LocalDensity.current
        CompositionLocalProvider(LocalDensity provides Density(density.density, minOf(density.fontScale, MAX_LABEL_FONT_SCALE))) {
            Text(
                place.label,
                style = MaterialTheme.typography.bodySmall.copy(fontSize = MochilFontSize.s11),
                color = if (selected) scheme.onSurface else scheme.onSurfaceVariant,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}
