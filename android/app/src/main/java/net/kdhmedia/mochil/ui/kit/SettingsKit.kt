package net.kdhmedia.mochil.ui.kit

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.material3.IconButton
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilSpace

/** A theme to choose: [fill] is the page colour it paints, [accent] the colour of its dot. Real values, from the palette. */
internal data class SwatchItem(val id: String, val label: String, val fill: Color, val accent: Color)

/**
 * Round previews of the themes with their names under them (mock-up sheet 6: "four circular swatches"). The page colour
 * is the circle and the accent is the dot, so two palettes with similar backgrounds still tell apart. The chosen one
 * wears an accent ring; the rest a hairline. A name in words is always there, so colour is never the only signal.
 */
@Composable
internal fun SwatchRow(items: List<SwatchItem>, selectedId: String?, onSelect: (String) -> Unit, modifier: Modifier = Modifier) {
    Row(modifier.fillMaxWidth().selectableGroup()) {
        items.forEach { item -> Swatch(item, selected = item.id == selectedId, onClick = { onSelect(item.id) }, modifier = Modifier.weight(1f)) }
    }
}

@Composable
private fun Swatch(item: SwatchItem, selected: Boolean, onClick: () -> Unit, modifier: Modifier) {
    val scheme = MaterialTheme.colorScheme
    Column(
        modifier.heightIn(min = MochilChrome.touchMin).selectable(selected = selected, role = Role.RadioButton, onClick = onClick),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(MochilSpace.s4),
    ) {
        Box(
            Modifier
                .size(MochilSpace.s32)
                .clip(CircleShape)
                .background(item.fill)
                .border(if (selected) MochilChrome.ring else MochilChrome.hairline, if (selected) scheme.primary else scheme.outlineVariant, CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            Box(Modifier.size(MochilSpace.s12).clip(CircleShape).background(item.accent))
        }
        Text(
            item.label,
            style = MaterialTheme.typography.labelSmall,
            color = if (selected) scheme.onSurface else scheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            // Two lines: "Magenta Cream" is the longest name and a swatch is a fifth of the width, so at a large font it wraps.
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/**
 * One row of the Settings hub (web/src/features/settings/screens/SettingsScreen.tsx): glyph, name, one line of what is
 * inside, chevron. The whole card is the button. The description wraps instead of clipping at large font sizes.
 */
@Composable
internal fun SettingsEntry(icon: ImageVector, label: String, description: String, onClick: () -> Unit, modifier: Modifier = Modifier) {
    val scheme = MaterialTheme.colorScheme
    MochilCard(modifier, onClick) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(MochilSpace.s12)) {
            Icon(icon, contentDescription = null, Modifier.size(MochilChrome.iconRow), tint = scheme.onSurfaceVariant)
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(MochilSpace.s4)) {
                Text(label, style = MaterialTheme.typography.bodyMedium, color = scheme.onSurface)
                Text(description, style = MaterialTheme.typography.bodySmall, color = scheme.onSurfaceVariant)
            }
            Icon(LucideIcons.ChevronRight, contentDescription = null, Modifier.size(MochilChrome.iconSmall), tint = scheme.onSurfaceVariant)
        }
    }
}

/** A sub-page's top line: the way back, then the page name as a real heading for TalkBack. */
@Composable
internal fun PageHeader(title: String, onBack: () -> Unit, modifier: Modifier = Modifier) {
    Row(modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(MochilSpace.s4)) {
        IconButton(onClick = onBack) { Icon(LucideIcons.ArrowLeft, contentDescription = "Back", Modifier.size(MochilChrome.iconRow), tint = MaterialTheme.colorScheme.onSurface) }
        Text(title, Modifier.semantics { heading() }, style = MaterialTheme.typography.headlineSmall, color = MaterialTheme.colorScheme.onSurface)
    }
}
