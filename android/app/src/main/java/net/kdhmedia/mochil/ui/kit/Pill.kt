package net.kdhmedia.mochil.ui.kit

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.gestures.ScrollableState
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.CompositingStrategy
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.ui.theme.MochilRadius
import net.kdhmedia.mochil.ui.theme.MochilSpace

/** One choice in a [PillRow] or [ChipRow]; the glyph is the library's or the filter's own, and optional. */
internal data class PillItem(val id: String, val label: String, val icon: ImageVector? = null)

/**
 * The web's PillRow (web/src/ui/PillRow.module.css): a row of single-choice pills that scrolls sideways when it does
 * not fit. The idle pill is the surface colour with a hairline; the active one is the accent fill with no border, and
 * its label keeps the same weight as the others so nothing shouts.
 *
 * Like the web's, the row brings the chosen pill into view (a library far along it must not open with the tick out of
 * sight) and fades the edge that has more behind it, so a row that overflows reads as scrollable and one that fits
 * looks finished.
 */
@Composable
internal fun PillRow(items: List<PillItem>, activeId: String?, onSelect: (String) -> Unit, modifier: Modifier = Modifier, role: Role = Role.RadioButton) {
    val state = rememberLazyListState()
    val still = LocalReduceMotion.current
    val fade = MochilSpace.s48
    val fadePx = with(LocalDensity.current) { fade.toPx() }.toInt()
    val activeIndex = items.indexOfFirst { it.id == activeId }
    LaunchedEffect(activeId, items.size, fadePx) {
        // Brought in clear of the fade: scrolled to the very start, the chosen pill would sit under the gradient.
        if (activeIndex >= 0 && !state.isFullyVisible(activeIndex, fadePx)) state.scrollTo(activeIndex, still, -fadePx)
    }
    LazyRow(
        modifier.fillMaxWidth().fadeEdges(state, fade).selectableGroup(),
        state = state,
        horizontalArrangement = Arrangement.spacedBy(MochilSpace.s8),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        itemsIndexed(items, key = { _, item -> item.id }) { _, item ->
            Pill(item.label, selected = item.id == activeId, onClick = { onSelect(item.id) }, role = role, icon = item.icon)
        }
    }
}

/** Visible with its whole body clear of the fade zones at both ends. */
private fun LazyListState.isFullyVisible(index: Int, fadePx: Int): Boolean {
    val info = layoutInfo.visibleItemsInfo.firstOrNull { it.index == index } ?: return false
    return info.offset >= layoutInfo.viewportStartOffset + fadePx && info.offset + info.size <= layoutInfo.viewportEndOffset - fadePx
}

/** [scrollTo] with the fade kept clear of the item; the list clamps at the start, so a pill near index 0 simply stays put. */
private suspend fun LazyListState.scrollTo(index: Int, still: Boolean, offsetPx: Int) {
    if (still) scrollToItem(index, offsetPx) else animateScrollToItem(index, offsetPx)
}

/** Fades the end of a scrolling row that has more behind it (web `.fading`): both ends once it is scrolled away from the start. A lazy row and a plain scroll both qualify. */
internal fun Modifier.fadeEdges(state: ScrollableState, width: Dp): Modifier = this
    .graphicsLayer { compositingStrategy = CompositingStrategy.Offscreen }
    .drawWithContent {
        drawContent()
        val fade = width.toPx().coerceAtMost(size.width / 2)
        if (state.canScrollForward) {
            drawRect(
                Brush.horizontalGradient(listOf(Color.Black, Color.Transparent), startX = size.width - fade, endX = size.width),
                topLeft = Offset(size.width - fade, 0f), size = Size(fade, size.height), blendMode = BlendMode.DstIn,
            )
        }
        if (state.canScrollBackward) {
            drawRect(
                Brush.horizontalGradient(listOf(Color.Transparent, Color.Black), startX = 0f, endX = fade),
                topLeft = Offset.Zero, size = Size(fade, size.height), blendMode = BlendMode.DstIn,
            )
        }
    }

/**
 * Drawn at the token height (24) so a row of them is as slim as the mock-up, but the touch target is a full 44 by 44: the
 * outer box takes the tap and the pill sits centred in it, so a short label such as "Dark" is not a narrow target. A minimum, not a fixed height, so large text grows the pill.
 */
@Composable
internal fun Pill(label: String, selected: Boolean, onClick: () -> Unit, modifier: Modifier = Modifier, role: Role = Role.RadioButton, icon: ImageVector? = null) {
    val scheme = MaterialTheme.colorScheme
    val shape = RoundedCornerShape(MochilRadius.pill)
    Box(
        modifier.heightIn(min = MochilChrome.touchMin).widthIn(min = MochilChrome.touchMin).selectable(selected = selected, role = role, onClick = onClick),
        // Start-aligned: a label shorter than the touch target extends the target to its right instead of indenting the pill, so a row lines up at the left edge.
        contentAlignment = Alignment.CenterStart,
    ) {
        Box(
            Modifier
                .heightIn(min = MochilChrome.pillHeight)
                .clip(shape)
                .background(if (selected) scheme.primary else scheme.surface)
                .then(if (selected) Modifier else Modifier.border(MochilChrome.hairline, scheme.outlineVariant, shape))
                .padding(horizontal = MochilSpace.s8),
            contentAlignment = Alignment.Center,
        ) {
            val ink = if (selected) scheme.onPrimary else scheme.onSurface
            Row(horizontalArrangement = Arrangement.spacedBy(MochilSpace.s4), verticalAlignment = Alignment.CenterVertically) {
                if (icon != null) Icon(icon, contentDescription = null, Modifier.size(MochilChrome.pillGlyph * LocalDensity.current.fontScale), tint = ink)
                Text(label, style = MaterialTheme.typography.labelSmall, color = ink, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        }
    }
}
