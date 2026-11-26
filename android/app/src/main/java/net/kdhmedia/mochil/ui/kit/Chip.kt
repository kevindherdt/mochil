package net.kdhmedia.mochil.ui.kit

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextOverflow
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilRadius
import net.kdhmedia.mochil.ui.theme.MochilSpace

/**
 * The web's FilterChips (web/src/ui/FilterChips.module.css): the status filter under the library pills, `All · In Progress ·
 * Unread`. A filter, not navigation, so it is text only and a little taller than a pill. It scrolls sideways when it does
 * not fit, which is what happens at a large font.
 */
@Composable
internal fun ChipRow(items: List<PillItem>, activeId: String?, onSelect: (String) -> Unit, modifier: Modifier = Modifier) {
    val scroll = rememberScrollState()
    Row(
        modifier.fadeEdges(scroll, MochilSpace.s32).horizontalScroll(scroll).selectableGroup(),
        horizontalArrangement = Arrangement.spacedBy(MochilSpace.s8),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        items.forEach { item -> Chip(item.label, selected = item.id == activeId, onClick = { onSelect(item.id) }) }
    }
}

/**
 * Drawn at the token height (28) and a full 44 by 44 to touch, like [Pill]: the outer box takes the tap, the chip sits centred
 * in it. The idle chip is a hairline over the page with muted text; the chosen one is the accent fill with no border.
 */
@Composable
internal fun Chip(label: String, selected: Boolean, onClick: () -> Unit, modifier: Modifier = Modifier, role: Role = Role.Tab) {
    val scheme = MaterialTheme.colorScheme
    val shape = RoundedCornerShape(MochilRadius.pill)
    Box(
        modifier.heightIn(min = MochilChrome.touchMin).widthIn(min = MochilChrome.touchMin).selectable(selected = selected, role = role, onClick = onClick),
        contentAlignment = Alignment.CenterStart,
    ) {
        Box(
            Modifier
                .heightIn(min = MochilChrome.chipHeight)
                .clip(shape)
                .then(if (selected) Modifier.background(scheme.primary) else Modifier.border(MochilChrome.hairline, scheme.outlineVariant, shape))
                .padding(horizontal = MochilSpace.s12),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                label,
                style = MaterialTheme.typography.labelMedium,
                color = if (selected) scheme.onPrimary else scheme.onSurfaceVariant,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}
