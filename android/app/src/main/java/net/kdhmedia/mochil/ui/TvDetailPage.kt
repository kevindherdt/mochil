package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.runtime.Composable
import androidx.compose.ui.layout.Layout
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/** Keep Cast atomic at the fold: show its heading AND portraits when they fit, otherwise
 * start the whole section after the first viewport. Focus scrolling can then reveal it.
 * Both decisions use measured children, including real text and the parent's dock inset.
 */
@Composable
internal fun TvDetailPage(
    viewportHeight: Dp,
    hero: @Composable () -> Unit,
    cast: @Composable () -> Unit,
) {
    Layout(content = { Box { hero() }; Box { cast() } }) { children, constraints ->
        val viewport = viewportHeight.roundToPx()
        val loose = constraints.copy(minWidth = 0, minHeight = 0)
        val header = children[0].measure(loose.copy(maxHeight = viewport))
        val actors = children[1].measure(loose)
        val next = header.height + 12.dp.roundToPx()
        val castTop = if (next + actors.height <= viewport) next else viewport
        layout(constraints.maxWidth, maxOf(viewport, castTop + actors.height)) {
            header.placeRelative(0, 0)
            actors.placeRelative(0, castTop)
        }
    }
}
